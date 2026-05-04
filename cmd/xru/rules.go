package main

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
)

func executeActions(rule *ast.Rule, content *string, scope *Scope, fileExt, cb, rulePath string) {
	if content != nil {
		for _, action := range rule.Actions {
			*content = applyAction(*content, action, fileExt, scope, cb, rulePath)
		}
	}
}

// executeRules parcourt et exécute une liste de règles dans le scope donné.
func executeRules(rules []ast.Rule, initialTarget, currentBase, rulePath string, scope *Scope, content *string, fileExt string) {
	cb := currentBase
	skipElse := false

	for _, rule := range rules {
		if verbose {
			fmt.Printf("[DEBUG] Executing rule type: %s at line %d (content context: %v)\n", rule.Type, rule.Line, content != nil)
		}
		target := util.Interpolate(rule.Target, scope)
		checkSyntaxError(target, rule.Line)

		switch rule.Type {
		case ast.RuleTypeVar:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
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
				checkSyntaxError(val, rule.Line)
				scope.Set(rule.Target, unescape(val), rule.Line)

		case ast.RuleTypeVarBlock:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
			content := util.Interpolate(rule.Content, scope)
			content = util.Dedent(content)

			var val interface{} = content
			if strings.HasPrefix(rule.Command, "JSON") {
				var parsed interface{}
				if err := json.Unmarshal([]byte(content), &parsed); err != nil {
					fmt.Printf("%s:%d: %serror:%s failed to parse JSON in #%s: %v\n",
						currentFile, rule.Line, colorRed, colorReset, rule.Command, err)
					os.Exit(1)
				}
				val = parsed
			}
			scope.Set(rule.Target, val, rule.Line)
			skipElse = false

		case ast.RuleTypeSelect:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
			checkSyntaxError(target, rule.Line)
			if filepath.IsAbs(target) {
				cb = target
			} else {
				cb = filepath.Join(initialTarget, target)
			}
			info, err := os.Stat(cb)
			if os.IsNotExist(err) {
				fmt.Printf("%s:%d: %serror:%s directory '%s' does not exist\n",
					currentFile, rule.Line, colorRed, colorReset, cb)
				os.Exit(1)
			}
			if !info.IsDir() {
				fmt.Printf("%s:%d: %serror:%s path '%s' is a file, but SELECT requires a directory\n",
					currentFile, rule.Line, colorRed, colorReset, cb)
				os.Exit(1)
			}
			if rule.As != "" {
				scope.Set(rule.As, cb, rule.Line)
			}
			skipElse = false

		case ast.RuleTypeIf:
			if evalCondition(rule.Target, scope, cb) {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath)
					}
				}
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt)
				skipElse = true
			} else {
				skipElse = false
			}

		case ast.RuleTypeElseIf:
			if !skipElse && evalCondition(rule.Target, scope, cb) {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath)
					}
				}
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt)
				skipElse = true
			}

		case ast.RuleTypeElse:
			if !skipElse {
				if content != nil {
					for _, action := range rule.Actions {
						*content = applyAction(*content, action, fileExt, scope, cb, rulePath)
					}
				}
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt)
			}
			skipElse = false

		case ast.RuleTypeUse:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
			name := util.Interpolate(rule.Target, scope)
			alias := rule.As
			if alias == "" {
				alias = name
			}
			scope.RegisterModule(alias, name, rule.Line)
			skipElse = false

		case ast.RuleTypeModule:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
			parts := strings.SplitN(rule.Target, ".", 2)
			content := util.Interpolate(rule.Content, scope)
			checkSyntaxError(content, rule.Line)
			executeModuleAction(scope, cb, rulePath, parts[0], parts[1], content, rule.As, rule.Line)
			skipElse = false

		case ast.RuleTypeInclude:
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := parser.ParseFile(includePath)
			if err == nil {
				executeRules(irf.Rules, initialTarget, cb, includePath, scope, content, fileExt)
			}
			skipElse = false

		case ast.RuleTypeCall:
			includePath := target
			executeRules(rule.SubRules, initialTarget, cb, rulePath, scope, content, fileExt)
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := parser.ParseFile(includePath)
			if err == nil {
				executeRules(irf.Rules, initialTarget, cb, includePath, scope, content, fileExt)
			}
			skipElse = false

		case ast.RuleTypeBegin, ast.RuleTypeCreate, ast.RuleTypeGlobal:
			applyFileRule(initialTarget, cb, rule, scope)
			skipElse = false

		case ast.RuleTypeArg:
			executeActions(&rule, content, scope, fileExt, cb, rulePath)
			val := getTerminalArg(target)
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
			line := util.Interpolate(rule.Target, scope)
			parts := strings.SplitN(line, " in ", 2)
			if len(parts) != 2 {
				fmt.Printf("%s:%d: %serror:%s invalid FOR syntax. Expected '#FOR: var in [list]'\n",
					currentFile, rule.Line, colorRed, colorReset)
				os.Exit(1)
			}
			varName := strings.TrimSpace(parts[0])
			listVal := parser.ParseValue(strings.TrimSpace(parts[1]))
			listVal = util.InterpolateValue(listVal, scope).(ast.Value)

			if arr, ok := listVal.(ast.Array); ok {
				for _, item := range arr {
					child := &Scope{Parent: scope, Capture: scope.Capture}
					child.Set(varName, item, rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath)
						}
					}
					executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt)
				}
			} else if obj, ok := listVal.(ast.Object); ok {
				var keys []string
				for k := range obj {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					child := &Scope{Parent: scope, Capture: scope.Capture}
					child.Set(varName, ast.Literal(k), rule.Line)
					if content != nil {
						for _, action := range rule.Actions {
							*content = applyAction(*content, action, fileExt, child, cb, rulePath)
						}
					}
					executeRules(rule.SubRules, initialTarget, cb, rulePath, child, content, fileExt)
				}
			}
			skipElse = false
		}
	}
}

// applyFileRule exécute les règles BEGIN, CREATE et GLOBAL.
func applyFileRule(initialTarget, currentBase string, rule ast.Rule, parentScope *Scope) {
	scope := &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  parentScope.Modules,
		Parent:   parentScope,
		Capture:  parentScope.Capture,
	}

	target := util.Interpolate(rule.Target, scope)
	if rule.As != "" {
		scope.Set(rule.As, target, rule.Line)
	}

	switch rule.Type {
	case ast.RuleTypeCreate:
		fullPath := filepath.Join(currentBase, target)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := util.Interpolate(rule.Content, scope)
		ext := filepath.Ext(fullPath)
		
		// Exécution récursive des règles et actions
		executeActions(&rule, &content, scope, ext, currentBase, currentFile)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope, &content, ext)
		
		os.WriteFile(fullPath, []byte(content), 0644)

	case ast.RuleTypeBegin:
		fullPath := filepath.Join(currentBase, target)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Printf("%s:%d: %serror:%s target file '%s' does not exist for BEGIN\n",
				currentFile, rule.Line, colorRed, colorReset, target)
			os.Exit(1)
		}
		content := string(data)
		ext := filepath.Ext(fullPath)
		
		// Exécution récursive des règles et actions
		executeActions(&rule, &content, scope, ext, currentBase, currentFile)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope, &content, ext)
		
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
			executeActions(&rule, &content, scope, ext, currentBase, currentFile)
			executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope, &content, ext)
			
			if content != string(data) {
				os.WriteFile(path, []byte(content), info.Mode())
			}
			return nil
		})
	}
}
