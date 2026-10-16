package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) handleUseBuiltinsCompletion() []CompletionItem {
	var items []CompletionItem
	builtins := []struct {
		name, insert, doc string
	}{
		{"std", "std", "Standard IO & formatting unit (std.println, std.log, etc.)"},
		{"net", "net as Net", "Network & HTTP client unit"},
		{"fs", "fs as Fs", "Filesystem operations unit"},
		{"sys", "sys as Sys", "Operating system & environment unit"},
		{"crypto", "crypto as Crypto", "Cryptographic hashes & encryption unit"},
	}
	for _, b := range builtins {
		items = append(items, CompletionItem{
			Label:            b.name,
			Kind:             CompletionItemKindModule,
			Detail:           fmt.Sprintf("use %s", b.insert),
			Documentation:    MarkupContent{Kind: "markdown", Value: b.doc},
			InsertText:       b.insert,
			InsertTextFormat: InsertTextFormatPlainText,
		})
	}
	return items
}

func (s *Server) handleFromCompletion(uri, prefixLine, trimmedPrefix string) []CompletionItem {
	var items []CompletionItem

	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath == "" {
		return items
	}

	modCfg, errs := modules.LoadModConfig(modFilePath)
	if len(errs) > 0 || modCfg == nil {
		return items
	}

	// Case A: contains "use" -> completing module or module member
	useIdx := strings.Index(trimmedPrefix, " use")
	if useIdx != -1 {
		beforeUse := strings.TrimSpace(trimmedPrefix[:useIdx])
		ns := strings.TrimPrefix(beforeUse, "from")
		ns = strings.TrimSpace(ns)
		ns = strings.Trim(ns, "\"")

		afterUse := strings.TrimSpace(trimmedPrefix[useIdx+4:])

		// Check if there is "::"
		if colonIdx := strings.Index(afterUse, "::"); colonIdx != -1 {
			modName := strings.TrimSpace(afterUse[:colonIdx])
			modDetails, err := modCfg.ResolveModule(ns, modName)
			if err == nil && modDetails != nil {
				for _, exp := range modDetails.Exports {
					kind := CompletionItemKindFunction
					if exp.Kind == "var" || exp.Kind == "const" || exp.Kind == "arg" {
						kind = CompletionItemKindVariable
					}
					items = append(items, CompletionItem{
						Label:            exp.Name,
						Kind:             kind,
						Detail:           fmt.Sprintf("%s %s (%s)", exp.Kind, exp.Name, exp.Type),
						Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Exported from `%s` in namespace `%s`", modName, ns)},
						InsertText:       exp.Name,
						InsertTextFormat: InsertTextFormatPlainText,
					})
				}
			}
			return items
		}

		// User is completing module name in namespace
		mods, err := modCfg.GetNamespaceModules(ns)
		if err == nil {
			for _, m := range mods {
				items = append(items, CompletionItem{
					Label:            m.Name,
					Kind:             CompletionItemKindModule,
					Detail:           fmt.Sprintf("module in namespace \"%s\"", ns),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Defined at `%s`", m.FilePath)},
					InsertText:       m.Name,
					InsertTextFormat: InsertTextFormatPlainText,
				})
			}
		}
		return items
	}

	// Case B: No "use" yet -> completing namespace from _mod.xru
	quoteCount := strings.Count(prefixLine, "\"")

	for ns, targetPath := range modCfg.Mappings {
		doc := fmt.Sprintf("Namespace mapped to `%s` in `_mod.xru`", targetPath)

		if quoteCount%2 == 1 {
			// Opened quote: `from "` or `from "ser`
			items = append(items, CompletionItem{
				Label:            ns,
				Kind:             CompletionItemKindFolder,
				Detail:           fmt.Sprintf("namespace (%s)", targetPath),
				Documentation:    MarkupContent{Kind: "markdown", Value: doc},
				InsertText:       fmt.Sprintf("%s\" use ", ns),
				FilterText:       ns,
				InsertTextFormat: InsertTextFormatPlainText,
			})
		} else if strings.HasSuffix(strings.TrimSpace(prefixLine), "\"") {
			// Quote already closed: e.g. `from "services"`
			items = append(items, CompletionItem{
				Label:            "use",
				Kind:             CompletionItemKindKeyword,
				Detail:           "use <module>",
				Documentation:    MarkupContent{Kind: "markdown", Value: "Specify module to import"},
				InsertText:       " use ",
				InsertTextFormat: InsertTextFormatPlainText,
			})
			return items
		} else {
			// No quotes: e.g. `from` or `from `
			items = append(items, CompletionItem{
				Label:            fmt.Sprintf("\"%s\"", ns),
				Kind:             CompletionItemKindFolder,
				Detail:           fmt.Sprintf("namespace (%s)", targetPath),
				Documentation:    MarkupContent{Kind: "markdown", Value: doc},
				InsertText:       fmt.Sprintf("\"%s\" use ", ns),
				FilterText:       ns,
				InsertTextFormat: InsertTextFormatPlainText,
			})
		}
	}

	return items
}

func (s *Server) handleScopeMemberCompletion(uri, beforeColon string) []CompletionItem {
	var items []CompletionItem
	trimmed := strings.TrimSpace(beforeColon)
	trimmedNoAmp := strings.TrimPrefix(trimmed, "&")
	var modIdent string
	for i := len(trimmedNoAmp) - 1; i >= 0; i-- {
		if !isWordChar(trimmedNoAmp[i]) {
			modIdent = trimmedNoAmp[i+1:]
			break
		}
		if i == 0 {
			modIdent = trimmedNoAmp
		}
	}
	if modIdent == "" {
		return items
	}

	// 1. Check if modIdent is a local unit, builtin unit, or self
	if text, ok := s.docs[uri]; ok {
		docLines := strings.Split(text, "\n")
		targetUnit := modIdent
		if modIdent == "self" {
			reFindUnit := regexp.MustCompile(`^\s*unit\s+([a-zA-Z0-9_]+)`)
			for _, l := range docLines {
				if m := reFindUnit.FindStringSubmatch(l); len(m) > 1 {
					targetUnit = m[1]
					break
				}
			}
		}
		unitMembers := collectUnitMembers(docLines, targetUnit)
		if len(unitMembers) > 0 {
			return unitMembers
		}
	}

	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath == "" {
		return items
	}

	modCfg, errs := modules.LoadModConfig(modFilePath)
	if len(errs) > 0 || modCfg == nil {
		return items
	}

	for ns := range modCfg.Mappings {
		mDetails, err := modCfg.ResolveModule(ns, modIdent)
		if err == nil && mDetails != nil {
			for _, exp := range mDetails.Exports {
				kind := CompletionItemKindFunction
				if exp.Kind == "var" || exp.Kind == "const" {
					kind = CompletionItemKindVariable
				}
				items = append(items, CompletionItem{
					Label:            exp.Name,
					Kind:             kind,
					Detail:           fmt.Sprintf("%s %s (%s)", exp.Kind, exp.Name, exp.Type),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Exported from `%s` in namespace `%s`", modIdent, ns)},
					InsertText:       exp.Name,
					InsertTextFormat: InsertTextFormatPlainText,
				})
			}
			return items
		}
	}
	return items
}

func resolveUnitAlias(lines []string, alias string) string {
	re := regexp.MustCompile(fmt.Sprintf(`\buse\s+([a-zA-Z0-9_]+)\s+as\s+%s\b`, regexp.QuoteMeta(alias)))
	for _, l := range lines {
		if m := re.FindStringSubmatch(l); len(m) > 1 {
			return m[1]
		}
	}
	reFrom := regexp.MustCompile(fmt.Sprintf(`\bfrom\s+"[^"]+"\s+use\s+([a-zA-Z0-9_]+)\s+as\s+%s\b`, regexp.QuoteMeta(alias)))
	for _, l := range lines {
		if m := reFrom.FindStringSubmatch(l); len(m) > 1 {
			return m[1]
		}
	}
	return alias
}
