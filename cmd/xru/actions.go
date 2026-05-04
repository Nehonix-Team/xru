package main

import (
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine"
)

// applyAction applique une action (inject, patch, var, module) au contenu d'un fichier.
func applyAction(content string, action engine.Action, fileExt string, scope *Scope, cb string, rulePath string) string {
	switch a := action.(type) {
	case engine.VarAction:
		val := engine.Interpolate(a.Value, scope)
		scope.Set(a.Name, unescape(val), a.Line)
		return content

	case engine.ModuleAction:
		executeModuleAction(scope, cb, rulePath, a.Module, a.Method, a.Target, a.As, a.Line)
		return content

	case engine.InjectAction:
		if a.Lang != "" && "."+strings.ToLower(a.Lang) != fileExt {
			return content
		}
		code := engine.Interpolate(a.Code, scope)
		checkSyntaxError(code, a.Line)
		return engine.InjectCode(content, a.Key, code)

	case engine.PatchAction:
		path := engine.Interpolate(a.Path, scope)
		val := engine.InterpolateValue(a.Value, scope).(engine.Value)
		return engine.ApplyPatch(content, a.Op, path, val)
	}
	return content
}
