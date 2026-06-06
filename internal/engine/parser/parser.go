package parser

import (
	"fmt"
	"os"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine/util"
)

// Parse converts .xru source text into a RuleFile AST.
func Parse(src string) (*ast.RuleFile, error) {
	return parseNew(src)
}

func parseNew(src string) (*ast.RuleFile, error) {
	lines := strings.Split(src, "\n")
	rf := &ast.RuleFile{}

	var stack []*ast.Rule
	var globalRule *ast.Rule

	type pendingAction struct {
		isInject bool
		lang     string
		key      string
		op       ast.PatchOp
		path     string
		buf      []string
		line     int
		raw      bool
	}
	var pending *pendingAction

	commitPending := func() {
		if pending == nil {
			return
		}
		body := strings.Join(pending.buf, "\n")
		var a ast.Action
		if pending.isInject {
			if !pending.raw {
				body = util.Dedent(body)
			}
			a = ast.InjectAction{Lang: pending.lang, Key: pending.key, Code: body, Raw: pending.raw, Line: pending.line}
		} else {
			val := ParseValue(body)
			// Special handling for Regex patches using the '~~: /re/ -> repl' syntax
			if pending.op == ast.PatchRegex {
				obj := make(ast.Object)
				lines := strings.Split(body, "\n")
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l == "" { continue }
					if strings.Contains(l, " -> ") {
						parts := strings.SplitN(l, " -> ", 2)
						re := strings.TrimSpace(parts[0])
						repl := strings.TrimSpace(parts[1])
						// Trim potential slashes /re/
						if strings.HasPrefix(re, "/") && strings.HasSuffix(re, "/") && len(re) >= 2 {
							re = re[1 : len(re)-1]
						}
						obj[re] = ast.Literal(repl)
					}
				}
				if len(obj) > 0 {
					val = obj
				}
			}
			a = ast.PatchAction{Op: pending.op, Path: pending.path, Value: val, Line: pending.line}
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Actions = append(parent.Actions, a)
		} else {
			if globalRule == nil {
				globalRule = &ast.Rule{Type: ast.RuleTypeGlobal, Line: pending.line}
			}
			globalRule.Actions = append(globalRule.Actions, a)
		}
		pending = nil
	}

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			if pending == nil && !(len(stack) > 0 && (stack[len(stack)-1].Type == ast.RuleTypeCreate || stack[len(stack)-1].Type == ast.RuleTypeVarBlock)) {
				continue
			}
		}

		if strings.HasPrefix(trimmed, "#") {
			directiveLine := "#" + strings.TrimSpace(trimmed[1:])
			
			if strings.HasPrefix(directiveLine, "#BEGIN:") {
				commitPending()
				target, as, _, _ := parseTarget(strings.TrimPrefix(directiveLine, "#BEGIN:"), true)
				rule := &ast.Rule{Type: ast.RuleTypeBegin, Target: target, As: as, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#CREATE:") {
				commitPending()
				target, as, _, raw := parseTarget(strings.TrimPrefix(directiveLine, "#CREATE:"), true)
				rule := &ast.Rule{Type: ast.RuleTypeCreate, Target: target, As: as, Raw: raw, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#IF:") {
				commitPending()
				cond := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#IF:"))
				rule := &ast.Rule{Type: ast.RuleTypeIf, Target: cond, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#ELSEIF:") {
				commitPending()
				if len(stack) > 0 {
					last := stack[len(stack)-1]
					if last.Type == ast.RuleTypeIf || last.Type == ast.RuleTypeElseIf {
						prev := *stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if len(stack) > 0 {
							stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, prev)
						} else {
							rf.Rules = append(rf.Rules, prev)
						}
					}
				}
				cond := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#ELSEIF:"))
				rule := &ast.Rule{Type: ast.RuleTypeElseIf, Target: cond, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#ELSE") {
				commitPending()
				if len(stack) > 0 {
					last := stack[len(stack)-1]
					if last.Type == ast.RuleTypeIf || last.Type == ast.RuleTypeElseIf {
						prev := *stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if len(stack) > 0 {
							stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, prev)
						} else {
							rf.Rules = append(rf.Rules, prev)
						}
					}
				}
				rule := &ast.Rule{Type: ast.RuleTypeElse, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#SELECT:") {
				commitPending()
				target, as, _, _ := parseTarget(strings.TrimPrefix(directiveLine, "#SELECT:"), true)
				rule := ast.Rule{Type: ast.RuleTypeSelect, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#LOG:") {
				commitPending()
				msg := strings.TrimPrefix(directiveLine, "#LOG:")
				rule := ast.Rule{Type: ast.RuleTypeLog, Target: msg, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#USE:") {
				commitPending()
				target, as, _, _ := parseTarget(strings.TrimPrefix(directiveLine, "#USE:"), false)
				rule := ast.Rule{Type: ast.RuleTypeUse, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#ARG:") {
				commitPending()
				target, as, orVal, _ := parseTarget(strings.TrimPrefix(directiveLine, "#ARG:"), false)
				rule := ast.Rule{Type: ast.RuleTypeArg, Target: target, As: as, Or: orVal, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#FOR:") {
				commitPending()
				line := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#FOR:"))
				rule := &ast.Rule{Type: ast.RuleTypeFor, Target: line, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasSuffix(strings.Split(directiveLine, ":")[0], "VAR") {
				commitPending()
				cmdParts := strings.Split(directiveLine, ":")
				cmdName := strings.TrimPrefix(strings.TrimSpace(cmdParts[0]), "#")
				parts := strings.SplitN(directiveLine, ":", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					rule := &ast.Rule{Type: ast.RuleTypeVarBlock, Command: cmdName, Target: name, Line: lineNum}
					stack = append(stack, rule)
					continue
				}
			}
			if strings.HasPrefix(directiveLine, "#INCLUDE:") {
				commitPending()
				target, as, _, _ := parseTarget(strings.TrimPrefix(directiveLine, "#INCLUDE:"), true)
				rule := ast.Rule{Type: ast.RuleTypeInclude, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#CALL:") {
				commitPending()
				target, as, _, _ := parseTarget(strings.TrimPrefix(directiveLine, "#CALL:"), true)
				rule := &ast.Rule{Type: ast.RuleTypeCall, Target: target, As: as, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#GLOBAL:") {
				commitPending()
				target := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#GLOBAL:"))
				rule := ast.Rule{Type: ast.RuleTypeGlobal, Target: target, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#GLOBAL") {
				commitPending()
				rule := &ast.Rule{Type: ast.RuleTypeGlobal, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#END") {
				commitPending()
				if len(stack) > 0 {
					rule := *stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if rule.Type == ast.RuleTypeCreate && !rule.Raw {
						rule.Content = util.Dedent(rule.Content)
					}
					if len(stack) > 0 {
						stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
					} else {
						rf.Rules = append(rf.Rules, rule)
					}
				}
				continue
			}
		}

		if idx := strings.Index(trimmed, "."); idx != -1 {
			colonIdx := strings.Index(trimmed, ":")
			if colonIdx != -1 && colonIdx > idx {
				callPart := strings.TrimSpace(trimmed[:colonIdx])
				if strings.Contains(callPart, ".") && !strings.ContainsAny(callPart, " /\\\"'{}[],") {
					commitPending()
					parts := strings.SplitN(trimmed, ":", 2)
					call := parts[0]
					rest := ""
					if len(parts) > 1 {
						rest = parts[1]
					}
					callParts := strings.SplitN(call, ".", 2)
					module := strings.TrimSpace(callParts[0])
					method := strings.TrimSpace(callParts[1])
					target, as, _, _ := parseTarget(rest, true)
					if len(stack) > 0 {
						stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, ast.Rule{Type: ast.RuleTypeModule, Target: module + "." + method, Content: target, As: as, Line: lineNum})
					} else {
						rf.Rules = append(rf.Rules, ast.Rule{Type: ast.RuleTypeModule, Target: module + "." + method, Content: target, As: as, Line: lineNum})
					}
					continue
				}
			}
		}

		if strings.HasPrefix(trimmed, "@") && strings.Contains(trimmed, "INJECT:") {
			commitPending()
			tag := strings.TrimPrefix(trimmed, "@")
			idx := strings.Index(tag, "INJECT:")
			lang := tag[:idx]
			keyPart := strings.TrimSpace(tag[idx+len("INJECT:"):])
			if (len(keyPart) >= 2 && keyPart[0] == '"' && keyPart[len(keyPart)-1] == '"') ||
				(len(keyPart) >= 2 && keyPart[0] == '\'' && keyPart[len(keyPart)-1] == '\'') {
				keyPart = keyPart[1 : len(keyPart)-1]
			}
			raw := false
			if strings.HasSuffix(keyPart, " --raw") {
				raw = true
				keyPart = strings.TrimSpace(strings.TrimSuffix(keyPart, " --raw"))
			}
			pending = &pendingAction{isInject: true, lang: lang, key: keyPart, raw: raw, line: lineNum}
			continue
		}

		if trimmed == "@END" {
			commitPending()
			pending = nil
			continue
		}

		if name, val, ok := parseVar(trimmed); ok {
			commitPending()
			if len(stack) > 0 {
				last := stack[len(stack)-1]
				if last.Type == ast.RuleTypeBegin || last.Type == ast.RuleTypeCreate {
					last.Actions = append(last.Actions, ast.VarAction{Name: name, Value: val, Line: lineNum})
				} else {
					last.SubRules = append(last.SubRules, ast.Rule{Type: ast.RuleTypeVar, Target: name, Content: val, Line: lineNum})
				}
			} else {
				rf.Rules = append(rf.Rules, ast.Rule{Type: ast.RuleTypeVar, Target: name, Content: val, Line: lineNum})
			}
			continue
		}

		op, path, initial, ok := parseActionLine(trimmed)
		if ok {
			commitPending()
			pending = &pendingAction{isInject: false, op: op, path: path, buf: []string{initial}, line: lineNum}
			continue
		}

		if pending != nil {
			pending.buf = append(pending.buf, line)
			continue
		}

		if trimmed == "" {
			if len(stack) > 0 && (stack[len(stack)-1].Type == ast.RuleTypeCreate || stack[len(stack)-1].Type == ast.RuleTypeVarBlock) {
				stack[len(stack)-1].Content += line + "\n"
			}
			continue
		}

		if len(stack) > 0 && (stack[len(stack)-1].Type == ast.RuleTypeCreate || stack[len(stack)-1].Type == ast.RuleTypeVarBlock) && !strings.HasPrefix(trimmed, "#END") {
			stack[len(stack)-1].Content += line + "\n"
			continue
		}

		return nil, fmt.Errorf("%d: syntax error: unknown directive or action '%s'", lineNum, trimmed)
	}

	commitPending()
	for len(stack) > 0 {
		rule := *stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
		} else {
			rf.Rules = append(rf.Rules, rule)
		}
	}

	if globalRule != nil {
		rf.Rules = append(rf.Rules, *globalRule)
	}

	return rf, nil
}

func ParseFile(path string) (*ast.RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseNew(string(data))
}
