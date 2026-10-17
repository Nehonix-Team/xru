package navigation

import (
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

// HandleDefinition resolves the definition location for a symbol, expression, property, or import at pos.
func (n *Navigator) HandleDefinition(uri string, pos Position) *Location {
	if n.GetDoc == nil {
		return nil
	}
	text, ok := n.GetDoc(uri)
	if !ok {
		return nil
	}

	fullExpr, word := GetFullExpressionAt(text, pos.Line, pos.Character)
	if word == "" {
		return nil
	}

	curFilePath := uriToPath(uri)
	fileDir := filepath.Dir(curFilePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	var modCfg *modules.ModConfig
	if modFilePath != "" {
		if cfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && cfg != nil {
			modCfg = cfg
		}
	}

	lines := strings.Split(text, "\n")
	var curLine string
	if pos.Line >= 0 && pos.Line < len(lines) {
		curLine = lines[pos.Line]
	}
	trimmedCurLine := strings.TrimSpace(curLine)
	isCurLineImport := strings.HasPrefix(trimmedCurLine, "from ") || strings.HasPrefix(trimmedCurLine, "use ")

	fileImports := ParseFileImports(text)

	// 1. Explicit qualified expression: module::member (e.g. gArgs::author)
	if strings.Contains(fullExpr, "::") {
		return n.resolveQualifiedDefinition(modCfg, fileImports, fullExpr, fileDir)
	}

	// 1.5 Member navigation: obj.member (e.g. doc.setJson, Fs.exists, std.println, opts.silent, params.a, $f{params.a})
	if loc := n.resolveMemberDefinition(lines, fileImports, modCfg, fullExpr, word, uri, fileDir); loc != nil {
		return loc
	}

	// 1.6 Clicked on a property key inside a typed map literal: e.g. `mode: "log"` in Sys.require("...", @map{ mode: "log" })
	if isKey, keyWord, mapStartLine := IsCursorOnMapKey(lines, pos); isKey {
		if n.ResolveExpectedMapType != nil {
			expectedType := n.ResolveExpectedMapType(lines, mapStartLine, curFilePath)
			if expectedType != "" {
				if loc := n.FindMemberInInterface(lines, expectedType, keyWord, uri); loc != nil {
					return loc
				}
			}
		}
		// In a map key context: NEVER fall through to external variables or modules!
		return nil
	}

	// 2. Clicked on a namespace name in _mod.xru
	if loc := resolveNamespaceDefinition(modCfg, modFilePath, word); loc != nil {
		return loc
	}

	// 3. Cursor is directly ON the import line
	if isCurLineImport {
		if loc := n.resolveImportLineDefinition(modCfg, fileImports, pos.Line, word, fileDir); loc != nil {
			return loc
		}
	}

	// 4. Search Local Definitions in active file (variables, lambdas, funcs, interfaces, types)
	if loc := resolveLocalDefinition(lines, word, uri); loc != nil {
		return loc
	}

	// 5. Check if word is imported into the active file
	if loc := n.resolveImportedSymbolDefinition(modCfg, fileImports, lines, word, uri, fileDir); loc != nil {
		return loc
	}

	// 6. Builtin primitive contracts & types in std/builtin.xru
	// 7. Standard library modules: std, path, fs, crypto, date, math, sys, cmd, json, net
	// 8. Standard library global functions and declarations (e.g. Drop, drop, Keep, keep, println)
	return n.resolveStdDefinition(word, fileDir)
}

func uriToPath(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

func pathToURI(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return "file://" + path
}
