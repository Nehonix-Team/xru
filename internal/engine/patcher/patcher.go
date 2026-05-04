/***************************************************************************
 * XFPM — Structured Text Patcher (STP)
 ***************************************************************************** */

package patcher

import (
	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

// ApplyPatch applies a structured patch to a string content.
func ApplyPatch(content string, op ast.PatchOp, path string, patch ast.Value) string {
	if path != "" {
		switch op {
		case ast.PatchMerge, ast.PatchSet:
			return mergeStructured(content, deepen(path, patch))
		case ast.PatchRM:
			return removeStructured(content, deepen(path, ast.Literal("")))
		case ast.PatchAppend, ast.PatchPush:
			return appendStructured(content, deepen(path, patch))
		}
	}

	switch op {
	case ast.PatchMerge:
		if obj, ok := patch.(ast.Object); ok {
			return mergeStructured(content, obj)
		}
	case ast.PatchRM:
		return removeStructured(content, patch)
	case ast.PatchAppend:
		if obj, ok := patch.(ast.Object); ok {
			return appendStructured(content, obj)
		}
	case ast.PatchRegex:
		if obj, ok := patch.(ast.Object); ok {
			return applyRegex(content, obj)
		}
	case ast.PatchRPK:
		if obj, ok := patch.(ast.Object); ok {
			return renameKeyStructured(content, obj)
		}
	case ast.PatchRPV:
		if obj, ok := patch.(ast.Object); ok {
			return mergeStructured(content, obj)
		}
	}
	return content
}
