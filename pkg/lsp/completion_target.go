package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type varTargetInfo struct {
	isTask           bool
	taskReturnType   string
	unitType         string
	isString         bool
	isNumber         bool
	isList           bool
	listElemType     string
	isMap            bool
	mapFields        []CompletionItem
	interfaceName    string
	fromModuleSource string
	fromModuleName   string
}

func findVariableTargetInfo(lines []string, varName string) *varTargetInfo {
	// 0. Check from "... " use mod::@map as alias
	reFromUseMap := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+([a-zA-Z_][a-zA-Z0-9_]*)::@map(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?`)
	for _, line := range lines {
		if m := reFromUseMap.FindStringSubmatch(line); len(m) > 2 {
			source := m[1]
			modName := m[2]
			alias := m[3]
			if alias == "" {
				alias = modName
			}
			if alias == varName {
				return &varTargetInfo{isMap: true, fromModuleSource: source, fromModuleName: modName}
			}
		}
	}

	reVar := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?(?:let|const|var)\s+%s(?:\s*:\s*([^=]+?))?(?:\s*=\s*(.*))?$`, regexp.QuoteMeta(varName)))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		cleanLine := trimmed
		if idx := strings.Index(cleanLine, "//"); idx != -1 {
			cleanLine = strings.TrimSpace(cleanLine[:idx])
		}
		if m := reVar.FindStringSubmatch(cleanLine); len(m) > 1 {
			typeStr := strings.TrimSpace(m[1])
			valStr := strings.TrimSpace(m[2])

			valClean := strings.TrimSpace(strings.TrimRight(valStr, "."))

			// 1. Explicit task type
			if strings.HasPrefix(typeStr, "task[") && strings.HasSuffix(typeStr, "]") {
				inner := strings.TrimSuffix(strings.TrimPrefix(typeStr, "task["), "]")
				return &varTargetInfo{isTask: true, taskReturnType: inner}
			} else if typeStr == "task" {
				return &varTargetInfo{isTask: true, taskReturnType: "any"}
			}

			// 2. Assigned with `with <expr>`
			if strings.HasPrefix(valStr, "with ") {
				withTarget := strings.TrimSpace(strings.TrimPrefix(valStr, "with "))
				retType := "any"
				if strings.Contains(strings.ToLower(withTarget), "net.fetch") {
					retType = "string"
				} else if parenIdx := strings.Index(withTarget, "("); parenIdx != -1 {
					memberPart := withTarget[:parenIdx]
					if dot := strings.LastIndex(memberPart, "."); dot != -1 {
						callObj := strings.TrimSpace(memberPart[:dot])
						callMethod := strings.TrimSpace(memberPart[dot+1:])
						retType = inferUnitMethodReturnType(lines, callObj, callMethod)
					}
				}
				if typeStr != "" && typeStr != "any" {
					retType = typeStr
				}
				return &varTargetInfo{isTask: true, taskReturnType: retType}
			}

			// 3. Date unit structured map return (e.g. date.now(), date.parse())
			if strings.Contains(valClean, "date.now") || strings.Contains(valClean, "date.parse") || strings.Contains(valClean, "date.add") {
				return &varTargetInfo{isMap: true, mapFields: getDateMapFields()}
			}

			// 4. Explicit Interface or Custom Type in annotation (takes precedence over map literal)
			if typeStr != "" && typeStr != "number" && typeStr != "string" && typeStr != "bool" && typeStr != "duration" && typeStr != "any" && typeStr != "void" && typeStr != "map" && !strings.HasPrefix(typeStr, "@map") {
				return &varTargetInfo{unitType: typeStr, interfaceName: typeStr, isMap: strings.HasPrefix(valClean, "@map") || strings.HasPrefix(valClean, "{")}
			}

			// 5. Inline map type in annotation: let opts: @map{ silent?: bool }
			if strings.HasPrefix(typeStr, "@map{") || strings.HasPrefix(typeStr, "@map {") {
				fields := parseInlineMapFields(typeStr)
				return &varTargetInfo{isMap: true, mapFields: fields}
			}

			// 6. Inline map literal or @map literal: { a: 1, b: 2 } or @map{ a: 1 }
			if strings.HasPrefix(valClean, "{") || strings.HasPrefix(valClean, "@map") {
				fields := extractMapLiteralFields(valClean)
				if len(fields) > 0 || typeStr == "map" || typeStr == "" {
					return &varTargetInfo{isMap: true, mapFields: fields}
				}
			}
			if typeStr == "map" {
				return &varTargetInfo{isMap: true}
			}

			// 6. Instantiated via @UnitName()
			if strings.HasPrefix(valClean, "@") && !strings.HasPrefix(valClean, "@map") {
				unitName := strings.TrimPrefix(valClean, "@")
				if parenIdx := strings.Index(unitName, "("); parenIdx != -1 {
					unitName = unitName[:parenIdx]
				}
				return &varTargetInfo{unitType: strings.TrimSpace(unitName)}
			}

			// 7. String type
			if typeStr == "string" || strings.HasPrefix(valClean, "\"") || strings.HasPrefix(valClean, "'") || strings.HasPrefix(valClean, "$f\"") {
				return &varTargetInfo{isString: true}
			}

			// 8. Number type
			if typeStr == "number" || isNumericLiteral(valClean) || strings.HasPrefix(valClean, "math.") || strings.HasPrefix(valClean, "date.timestamp") || strings.HasPrefix(valClean, "date.diff") {
				return &varTargetInfo{isNumber: true}
			}

			// 9. List type
			if strings.HasPrefix(typeStr, "list") || strings.HasPrefix(valClean, "[") {
				elemT := ""
				if strings.HasPrefix(typeStr, "list[") && strings.HasSuffix(typeStr, "]") {
					elemT = strings.TrimSuffix(strings.TrimPrefix(typeStr, "list["), "]")
				}
				return &varTargetInfo{isList: true, listElemType: elemT}
			}

			// 10. Custom unit or interface type in annotation
			if typeStr != "" && typeStr != "number" && typeStr != "bool" && typeStr != "duration" && typeStr != "any" && typeStr != "void" {
				return &varTargetInfo{unitType: typeStr, interfaceName: typeStr}
			}

			// 11. Inferred expression type resolution for unannotated variables
			if valClean != "" {
				if exprInfo := resolveExpressionTargetInfo("", lines, valClean); exprInfo != nil {
					return exprInfo
				}
			}
		}
	}

	// 9. Check file statement parameters (e, data) or (doc, err)
	reFileStmt := regexp.MustCompile(`^\s*file\s+.*?(?:,\s*)?\((.*?)\)\s*(?:->|=>)`)
	for _, line := range lines {
		if m := reFileStmt.FindStringSubmatch(line); len(m) > 1 {
			rawParams := strings.Split(m[1], ",")
			var pNames []string
			var pTypes []string
			for _, p := range rawParams {
				pTrim := strings.TrimSpace(p)
				if colon := strings.Index(pTrim, ":"); colon != -1 {
					pNames = append(pNames, strings.TrimSpace(pTrim[:colon]))
					pTypes = append(pTypes, strings.TrimSpace(pTrim[colon+1:]))
				} else {
					pNames = append(pNames, pTrim)
					pTypes = append(pTypes, "")
				}
			}
			isErrFirst := false
			if len(pNames) >= 2 {
				p0Low := strings.ToLower(pNames[0])
				p1Low := strings.ToLower(pNames[1])
				if pTypes[1] == "Document" || ((p0Low == "e" || p0Low == "err" || p0Low == "error") && (p1Low == "data" || p1Low == "d" || p1Low == "doc" || p1Low == "content" || pTypes[1] == "Document")) {
					isErrFirst = true
				}
			}
			hasLocalInterface := func(ifaceName string) bool {
				reIface := regexp.MustCompile(fmt.Sprintf(`^\s*interface\s+%s(?:\s+@map)?\s*\{`, regexp.QuoteMeta(ifaceName)))
				for _, line := range lines {
					if reIface.MatchString(line) {
						return true
					}
				}
				return false
			}

			resolveDocOrCustomTarget := func(typeName string) *varTargetInfo {
				if typeName == "" {
					typeName = "Document"
				}
				if hasLocalInterface(typeName) {
					return &varTargetInfo{interfaceName: typeName}
				}
				if typeName == "Document" {
					return &varTargetInfo{unitType: "Document"}
				}
				return &varTargetInfo{interfaceName: typeName}
			}

			resolveErrTarget := func(typeName string) *varTargetInfo {
				if typeName != "" && typeName != "any" && typeName != "string" {
					if len(collectInterfaceMembers(lines, typeName, "")) > 0 {
						return &varTargetInfo{interfaceName: typeName}
					}
				}
				return &varTargetInfo{isString: true}
			}

			if len(pNames) == 1 {
				if pNames[0] == varName {
					p0Low := strings.ToLower(pNames[0])
					if p0Low == "e" || p0Low == "err" || p0Low == "error" {
						return resolveErrTarget(pTypes[0])
					}
					return resolveDocOrCustomTarget(pTypes[0])
				}
			} else if len(pNames) >= 2 {
				if isErrFirst {
					if pNames[0] == varName {
						return resolveErrTarget(pTypes[0])
					}
					if pNames[1] == varName {
						return resolveDocOrCustomTarget(pTypes[1])
					}
				} else {
					if pNames[0] == varName {
						return resolveDocOrCustomTarget(pTypes[0])
					}
					if pNames[1] == varName {
						return resolveErrTarget(pTypes[1])
					}
				}
			}
		}
	}

	// 10. Check func parameters
	fullDoc := strings.Join(lines, "\n")
	reFuncStart := regexp.MustCompile(`(?m)^\s*(?:global\s+)?func\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\(`)
	funcMatches := reFuncStart.FindAllStringIndex(fullDoc, -1)
	for _, m := range funcMatches {
		openParen := m[1] - 1
		depth := 1
		closeParen := -1
		for i := openParen + 1; i < len(fullDoc); i++ {
			if fullDoc[i] == '(' || fullDoc[i] == '{' {
				depth++
			} else if fullDoc[i] == ')' || fullDoc[i] == '}' {
				depth--
				if depth == 0 {
					closeParen = i
					break
				}
			}
		}
		if closeParen > openParen {
			paramsStr := fullDoc[openParen+1 : closeParen]
			for _, pStr := range splitParamsRespectingBraces(paramsStr) {
				colonIdx := -1
				depthBrace := 0
				for idx, r := range pStr {
					if r == '{' {
						depthBrace++
					} else if r == '}' {
						depthBrace--
					} else if r == ':' && depthBrace == 0 {
						colonIdx = idx
						break
					}
				}
				pName := pStr
				pType := "any"
				if colonIdx != -1 {
					pName = strings.TrimSpace(pStr[:colonIdx])
					pType = strings.TrimSpace(pStr[colonIdx+1:])
				}
				pName = strings.TrimSuffix(strings.TrimSpace(pName), "?")
				if pName == varName {
					cleanType := strings.TrimSpace(pType)
					if strings.HasPrefix(cleanType, "@map{") || strings.HasPrefix(cleanType, "@map {") {
						fields := parseInlineMapFields(cleanType)
						if len(fields) > 0 {
							return &varTargetInfo{isMap: true, mapFields: fields}
						}
					}
					if cleanType == "string" {
						return &varTargetInfo{isString: true}
					}
					if cleanType == "number" {
						return &varTargetInfo{isNumber: true}
					}
					if strings.HasPrefix(cleanType, "list") {
						return &varTargetInfo{isList: true}
					}
					if cleanType == "map" {
						return &varTargetInfo{isMap: true}
					}
					if cleanType != "" && cleanType != "any" && cleanType != "bool" && cleanType != "duration" && cleanType != "void" {
						return &varTargetInfo{unitType: cleanType, interfaceName: cleanType}
					}
				}
			}
		}
	}

	return nil
}

func splitParamsRespectingBraces(s string) []string {
	var res []string
	var cur strings.Builder
	depthParen := 0
	depthBrace := 0
	depthBracket := 0

	for _, r := range s {
		switch r {
		case '(':
			depthParen++
			cur.WriteRune(r)
		case ')':
			depthParen--
			cur.WriteRune(r)
		case '{':
			depthBrace++
			cur.WriteRune(r)
		case '}':
			depthBrace--
			cur.WriteRune(r)
		case '[':
			depthBracket++
			cur.WriteRune(r)
		case ']':
			depthBracket--
			cur.WriteRune(r)
		case ',':
			if depthParen == 0 && depthBrace == 0 && depthBracket == 0 {
				res = append(res, strings.TrimSpace(cur.String()))
				cur.Reset()
				continue
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		res = append(res, strings.TrimSpace(cur.String()))
	}
	return res
}

func extractMapLiteralFields(valStr string) []CompletionItem {
	var items []CompletionItem
	start := strings.Index(valStr, "{")
	end := strings.LastIndex(valStr, "}")
	if start == -1 || end == -1 || end <= start {
		return items
	}
	inner := valStr[start+1 : end]
	reKey := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*|"(?:[^"\\]|\\.)*")\s*(?::|->)\s*([^,;}\n]+)`)
	matches := reKey.FindAllStringSubmatch(inner, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		k := strings.Trim(strings.TrimSpace(m[1]), "\"")
		val := strings.TrimSpace(m[2])
		if !seen[k] && k != "" {
			seen[k] = true
			items = append(items, CompletionItem{
				Label:            k,
				Kind:             CompletionItemKindProperty,
				Detail:           fmt.Sprintf("property %s: %s", k, val),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Inferred from map literal: `%s`", val)},
				InsertText:       k,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", k),
			})
		}
	}
	return items
}

func parseInlineMapFields(typeStr string) []CompletionItem {
	var items []CompletionItem
	start := strings.Index(typeStr, "{")
	end := strings.LastIndex(typeStr, "}")
	if start == -1 || end == -1 || end <= start {
		return items
	}
	inner := typeStr[start+1 : end]
	parts := splitParamsRespectingBraces(inner)
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		colonIdx := -1
		depthBrace := 0
		for idx, r := range trimmed {
			if r == '{' {
				depthBrace++
			} else if r == '}' {
				depthBrace--
			} else if r == ':' && depthBrace == 0 {
				colonIdx = idx
				break
			}
		}
		fName := trimmed
		fType := "any"
		isOptional := false
		if colonIdx != -1 {
			fName = strings.TrimSpace(trimmed[:colonIdx])
			fType = strings.TrimSpace(trimmed[colonIdx+1:])
		}
		if strings.HasSuffix(fName, "?") {
			isOptional = true
			fName = strings.TrimSpace(strings.TrimSuffix(fName, "?"))
		}
		if strings.HasSuffix(fType, "?") {
			isOptional = true
			fType = strings.TrimSpace(strings.TrimSuffix(fType, "?"))
		}
		kind := CompletionItemKindProperty
		insertText := fName
		if strings.HasPrefix(fType, "func") {
			kind = CompletionItemKindMethod
			insertText = fName + "()"
		}
		optStr := ""
		if isOptional {
			optStr = "?"
		}
		items = append(items, CompletionItem{
			Label:            fName,
			Kind:             kind,
			Detail:           fmt.Sprintf("%s%s: %s", fName, optStr, fType),
			Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Inline field of type `%s`", fType)},
			InsertText:       insertText,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("00_%s", fName),
		})
	}
	return items
}

func collectInterfaceMembers(lines []string, ifaceName string, currentFilePath string) []CompletionItem {
	var items []CompletionItem
	reInterface := regexp.MustCompile(fmt.Sprintf(`^\s*(?:interface|type)\s+%s\b`, regexp.QuoteMeta(ifaceName)))

	scanLines := func(srcLines []string) bool {
		inIface := false
		depth := 0
		found := false
		for _, l := range srcLines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if !inIface {
				if reInterface.MatchString(l) {
					inIface = true
					found = true
					depth += strings.Count(l, "{") - strings.Count(l, "}")
				}
				continue
			}

			depth += strings.Count(l, "{") - strings.Count(l, "}")
			if depth <= 0 {
				break
			}
			if depth > 1 && strings.Count(l, "{") == 0 {
				continue
			}

			lineContent := trimmed
			if idx := strings.Index(lineContent, "//"); idx != -1 {
				lineContent = strings.TrimSpace(lineContent[:idx])
			}
			lineContent = strings.TrimSuffix(strings.TrimSuffix(lineContent, ","), ";")
			lineContent = strings.TrimSpace(lineContent)
			if lineContent == "" || lineContent == "{" || lineContent == "}" {
				continue
			}

			reField := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)(\?)?\s*(?::\s*(.+)|(?:\s+(.+)))?$`)
			if m := reField.FindStringSubmatch(lineContent); len(m) > 1 {
				fName := m[1]
				isOptional := m[2] == "?"
				fType := strings.TrimSpace(m[3])
				if fType == "" {
					fType = strings.TrimSpace(m[4])
				}
				if fType == "" {
					fType = "any"
				}

				kind := CompletionItemKindProperty
				insertText := fName
				if strings.HasPrefix(fType, "func") {
					kind = CompletionItemKindMethod
					insertText = fName + "()"
				}

				optStr := ""
				if isOptional {
					optStr = "?"
				}
				detail := fmt.Sprintf("%s%s: %s", fName, optStr, fType)

				items = append(items, CompletionItem{
					Label:            fName,
					Kind:             kind,
					Detail:           detail,
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Field of interface `%s`", ifaceName)},
					InsertText:       insertText,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("00_%s", fName),
				})
			}
		}
		return found
	}

	// 1. Scan in current file lines
	if scanLines(lines) {
		return items
	}

	// 2. Scan in workspace files in same directory if not found in current file
	if currentFilePath != "" {
		dir := filepath.Dir(currentFilePath)
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
					filePath := filepath.Join(dir, entry.Name())
					if filePath == currentFilePath {
						continue
					}
					content, readErr := os.ReadFile(filePath)
					if readErr == nil {
						fileLines := strings.Split(string(content), "\n")
						if scanLines(fileLines) {
							return items
						}
					}
				}
			}
		}
	}

	// 3. Scan in std directory for standard library interfaces (e.g. DateInfo in std/date.xru)
	curDir := ""
	if currentFilePath != "" {
		curDir = filepath.Dir(currentFilePath)
	}
	if stdDir := FindStdDir(curDir); stdDir != "" {
		stdEntries, err := os.ReadDir(stdDir)
		if err == nil {
			for _, entry := range stdEntries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
					content, readErr := os.ReadFile(filepath.Join(stdDir, entry.Name()))
					if readErr == nil {
						if scanLines(strings.Split(string(content), "\n")) {
							return items
						}
					}
				}
			}
		}
	}

	return items
}

func inferUnitMethodReturnType(lines []string, objName, methodName string) string {
	unitName := strings.TrimPrefix(objName, "&")
	if varInfo := findVariableTargetInfo(lines, objName); varInfo != nil && varInfo.unitType != "" {
		unitName = varInfo.unitType
	}
	if unitName == "self" {
		reFindUnit := regexp.MustCompile(`^\s*unit\s+([a-zA-Z0-9_]+)`)
		for _, l := range lines {
			if m := reFindUnit.FindStringSubmatch(l); len(m) > 1 {
				unitName = m[1]
				break
			}
		}
	}

	// 1. Built-in units
	if unitDoc, ok := BuiltinUnitsDocs[strings.ToLower(unitName)]; ok {
		if mDoc, ok := unitDoc.Methods[strings.ToLower(methodName)]; ok {
			if arrowIdx := strings.Index(mDoc.Detail, "->"); arrowIdx != -1 {
				return strings.TrimSpace(mDoc.Detail[arrowIdx+2:])
			}
		}
	}

	// 2. Local unit declaration in document
	reUnitStart := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\b`, regexp.QuoteMeta(unitName)))
	reFunc := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?func\s+%s\s*\(.*?\)(?:\s*->\s*(.+))?`, regexp.QuoteMeta(methodName)))

	inUnit := false
	depth := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !inUnit {
			if reUnitStart.MatchString(l) {
				inUnit = true
				depth += strings.Count(l, "{") - strings.Count(l, "}")
			}
			continue
		}

		depth += strings.Count(l, "{") - strings.Count(l, "}")
		if depth <= 0 {
			break
		}

		if m := reFunc.FindStringSubmatch(l); len(m) > 0 {
			if len(m) > 1 && m[1] != "" {
				return strings.TrimSpace(m[1])
			}
			return "void"
		}
	}

	// 3. Check included interface: unit <unitName> includes <IfaceName>
	reUnitIncludes := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\s+(?:includes|extends)\s+([a-zA-Z0-9_]+)`, regexp.QuoteMeta(unitName)))
	var ifaceName string
	for _, l := range lines {
		if m := reUnitIncludes.FindStringSubmatch(l); len(m) > 1 {
			ifaceName = m[1]
			break
		}
	}
	if ifaceName != "" {
		reIfaceStart := regexp.MustCompile(fmt.Sprintf(`^\s*interface\s+%s\b`, regexp.QuoteMeta(ifaceName)))
		reIfaceFunc := regexp.MustCompile(fmt.Sprintf(`^\s*func\s+%s\s*\(.*?\)(?:\s*->\s*(.+))?`, regexp.QuoteMeta(methodName)))
		inIface := false
		iDepth := 0
		for _, l := range lines {
			trimmed := strings.TrimSpace(l)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			if !inIface {
				if reIfaceStart.MatchString(l) {
					inIface = true
					iDepth += strings.Count(l, "{") - strings.Count(l, "}")
				}
				continue
			}
			iDepth += strings.Count(l, "{") - strings.Count(l, "}")
			if iDepth <= 0 {
				break
			}
			if m := reIfaceFunc.FindStringSubmatch(l); len(m) > 0 {
				if len(m) > 1 && m[1] != "" {
					return strings.TrimSpace(m[1])
				}
				return "void"
			}
		}
	}

	return "any"
}

func isNumericLiteral(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	re := regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)
	return re.MatchString(s)
}

func getDateMapFields() []CompletionItem {
	return []CompletionItem{
		{
			Label:            "iso",
			Kind:             CompletionItemKindProperty,
			Detail:           "iso: string",
			Documentation:    MarkupContent{Kind: "markdown", Value: "ISO-8601 formatted date string (e.g. `\"2026-10-15T12:00:00Z\"`)."},
			InsertText:       "iso",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "00_iso",
		},
		{
			Label:            "timestamp",
			Kind:             CompletionItemKindProperty,
			Detail:           "timestamp: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Unix timestamp in milliseconds."},
			InsertText:       "timestamp",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "01_timestamp",
		},
		{
			Label:            "year",
			Kind:             CompletionItemKindProperty,
			Detail:           "year: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Year component as a 4-digit number (e.g. `2026`)."},
			InsertText:       "year",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "02_year",
		},
		{
			Label:            "month",
			Kind:             CompletionItemKindProperty,
			Detail:           "month: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Month component from 1 to 12."},
			InsertText:       "month",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "03_month",
		},
		{
			Label:            "day",
			Kind:             CompletionItemKindProperty,
			Detail:           "day: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Day of the month from 1 to 31."},
			InsertText:       "day",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "04_day",
		},
		{
			Label:            "hour",
			Kind:             CompletionItemKindProperty,
			Detail:           "hour: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Hour component from 0 to 23."},
			InsertText:       "hour",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "05_hour",
		},
		{
			Label:            "minute",
			Kind:             CompletionItemKindProperty,
			Detail:           "minute: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Minute component from 0 to 59."},
			InsertText:       "minute",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "06_minute",
		},
		{
			Label:            "second",
			Kind:             CompletionItemKindProperty,
			Detail:           "second: number",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Second component from 0 to 59."},
			InsertText:       "second",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "07_second",
		},
		{
			Label:            "weekday",
			Kind:             CompletionItemKindProperty,
			Detail:           "weekday: string",
			Documentation:    MarkupContent{Kind: "markdown", Value: "Day of the week name (e.g. `\"Monday\"`, `\"Thursday\"`)."},
			InsertText:       "weekday",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "08_weekday",
		},
	}
}

func resolveExpressionTargetInfo(uri string, lines []string, expr string) *varTargetInfo {
	return resolveExpressionTargetInfoDepth(uri, lines, expr, 0)
}

func resolveExpressionTargetInfoDepth(uri string, lines []string, expr string, depth int) *varTargetInfo {
	if depth > 10 {
		return nil
	}
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil
	}

	// 1. String literal
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") && len(expr) >= 2) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'") && len(expr) >= 2) ||
		strings.HasPrefix(expr, "$f\"") {
		return &varTargetInfo{isString: true}
	}

	// 2. Numeric literal
	if isNumericLiteral(expr) {
		return &varTargetInfo{isNumber: true}
	}

	// 3. List literal
	if strings.HasPrefix(expr, "[") && strings.HasSuffix(expr, "]") {
		return &varTargetInfo{isList: true}
	}

	// 4. Map literal: { a: 1 } or @map{ a: 1 }
	if (strings.HasPrefix(expr, "{") && strings.HasSuffix(expr, "}")) ||
		(strings.HasPrefix(expr, "@map{") && strings.HasSuffix(expr, "}")) ||
		(strings.HasPrefix(expr, "@map {") && strings.HasSuffix(expr, "}")) {
		fields := extractMapLiteralFields(expr)
		return &varTargetInfo{isMap: true, mapFields: fields}
	}

	// 5. Method or Function Call ending with ')'
	if strings.HasSuffix(expr, ")") {
		parenDepth := 0
		openParen := -1
		for i := len(expr) - 1; i >= 0; i-- {
			if expr[i] == ')' {
				parenDepth++
			} else if expr[i] == '(' {
				parenDepth--
				if parenDepth == 0 {
					openParen = i
					break
				}
			}
		}

		if openParen > 0 {
			caller := strings.TrimSpace(expr[:openParen])
			lastDot := findLastDotOutsideParens(caller)
			var callerObj, callerMethod string
			if lastDot != -1 {
				callerObj = strings.TrimSpace(caller[:lastDot])
				callerMethod = strings.TrimSpace(caller[lastDot+1:])
			} else if colonIdx := strings.LastIndex(caller, "::"); colonIdx != -1 {
				callerObj = strings.TrimSpace(caller[:colonIdx])
				callerMethod = strings.TrimSpace(caller[colonIdx+2:])
			}

			if callerObj != "" && callerMethod != "" {
				// 0. Unit or Self method call (e.g. path.clean(p), &date::now(), &self::now(), &path::clean(p))
				cleanUnit := strings.TrimPrefix(callerObj, "&")
				unitRetType := inferUnitMethodReturnType(lines, cleanUnit, callerMethod)
				if unitRetType != "" && unitRetType != "void" && unitRetType != "any" {
					if info := typeStringToTargetInfo(lines, unitRetType, uriToPath(uri)); info != nil {
						return info
					}
				}

				// Special Built-in: date
				if strings.EqualFold(callerObj, "date") {
					mLower := strings.ToLower(callerMethod)
					if mLower == "now" || mLower == "parse" || mLower == "add" {
						return &varTargetInfo{isMap: true, mapFields: getDateMapFields()}
					}
					if mLower == "format" {
						return &varTargetInfo{isString: true}
					}
					if mLower == "diff" || mLower == "timestamp" {
						return &varTargetInfo{isNumber: true}
					}
				}
				// Special Built-in: math
				if strings.EqualFold(callerObj, "math") {
					return &varTargetInfo{isNumber: true}
				}
				// Special Built-in: net.fetch
				if strings.EqualFold(callerObj, "net") && strings.EqualFold(callerMethod, "fetch") {
					return &varTargetInfo{isString: true}
				}
				// Special Built-in: crypto
				if strings.EqualFold(callerObj, "crypto") {
					return &varTargetInfo{isString: true}
				}

				// General Object Method Call: resolve callerObj recursively
				objInfo := resolveExpressionTargetInfoDepth(uri, lines, callerObj, depth+1)
				if objInfo != nil {
					mLower := strings.ToLower(callerMethod)
					if objInfo.isString {
						switch mLower {
						case "len":
							return &varTargetInfo{isNumber: true}
						case "split":
							return &varTargetInfo{isList: true, listElemType: "string"}
						case "contains", "startswith", "endswith":
							return nil
						case "tostring", "to_string":
							return &varTargetInfo{isString: true}
						default: // trim, toupper, tolower, replace, slice, etc.
							return &varTargetInfo{isString: true}
						}
					}
					if objInfo.isNumber {
						switch mLower {
						case "tostring", "to_string":
							return &varTargetInfo{isString: true}
						default:
							return &varTargetInfo{isNumber: true}
						}
					}
					if objInfo.isList {
						switch mLower {
						case "len":
							return &varTargetInfo{isNumber: true}
						case "join":
							return &varTargetInfo{isString: true}
						case "get", "at", "pop":
							if objInfo.listElemType != "" {
								return typeStringToTargetInfo(lines, objInfo.listElemType, uriToPath(uri))
							}
							return &varTargetInfo{}
						default:
							return &varTargetInfo{isList: true, listElemType: objInfo.listElemType}
						}
					}
					if objInfo.isTask {
						switch mLower {
						case "wait", "result":
							retType := objInfo.taskReturnType
							if retType != "" && retType != "any" {
								return typeStringToTargetInfo(lines, retType, uriToPath(uri))
							}
						}
					}
					if objInfo.unitType != "" {
						retType := inferUnitMethodReturnType(lines, objInfo.unitType, callerMethod)
						if retType != "" && retType != "void" && retType != "any" {
							return typeStringToTargetInfo(lines, retType, uriToPath(uri))
						}
					}
					if objInfo.isMap {
						switch mLower {
						case "len":
							return &varTargetInfo{isNumber: true}
						case "keys", "values":
							return &varTargetInfo{isList: true}
						}
					}
				}
			} else {
				// Standalone function call: funcName(...)
				retType := findFunctionReturnType(lines, caller)
				if retType != "" && retType != "void" && retType != "any" {
					return typeStringToTargetInfo(lines, retType, uriToPath(uri))
				}
			}
		}
	}

	// 6. Chained Property Access (e.g. date.now().iso or obj.prop)
	lastDot := findLastDotOutsideParens(expr)
	if lastDot != -1 {
		parentExpr := strings.TrimSpace(expr[:lastDot])
		propName := strings.TrimSpace(expr[lastDot+1:])
		parentInfo := resolveExpressionTargetInfoDepth(uri, lines, parentExpr, depth+1)
		if parentInfo != nil {
			if parentInfo.isMap && len(parentInfo.mapFields) > 0 {
				for _, f := range parentInfo.mapFields {
					if f.Label == propName {
						return detailToTargetInfo(lines, f.Detail, uriToPath(uri))
					}
				}
			}
			if parentInfo.interfaceName != "" {
				ifaceMembers := collectInterfaceMembers(lines, parentInfo.interfaceName, uriToPath(uri))
				for _, m := range ifaceMembers {
					if m.Label == propName {
						return detailToTargetInfo(lines, m.Detail, uriToPath(uri))
					}
				}
			}
			if parentInfo.unitType != "" {
				unitMembers := collectUnitMembers(lines, parentInfo.unitType)
				for _, m := range unitMembers {
					if m.Label == propName {
						return detailToTargetInfo(lines, m.Detail, uriToPath(uri))
					}
				}
			}
		}
	}

	// 7. Single Identifier (local variable, parameter, etc.)
	if isIdentifier(expr) {
		if varInfo := findVariableTargetInfo(lines, expr); varInfo != nil {
			return varInfo
		}
	}

	return nil
}

func findLastDotOutsideParens(s string) int {
	lastDot := -1
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	inString := false
	var quoteChar byte

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inString {
			if ch == quoteChar && (i == 0 || s[i-1] != '\\') {
				inString = false
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inString = true
			quoteChar = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{':
			braceDepth++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
		case '.':
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				lastDot = i
			}
		}
	}
	return lastDot
}

func findFunctionReturnType(lines []string, funcName string) string {
	reFunc := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?func\s+%s\s*\(.*?\)(?:\s*->\s*([^\s{]+))?`, regexp.QuoteMeta(funcName)))
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if m := reFunc.FindStringSubmatch(trimmed); len(m) > 1 {
			if m[1] != "" {
				return strings.TrimSpace(m[1])
			}
			return "void"
		}
	}
	return ""
}

func typeStringToTargetInfo(lines []string, tStr string, filePath string) *varTargetInfo {
	tStr = strings.TrimSpace(tStr)
	if tStr == "string" {
		return &varTargetInfo{isString: true}
	}
	if tStr == "number" || tStr == "duration" {
		return &varTargetInfo{isNumber: true}
	}
	if strings.HasPrefix(tStr, "list") {
		elemT := ""
		if strings.HasPrefix(tStr, "list[") && strings.HasSuffix(tStr, "]") {
			elemT = strings.TrimSuffix(strings.TrimPrefix(tStr, "list["), "]")
		}
		return &varTargetInfo{isList: true, listElemType: elemT}
	}
	if strings.HasPrefix(tStr, "@map") || strings.HasPrefix(tStr, "{") {
		return &varTargetInfo{isMap: true, mapFields: parseInlineMapFields(tStr)}
	}
	if tStr == "map" {
		return &varTargetInfo{isMap: true}
	}
	if strings.HasPrefix(tStr, "task[") && strings.HasSuffix(tStr, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(tStr, "task["), "]")
		return &varTargetInfo{isTask: true, taskReturnType: inner}
	}
	if tStr == "task" {
		return &varTargetInfo{isTask: true, taskReturnType: "any"}
	}
	if len(collectInterfaceMembers(lines, tStr, filePath)) > 0 {
		return &varTargetInfo{interfaceName: tStr}
	}
	if len(collectUnitMembers(lines, tStr)) > 0 {
		return &varTargetInfo{unitType: tStr}
	}
	return nil
}

func detailToTargetInfo(lines []string, detail string, filePath string) *varTargetInfo {
	detail = strings.TrimSpace(detail)
	colonIdx := strings.LastIndex(detail, ":")
	if colonIdx == -1 {
		return nil
	}
	tStr := strings.TrimSpace(detail[colonIdx+1:])
	return typeStringToTargetInfo(lines, tStr, filePath)
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && r != '_' {
				return false
			}
		} else {
			if !isWordChar(byte(r)) {
				return false
			}
		}
	}
	return true
}


