package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalFile(fileStmt *ast.FileStatement, env *Environment) (Value, error) {
	// 1. Interpolation des variables dans le chemin du fichier
	rawPath := fileStmt.Path
	for k, v := range env.All() {
		rawPath = strings.ReplaceAll(rawPath, fmt.Sprintf("{%s}", k), v.String())
	}

	targetPath := rawPath
	if !filepath.IsAbs(targetPath) && env.CurrentDir() != "" {
		targetPath = filepath.Join(env.CurrentDir(), targetPath)
	}

	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return NilValue{}, fmt.Errorf("failed to open file '%s': %w", targetPath, err)
	}
	content := string(contentBytes)
	modified := false

	// 2. Création de l'environnement confiné au fichier avec les opérations chirurgicales
	fileEnv := NewEnvironment(env)
	fileEnv.SetCurrentDir(filepath.Dir(targetPath))

	// Replace(target, replacement)
	fileEnv.Set("Replace", FunctionValue{
		Name: "Replace",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("Replace requires target string and replacement")
			}
			target := args[0].String()
			repl := args[1].String()
			if strings.Contains(content, target) {
				content = strings.ReplaceAll(content, target, repl)
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Remove(target)
	fileEnv.Set("Remove", FunctionValue{
		Name: "Remove",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Remove requires target string")
			}
			target := args[0].String()
			if strings.Contains(content, target) {
				content = strings.ReplaceAll(content, target, "")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveBlock(startPattern, endPattern)
	fileEnv.Set("RemoveBlock", FunctionValue{
		Name: "RemoveBlock",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("RemoveBlock requires start and end patterns")
			}
			startPat := args[0].String()
			endPat := args[1].String()

			sIdx := strings.Index(content, startPat)
			if sIdx != -1 {
				lineStart := strings.LastIndex(content[:sIdx], "\n")
				if lineStart == -1 {
					lineStart = 0
				} else {
					lineStart += 1
				}

				eRel := strings.Index(content[sIdx:], endPat)
				if eRel != -1 {
					eIdx := sIdx + eRel + len(endPat)
					lineEnd := strings.Index(content[eIdx:], "\n")
					if lineEnd == -1 {
						eIdx = len(content)
					} else {
						eIdx = eIdx + lineEnd + 1
					}
					content = content[:lineStart] + content[eIdx:]
					modified = true
				}
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveUntil(startPattern, untilPattern)
	fileEnv.Set("RemoveUntil", FunctionValue{
		Name: "RemoveUntil",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("RemoveUntil requires start pattern and until pattern")
			}
			startPat := args[0].String()
			untilPat := args[1].String()

			sIdx := strings.Index(content, startPat)
			if sIdx != -1 {
				lineStart := strings.LastIndex(content[:sIdx], "\n")
				if lineStart == -1 {
					lineStart = 0
				} else {
					lineStart += 1
				}

				uRel := strings.Index(content[sIdx:], untilPat)
				if uRel != -1 {
					untilIdx := sIdx + uRel
					lineUntil := strings.LastIndex(content[:untilIdx], "\n")
					if lineUntil == -1 || lineUntil < lineStart {
						lineUntil = untilIdx
					} else {
						lineUntil += 1
					}
					content = content[:lineStart] + content[lineUntil:]
					modified = true
				}
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveLines(pattern)
	fileEnv.Set("RemoveLines", FunctionValue{
		Name: "RemoveLines",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("RemoveLines requires pattern string")
			}
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			var kept []string
			changed := false
			for _, line := range lines {
				if strings.Contains(line, pattern) {
					changed = true
					continue
				}
				kept = append(kept, line)
			}
			if changed {
				content = strings.Join(kept, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// ReplaceLine(pattern, replacement)
	fileEnv.Set("ReplaceLine", FunctionValue{
		Name: "ReplaceLine",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("ReplaceLine requires pattern string and replacement")
			}
			pattern := args[0].String()
			repl := args[1].String()
			lines := strings.Split(content, "\n")
			changed := false
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					lines[i] = repl
					changed = true
				}
			}
			if changed {
				content = strings.Join(lines, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// CommentOut(pattern)
	fileEnv.Set("CommentOut", FunctionValue{
		Name: "CommentOut",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("CommentOut requires pattern string")
			}
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			changed := false
			for i, line := range lines {
				if strings.Contains(line, pattern) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
					lines[i] = "// " + line
					changed = true
				}
			}
			if changed {
				content = strings.Join(lines, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Append(text)
	fileEnv.Set("Append", FunctionValue{
		Name: "Append",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Append requires string")
			}
			content += "\n" + args[0].String()
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Prepend(text)
	fileEnv.Set("Prepend", FunctionValue{
		Name: "Prepend",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Prepend requires string")
			}
			content = args[0].String() + "\n" + content
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// JsonSet(keyPath, val)
	fileEnv.Set("JsonSet", FunctionValue{
		Name: "JsonSet",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("JsonSet requires key path and value")
			}
			keyPath := args[0].String()
			val := args[1].String()

			var data map[string]any
			if err := json.Unmarshal([]byte(content), &data); err != nil {
				return NilValue{}, fmt.Errorf("invalid json in %s: %w", targetPath, err)
			}
			keys := strings.Split(keyPath, ".")
			setNestedJson(data, keys, val)

			updatedBytes, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return NilValue{}, err
			}
			content = string(updatedBytes) + "\n"
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// JsonRemove(key)
	fileEnv.Set("JsonRemove", FunctionValue{
		Name: "JsonRemove",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("JsonRemove requires key name")
			}
			key := args[0].String()

			var data map[string]any
			if err := json.Unmarshal([]byte(content), &data); err != nil {
				return NilValue{}, fmt.Errorf("invalid json in %s: %w", targetPath, err)
			}
			delete(data, key)

			updatedBytes, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return NilValue{}, err
			}
			content = string(updatedBytes) + "\n"
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// 3. Exécution de toutes les instructions dans le scope du fichier
	for _, s := range fileStmt.Body {
		if _, err := ev.Eval(s, fileEnv); err != nil {
			return NilValue{}, err
		}
	}

	// 4. Écriture si le fichier a été modifié
	if modified {
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return NilValue{}, fmt.Errorf("failed to write file '%s': %w", targetPath, err)
		}
	}

	return BoolValue{Val: true}, nil
}

func setNestedJson(data map[string]any, keys []string, val any) {
	if len(keys) == 1 {
		data[keys[0]] = val
		return
	}
	currentKey := keys[0]
	subMap, exists := data[currentKey].(map[string]any)
	if !exists {
		subMap = make(map[string]any)
		data[currentKey] = subMap
	}
	setNestedJson(subMap, keys[1:], val)
}
