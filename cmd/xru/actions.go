package main

import (
	"fmt"
	"strings"
	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/engine/patcher"
	"github.com/Nehonix-Team/xru/internal/engine/util" 
	"github.com/Nehonix-Team/xru/internal/engine/parser"
)

// applyAction applique une action (inject, patch, var, module) au contenu d'un fichier.
func applyAction(content string, action ast.Action, fileExt string, scope *Scope, cb string, rulePath string) string {
	switch a := action.(type) {
	case ast.VarAction:
		val := util.Interpolate(a.Value, scope)
		scope.Set(a.Name, unescape(val), a.Line)
		return content

	case ast.ModuleAction:
		executeModuleAction(scope, cb, rulePath, a.Module, a.Method, a.Target, a.As, a.Line)
		return content

	case ast.InjectAction:
		if a.Lang != "" && "."+strings.ToLower(a.Lang) != fileExt {
			return content
		}
		code := processInception(a.Code, scope)
		checkSyntaxError(code, a.Line)
		return engine.InjectCode(content, a.Key, code)

	case ast.PatchAction:
		path := util.Interpolate(a.Path, scope)
		val := util.InterpolateValue(a.Value, scope).(ast.Value)
		// Si la valeur est un Literal (string), on lui applique aussi l'inception
		if lit, ok := val.(ast.Literal); ok {
			val = ast.Literal(processInception(string(lit), scope))
		}
		return patcher.ApplyPatch(content, a.Op, path, val)
	}
	return content
}

// processInception traite les blocs <# ... > et les variables {VAR} dans le texte.
func processInception(src string, scope *Scope) string {
	if verbose {
		fmt.Printf("[DEBUG] processInception on source (len %d)\n", len(src))
	}
	if !strings.Contains(src, "<#") {
		return util.Interpolate(src, scope)
	}

	// On transforme le texte en un script XRU virtuel
	// Les parties de texte deviennent des #LOG: "..."
	// Les parties <#... > deviennent des directives réelles.
	
	virtualScript := ""
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "<#") && strings.Contains(trimmed, ">") {
			// On sépare le texte avant, la directive, et le texte après
			start := strings.Index(line, "<#")
			end := strings.Index(line, ">")
			before := line[:start]
			directive := line[start+2 : end]
			after := line[end+1:]
			
			before = strings.ReplaceAll(before, "{", "\\{")
			before = strings.ReplaceAll(before, "}", "\\}")
			after = strings.ReplaceAll(after, "{", "\\{")
			after = strings.ReplaceAll(after, "}", "\\}")
			
			if strings.TrimSpace(before) != "" {
				virtualScript += "#LOG: " + before + "\n"
			}
			virtualScript += "#" + strings.TrimSpace(directive) + "\n"
			if strings.TrimSpace(after) != "" {
				virtualScript += "#LOG: " + after + "\n"
			}
		} else {
			// Texte pur : on échappe les accolades pour éviter les erreurs d'interpolation
			// sur du code (ex: JS/TS) tout en gardant la possibilité d'utiliser {VAR}
			// si elles sont bien formées. 
			// Mais pour l'instant, on va être radical: on échappe tout ce qui n'est pas <#
			// et on laisse Interpolate gérer les variables s'il y en a.
			// En fait, le plus simple est de ne PAS échapper si c'est une variable valide.
			// Mais pour corriger le crash immédiat:
			line = strings.ReplaceAll(line, "{", "\\{")
			line = strings.ReplaceAll(line, "}", "\\}")
			virtualScript += "#LOG: " + line + "\n"
		}
	}


	// On parse ce script virtuel
	rf, err := parser.Parse(virtualScript)
	if err != nil {
		return "[INCEPTION_ERROR: " + err.Error() + "]"
	}
	// On l'exécute avec capture
	capture := &strings.Builder{}
	childScope := &Scope{
		Vars:     scope.Vars,
		DefLines: scope.DefLines,
		Used:     scope.Used,
		Modules:  scope.Modules,
		Parent:   scope,
		Capture:  capture,
	}

	executeRules(rf.Rules, "", "", "", childScope, nil, "")
	
	return strings.TrimSuffix(capture.String(), "\n")
}
