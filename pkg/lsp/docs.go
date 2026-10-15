package lsp

import (
	"fmt"
	"strings"
)

type DocItem struct {
	Detail string
	Doc    string
}

type UnitDoc struct {
	Detail  string
	Doc     string
	Methods map[string]DocItem
}

type DirectiveDoc struct {
	Name    string
	Snippet string
	Detail  string
	Doc     string
}

// GetDocForWord recherche une documentation complète pour un mot ou une expression membre
func GetDocForWord(fullExpr, targetWord string) *DocItem {
	fullExpr = strings.TrimSpace(fullExpr)
	targetWord = strings.ToLower(strings.TrimSpace(targetWord))

	// 1. Recherche par expression membre complète ex: "Utils.log" ou "Net.fetch"
	if strings.Contains(fullExpr, ".") {
		parts := strings.Split(fullExpr, ".")
		if len(parts) == 2 {
			unitKey := strings.ToLower(parts[0])
			methodKey := strings.ToLower(parts[1])
			if unitDoc, ok := BuiltinUnitsDocs[unitKey]; ok {
				if methodDoc, ok := unitDoc.Methods[methodKey]; ok {
					return &methodDoc
				}
				for mK, mV := range unitDoc.Methods {
					if strings.EqualFold(mK, methodKey) {
						return &mV
					}
				}
			}
			if strDoc, ok := StringMethodsDocs[methodKey]; ok {
				return &strDoc
			}
			if numDoc, ok := NumberMethodsDocs[methodKey]; ok {
				return &numDoc
			}
			if listDoc, ok := ListMethodsDocs[methodKey]; ok {
				return &listDoc
			}
			if mapDoc, ok := MapMethodsDocs[methodKey]; ok {
				return &mapDoc
			}
		}
	}

	// 2. Recherche par unité
	if unitDoc, ok := BuiltinUnitsDocs[targetWord]; ok {
		return &DocItem{
			Detail: unitDoc.Detail,
			Doc:    unitDoc.Doc,
		}
	}

	// 3. Méthodes document
	if docUnit, ok := BuiltinUnitsDocs["document"]; ok {
		for mK, mV := range docUnit.Methods {
			if strings.EqualFold(mK, targetWord) {
				return &mV
			}
		}
	}

	// 4. Méthodes natives String, Number, List, Map
	if strDoc, ok := StringMethodsDocs[targetWord]; ok {
		return &strDoc
	}
	if numDoc, ok := NumberMethodsDocs[targetWord]; ok {
		return &numDoc
	}
	if listDoc, ok := ListMethodsDocs[targetWord]; ok {
		return &listDoc
	}
	if mapDoc, ok := MapMethodsDocs[targetWord]; ok {
		return &mapDoc
	}

	// 5. Méthodes chirurgicales legacy
	if surgDoc, ok := SurgicalMethodsDocs[targetWord]; ok {
		return &surgDoc
	}

	// 6. Mots-clés
	if kwDoc, ok := KeywordsDocs[targetWord]; ok {
		return &kwDoc
	}

	return nil
}

func FormatMarkdown(item *DocItem) string {
	if item == nil {
		return ""
	}
	var sb strings.Builder
	if item.Detail != "" {
		sb.WriteString(fmt.Sprintf("```xru\n%s\n```\n", item.Detail))
	}
	if item.Doc != "" {
		sb.WriteString(item.Doc)
	}
	return sb.String()
}
