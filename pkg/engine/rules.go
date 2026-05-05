package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine/parser"
	"github.com/Nehonix-Team/xru/internal/engine/util"
	"github.com/Nehonix-Team/xru/internal/engine"
)

func executeActions(rule *ast.Rule, content *string, scope *Scope, fileExt, cb, rulePath string, r *Runner) {
	if content != nil {
		for _, action := range rule.Actions {
			*content = applyAction(*content, action, fileExt, scope, cb, rulePath, r)
		}
	}
}

// executeRules parcourt et exécute une liste de règles dans le scope donné.
func executeRules(rules []ast.Rule, initialTarget, currentBase, rulePath string, scope *Scope, content *string, fileExt string, r *Runner) error {
	cb := currentBase
	skipElse := false

	for _, rule := range rules {
		if r.Verbose {
			fmt.Printf("[DEBUG] Executing rule type: %s at line %d (content context: %v)\n", rule.Type, rule.Line, content != nil)
		}
		target := util.Interpolate(rule.Target, scope)

		switch rule.Type {
		case ast.RuleTypeVar:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			// Puis la logique spécifique
				skipElse = false
				trimmed := strings.TrimSpace(rule.Content)

				// Résolution directe d'une variable objet: {VAR}
				if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
					name := trimmed[1 : len(trimmed)-1]
					for strings.Contains(name, "{") {
						name = util.Interpolate(name, scope)
					}
					if val, ok := scope.Get(name); ok {
						scope.Set(rule.Target, val, rule.Line)
						continue
					}
				}

				val := util.Interpolate(rule.Content, scope)
				if err := checkSyntaxError(val, rule.Line, r); err != nil {
					return err
				}
				scope.Set(rule.Target, unescape(val), rule.Line)

		case ast.RuleTypeVarBlock:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			content := processInception(rule.Content, scope, r)
			content = util.Dedent(content)

			var val interface{} = content
			if strings.HasPrefix(rule.Command, "JSON") {
				var parsed interface{}
				if err := json.Unmarshal([]byte(content), &parsed); err != nil {
					return fmt.Errorf("%s:%d: failed to parse JSON in #%s: %v",
						r.CurrentFile, rule.Line, rule.Command, err)
				}
				val = parsed
			}
			scope.Set(rule.Target, val, rule.Line)
			skipElse = false

		case ast.RuleTypeSelect:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			if err := checkSyntaxError(target, rule.Line, r); err != nil {
				return err
			}
			if filepath.IsAbs(target) {
				cb = target
			} else {
				cb = filepath.Join(initialTarget, target)
			}
			info, err := os.Stat(cb)
			if os.IsNotExist(err) {
				return fmt.Errorf("%s:%d: directory '%s' does not exist",
					r.CurrentFile, rule.Line, cb)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s:%d: path '%s' is a file, but SELECT requires a directory",
					r.CurrentFile, rule.Line, cb)
			}
			if rule.As != "" {
				scope.Set(rule.As, cb, rule.Line)
			}
			skipElse = false

		case ast.RuleTypeIf:
			if evalCondition(rule.Target, scope, cb, r) {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath, r)
					}
				}
				if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt, r); err != nil {
					return err
				}
				skipElse = true
			} else {
				skipElse = false
			}

		case ast.RuleTypeElseIf:
			if !skipElse && evalCondition(rule.Target, scope, cb, r) {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath, r)
					}
				}
				if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt, r); err != nil {
					return err
				}
				skipElse = true
			}

		case ast.RuleTypeElse:
			if !skipElse {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath, r)
					}
				}
				if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt, r); err != nil {
					return err
				}
			}
			skipElse = false

		case ast.RuleTypeUse:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			name := util.Interpolate(rule.Target, scope)
			alias := rule.As
			if alias == "" {
				alias = name
			}
			if err := scope.RegisterModule(alias, name, rule.Line); err != nil {
				return fmt.Errorf("%s:%d: %v", r.CurrentFile, rule.Line, err)
			}
			skipElse = false

		case ast.RuleTypeModule:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			parts := strings.SplitN(rule.Target, ".", 2)
			content := util.Interpolate(rule.Content, scope)
			if err := checkSyntaxError(content, rule.Line, r); err != nil {
				return err
			}
			if err := executeModuleAction(scope, cb, rulePath, parts[0], parts[1], content, rule.As, rule.Line, r); err != nil {
				return err
			}
			skipElse = false

		case ast.RuleTypeInclude:
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := parser.ParseFile(includePath)
			if err == nil {
				if err := executeRules(irf.Rules, initialTarget, cb, includePath, scope, content, fileExt, r); err != nil {
					return err
				}
			}
			skipElse = false

		case ast.RuleTypeCall:
			includePath := target
			if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt, r); err != nil {
				return err
			}
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := parser.ParseFile(includePath)
			if err == nil {
				if err := executeRules(irf.Rules, initialTarget, cb, includePath, scope, content, fileExt, r); err != nil {
					return err
				}
			}
			skipElse = false

		case ast.RuleTypeBegin, ast.RuleTypeCreate, ast.RuleTypeGlobal:
			if err := applyFileRule(initialTarget, cb, rule, scope, r); err != nil {
				return err
			}
			skipElse = false

		case ast.RuleTypeArg:
			executeActions(&rule, content, scope, fileExt, cb, rulePath, r)
			val := getTerminalArg(target, r)
			if rule.As != "" {
				scope.Set(rule.As, val, rule.Line)
			}
			skipElse = false

		case ast.RuleTypeLog:
			msg := util.Interpolate(rule.Target, scope)
			if scope.Capture != nil {
				scope.Capture.WriteString(msg + "\n")
			} else {
				fmt.Println(colorify(unescape(msg)))
			}
			skipElse = false

		case ast.RuleTypeFor:
			parts := strings.SplitN(rule.Target, " in ", 2)
			if len(parts) != 2 {
				return fmt.Errorf("%s:%d: invalid FOR syntax. Expected '#FOR: var in [list]'",
					r.CurrentFile, rule.Line)
			}
			varName := strings.TrimSpace(parts[0])
			
			// Si on a directement un {VAR}, InterpolateValue va retourner l'objet brut
			listValStr := strings.TrimSpace(parts[1])
			var listVal interface{} = ast.Literal(listValStr)
			listVal = util.InterpolateValue(listVal, scope)

			if arr, ok := listVal.(ast.Array); ok {
				for _, item := range arr {
					child := &Scope{Parent: scope, Capture: scope.Capture, Runner: r}
					child.Set(varName, item, rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath, r)
						}
					}
					if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt, r); err != nil {
						return err
					}
				}
			} else if arrRaw, ok := listVal.([]interface{}); ok {
				for _, item := range arrRaw {
					child := &Scope{Parent: scope, Capture: scope.Capture, Runner: r}
					child.Set(varName, item, rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath, r)
						}
					}
					if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt, r); err != nil {
						return err
					}
				}
			} else if obj, ok := listVal.(ast.Object); ok {
				var keys []string
				for k := range obj {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					child := &Scope{Parent: scope, Capture: scope.Capture, Runner: r}
					child.Set(varName, ast.Literal(k), rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath, r)
						}
					}
					if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt, r); err != nil {
						return err
					}
				}
			} else if objRaw, ok := listVal.(map[string]interface{}); ok {
				var keys []string
				for k := range objRaw {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					child := &Scope{Parent: scope, Capture: scope.Capture, Runner: r}
					child.Set(varName, ast.Literal(k), rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath, r)
						}
					}
					if err := executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt, r); err != nil {
						return err
					}
				}
			}
			skipElse = false
		}
	}
	return nil
}

// applyFileRule exécute les règles BEGIN, CREATE et GLOBAL.
func applyFileRule(initialTarget, currentBase string, rule ast.Rule, parentScope *Scope, r *Runner) error {
	scope := &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  parentScope.Modules,
		Parent:   parentScope,
		Capture:  parentScope.Capture,
		Runner:   r,
	}

	target := util.Interpolate(rule.Target, scope)
	if rule.As != "" {
		scope.Set(rule.As, target, rule.Line)
	}

	switch rule.Type {
	case ast.RuleTypeCreate:
		fullPath := filepath.Join(currentBase, target)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := processInception(rule.Content, scope, r)
		ext := filepath.Ext(fullPath)
		
		// Exécution récursive des règles et actions
		executeActions(&rule, &content, scope, ext, currentBase, r.CurrentFile, r)
		if err := executeRules(rule.SubRules, initialTarget, currentBase, r.CurrentFile, scope, &content, ext, r); err != nil {
			return err
		}
		
		content = engine.CleanOrphans(content)
		os.WriteFile(fullPath, []byte(content), 0644)

	case ast.RuleTypeBegin:
		fullPath := filepath.Join(currentBase, target)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("%s:%d: target file '%s' does not exist for BEGIN",
				r.CurrentFile, rule.Line, target)
		}
		content := string(data)
		ext := filepath.Ext(fullPath)
		
		// Exécution récursive des règles et actions
		executeActions(&rule, &content, scope, ext, currentBase, r.CurrentFile, r)
		if err := executeRules(rule.SubRules, initialTarget, currentBase, r.CurrentFile, scope, &content, ext, r); err != nil {
			return err
		}
		
		content = engine.CleanOrphans(content)
		os.WriteFile(fullPath, []byte(content), 0644)

	case ast.RuleTypeGlobal:
		filepath.Walk(currentBase, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			data, _ := os.ReadFile(path)
			content := string(data)
			ext := filepath.Ext(path)
			
			// Pour GLOBAL, on applique les actions de la règle elle-même d'abord
			executeActions(&rule, &content, scope, ext, currentBase, r.CurrentFile, r)
			if err := executeRules(rule.SubRules, initialTarget, currentBase, r.CurrentFile, scope, &content, ext, r); err != nil {
				return err
			}
			
			content = engine.CleanOrphans(content)
			if content != string(data) {
				os.WriteFile(path, []byte(content), info.Mode())
			}
			return nil
		})
	}
	return nil
}

