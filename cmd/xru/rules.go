package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine"
)

// executeRules parcourt et exécute une liste de règles dans le scope donné.
func executeRules(rules []engine.Rule, initialTarget, currentBase, rulePath string, scope *Scope) {
	cb := currentBase
	skipElse := false

	for _, rule := range rules {
		target := engine.Interpolate(rule.Target, scope)
		checkSyntaxError(target, rule.Line)

		switch rule.Type {
		case engine.RuleTypeVar:
			skipElse = false
			trimmed := strings.TrimSpace(rule.Content)

			// Résolution directe d'une variable objet: {VAR}
			if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
				name := trimmed[1 : len(trimmed)-1]
				for strings.Contains(name, "{") {
					name = engine.Interpolate(name, scope)
				}
				if val, ok := scope.Get(name); ok {
					scope.Set(rule.Target, val, rule.Line)
					continue
				}
			}

			val := engine.Interpolate(rule.Content, scope)
			checkSyntaxError(val, rule.Line)
			scope.Set(rule.Target, unescape(val), rule.Line)

		case engine.RuleTypeVarBlock:
			content := engine.Interpolate(rule.Content, scope)
			content = engine.Dedent(content)

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

		case engine.RuleTypeSelect:
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

		case engine.RuleTypeIf:
			if evalCondition(rule.Target, scope, cb) {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
				skipElse = true
			} else {
				skipElse = false
			}

		case engine.RuleTypeElseIf:
			if !skipElse && evalCondition(rule.Target, scope, cb) {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
				skipElse = true
			}

		case engine.RuleTypeElse:
			if !skipElse {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
			}
			skipElse = false

		case engine.RuleTypeUse:
			name := engine.Interpolate(rule.Target, scope)
			alias := rule.As
			if alias == "" {
				alias = name
			}
			scope.RegisterModule(alias, name, rule.Line)
			skipElse = false

		case engine.RuleTypeModule:
			parts := strings.SplitN(rule.Target, ".", 2)
			content := engine.Interpolate(rule.Content, scope)
			checkSyntaxError(content, rule.Line)
			executeModuleAction(scope, cb, rulePath, parts[0], parts[1], content, rule.As, rule.Line)
			skipElse = false

		case engine.RuleTypeInclude:
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := engine.ParseFile(includePath)
			if err == nil {
				executeRules(irf.Rules, initialTarget, cb, includePath, scope)
			}
			skipElse = false

		case engine.RuleTypeCall:
			includePath := target
			executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := engine.ParseFile(includePath)
			if err == nil {
				executeRules(irf.Rules, initialTarget, cb, includePath, scope)
			}
			skipElse = false

		case engine.RuleTypeBegin, engine.RuleTypeCreate, engine.RuleTypeGlobal:
			applyFileRule(initialTarget, cb, rule, scope)
			skipElse = false

		case engine.RuleTypeArg:
			val := getTerminalArg(target)
			if rule.As != "" {
				scope.Set(rule.As, val, rule.Line)
			}
			skipElse = false

		case engine.RuleTypeFor:
			line := engine.Interpolate(rule.Target, scope)
			parts := strings.SplitN(line, " in ", 2)
			if len(parts) != 2 {
				fmt.Printf("%s:%d: %serror:%s invalid FOR syntax. Expected '#FOR: var in [list]'\n",
					currentFile, rule.Line, colorRed, colorReset)
				os.Exit(1)
			}
			varName := strings.TrimSpace(parts[0])
			listVal := engine.ParseValue(strings.TrimSpace(parts[1]))
			listVal = engine.InterpolateValue(listVal, scope).(engine.Value)

			if arr, ok := listVal.(engine.Array); ok {
				for _, item := range arr {
					child := &Scope{Parent: scope}
					child.Set(varName, item, rule.Line)
					executeRules(rule.SubRules, initialTarget, cb, rulePath, child)
				}
			} else if obj, ok := listVal.(engine.Object); ok {
				var keys []string
				for k := range obj {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					child := &Scope{Parent: scope}
					child.Set(varName, engine.Literal(k), rule.Line)
					executeRules(rule.SubRules, initialTarget, cb, rulePath, child)
				}
			}
			skipElse = false
		}
	}
}

// applyFileRule exécute les règles BEGIN, CREATE et GLOBAL.
func applyFileRule(initialTarget, currentBase string, rule engine.Rule, parentScope *Scope) {
	scope := &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  parentScope.Modules,
		Parent:   parentScope,
	}

	target := engine.Interpolate(rule.Target, scope)
	if rule.As != "" {
		scope.Set(rule.As, target, rule.Line)
	}

	switch rule.Type {
	case engine.RuleTypeCreate:
		fullPath := filepath.Join(currentBase, target)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := engine.Interpolate(rule.Content, scope)
		for _, action := range rule.Actions {
			content = applyAction(content, action, filepath.Ext(fullPath), scope, currentBase, currentFile)
		}
		os.WriteFile(fullPath, []byte(content), 0644)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)

	case engine.RuleTypeBegin:
		fullPath := filepath.Join(currentBase, target)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return
		}
		content := string(data)
		for _, action := range rule.Actions {
			content = applyAction(content, action, filepath.Ext(fullPath), scope, currentBase, currentFile)
		}
		os.WriteFile(fullPath, []byte(content), 0644)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)

	case engine.RuleTypeGlobal:
		filepath.Walk(currentBase, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			data, _ := os.ReadFile(path)
			content := string(data)
			original := content
			for _, action := range rule.Actions {
				content = applyAction(content, action, filepath.Ext(path), scope, currentBase, currentFile)
			}
			if content != original {
				os.WriteFile(path, []byte(content), info.Mode())
			}
			return nil
		})
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)
	}
}
