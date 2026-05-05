package engine

import (
	"strings"
	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/engine/patcher"
	"github.com/Nehonix-Team/xru/internal/engine/util" 
	"github.com/Nehonix-Team/xru/internal/engine/parser"
)

// applyAction applique une action (inject, patch, var, module) au contenu d'un fichier.
func applyAction(content string, action ast.Action, fileExt string, scope *Scope, cb string, rulePath string, r *Runner) string {
	switch a := action.(type) {
	case ast.VarAction:
		val := util.Interpolate(a.Value, scope)
		scope.Set(a.Name, unescape(val), a.Line)
		return content

	case ast.ModuleAction:
		executeModuleAction(scope, cb, rulePath, a.Module, a.Method, a.Target, a.As, a.Line, r)
		return content

	case ast.InjectAction:
		if a.Lang != "" && "."+strings.ToLower(a.Lang) != fileExt {
			return content
		}
		code := processInception(a.Code, scope, r)
		checkSyntaxError(code, a.Line, r)
		return engine.InjectCode(content, a.Key, code, a.Raw)

	case ast.PatchAction:
		path := util.Interpolate(a.Path, scope)
		val := util.InterpolateValue(a.Value, scope).(ast.Value)
		// Si la valeur est un Literal (string), on lui applique aussi l'inception
		if lit, ok := val.(ast.Literal); ok {
			val = ast.Literal(processInception(string(lit), scope, r))
		}
		return patcher.ApplyPatch(content, a.Op, path, val)
	}
	return content
}

func processInception(src string, scope *Scope, r *Runner) string {
	if !strings.Contains(src, "<#") {
		return util.Interpolate(src, scope)
	}

	virtualScript := ""
	remaining := src
	
	for {
		startIdx := strings.Index(remaining, "<#")
		if startIdx == -1 {
			break
		}

		// Check if tag is on a line by itself (only preceded by whitespace since last newline)
		isLineStart := true
		for i := startIdx - 1; i >= 0; i-- {
			if remaining[i] == '\n' {
				break
			}
			if remaining[i] != ' ' && remaining[i] != '\t' {
				isLineStart = false
				break
			}
		}

		// Text before tag
		text := remaining[:startIdx]
		if isLineStart {
			lastNL := strings.LastIndex(text, "\n")
			if lastNL == -1 {
				text = ""
			} else {
				text = text[:lastNL+1]
			}
		}

		if text != "" {
			lines := strings.Split(text, "\n")
			for i, l := range lines {
				if i == len(lines)-1 && l == "" {
					break
				}
				virtualScript += "#LOG: " + l + "\n"
			}
		}

		// Find end of tag
		remaining = remaining[startIdx+2:]
		endIdx := strings.Index(remaining, ">")
		if endIdx == -1 {
			remaining = "<#" + remaining
			break
		}

		directive := strings.TrimSpace(remaining[:endIdx])
		virtualScript += "#" + directive + "\n"
		remaining = remaining[endIdx+1:]

		// If it was on a line by itself, consume the trailing newline
		if isLineStart {
			trimLen := 0
			for i := 0; i < len(remaining); i++ {
				if remaining[i] == ' ' || remaining[i] == '\t' {
					trimLen++
				} else if remaining[i] == '\r' && i+1 < len(remaining) && remaining[i+1] == '\n' {
					trimLen += 2
					break
				} else if remaining[i] == '\n' {
					trimLen++
					break
				} else {
					break
				}
			}
			remaining = remaining[trimLen:]
		}
	}

	if remaining != "" {
		lines := strings.Split(remaining, "\n")
		for i, l := range lines {
			if i == len(lines)-1 && l == "" && len(lines) > 1 {
				break
			}
			virtualScript += "#LOG: " + l + "\n"
		}
	}

	rf, err := parser.Parse(virtualScript)
	if err != nil {
		return "[INCEPTION_ERROR: " + err.Error() + "]"
	}
	
	capture := &strings.Builder{}
	childScope := &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  scope.Modules,
		Parent:   scope,
		Capture:  capture,
		Runner:   r,
	}

	executeRules(rf.Rules, "", "", "", childScope, nil, "", r)
	
	return strings.TrimSuffix(capture.String(), "\n")
}

