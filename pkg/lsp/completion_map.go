package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// handleMapLiteralCompletion proposes typed keys for a map literal,
// filtering out any properties that have already been declared in the current map.
func (s *Server) handleMapLiteralCompletion(uri string, lines []string, pos Position) ([]CompletionItem, bool) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return nil, false
	}
	currentLine := lines[pos.Line]
	col := pos.Character
	if col > len(currentLine) {
		col = len(currentLine)
	}
	prefixLine := currentLine[:col]

	// If the cursor is already after a colon on this line, the user is typing the VALUE, not the key.
	if strings.Contains(prefixLine, ":") {
		return nil, false
	}

	// 1. Determine if the cursor is inside a @map { ... } literal
	// Scan backwards from pos to find the opening of the @map
	depth := 0
	for _, r := range prefixLine {
		if r == '{' {
			depth++
		} else if r == '}' {
			depth--
		}
	}

	mapStartLine := -1
	if depth > 0 && strings.Contains(prefixLine, "@map") {
		mapStartLine = pos.Line
	} else {
		for l := pos.Line - 1; l >= 0; l-- {
			line := lines[l]
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}
			for _, r := range line {
				if r == '{' {
					depth++
				} else if r == '}' {
					depth--
				}
			}
			if depth > 0 {
				if strings.Contains(line, "@map") {
					mapStartLine = l
					break
				}
				if l > 0 && strings.Contains(lines[l-1], "@map") {
					mapStartLine = l - 1
					break
				}
				// Not a map literal
				return nil, false
			}
		}
	}

	if mapStartLine == -1 {
		return nil, false
	}

	// 2. Resolve the expected type for this map literal
	expectedType := resolveExpectedMapType(lines, mapStartLine, uriToPath(uri))
	if expectedType == "" {
		return nil, false
	}

	// 3. Collect available properties based on expectedType
	var availableMembers []CompletionItem
	if strings.HasPrefix(expectedType, "@map") && strings.Contains(expectedType, "{") {
		availableMembers = parseInlineMapFields(expectedType)
	} else {
		availableMembers = collectInterfaceMembers(lines, expectedType, uriToPath(uri))
	}

	if len(availableMembers) == 0 {
		return nil, false
	}

	// 4. Scan all lines in this map literal to collect ALREADY DECLARED keys
	usedKeys := make(map[string]bool)
	scanDepth := 0
	reField := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*|"(?:[^"\\]|\\.)*")\s*(?::|->)`)

	for l := mapStartLine; l < len(lines); l++ {
		line := lines[l]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		for _, r := range line {
			if r == '{' {
				scanDepth++
			} else if r == '}' {
				scanDepth--
			}
		}

		// Direct fields of this map literal
		if scanDepth == 1 || (scanDepth == 2 && strings.Contains(line, "{")) {
			if l == pos.Line {
				// Don't count the property being actively completed
				continue
			}
			if m := reField.FindStringSubmatch(trimmed); len(m) > 1 {
				k := strings.Trim(m[1], "\"")
				usedKeys[k] = true
			}
		}

		if scanDepth <= 0 {
			break
		}
	}

	// 5. Filter out already used keys ("quand une donnée a été déjà utilisée bah faut plus la suggérer")
	var items []CompletionItem
	for _, m := range availableMembers {
		if usedKeys[m.Label] {
			continue
		}
		items = append(items, CompletionItem{
			Label:            m.Label,
			Kind:             m.Kind,
			Detail:           m.Detail,
			Documentation:    m.Documentation,
			InsertText:       m.Label + ": ",
			FilterText:       m.Label,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("00_%s", m.Label),
		})
	}

	if len(items) > 0 {
		return items, true
	}

	return nil, false
}

// resolveExpectedMapType inspects the declaration preceding a map literal to deduce its expected type.
func resolveExpectedMapType(lines []string, mapStartLine int, currentFilePath string) string {
	if mapStartLine < 0 || mapStartLine >= len(lines) {
		return ""
	}
	line := lines[mapStartLine]

	// 1. Variable or argument declaration: `let x: TExample = @map{` or `const data: TExample = {`
	reVar := regexp.MustCompile(`(?:let|const|var|arg)\s+[a-zA-Z_][a-zA-Z0-9_]*\s*:\s*([^=]+?)\s*=\s*(?:@map)?\s*\{?`)
	if m := reVar.FindStringSubmatch(line); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}

	// Also check if type annotation was on preceding line:
	// let x: TExample =
	//     @map{
	if mapStartLine > 0 {
		prevLine := lines[mapStartLine-1]
		if m := reVar.FindStringSubmatch(prevLine + " " + line); len(m) > 1 {
			return strings.TrimSpace(m[1])
		}
	}

	// 2. Nested property in parent map: `d: @map {` or `d: {`
	reProp := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:\s*(?:@map)?\s*\{`)
	if m := reProp.FindStringSubmatch(line); len(m) > 1 {
		propName := m[1]
		// Find parent map's type by looking up further
		parentDepth := 0
		for l := mapStartLine - 1; l >= 0; l-- {
			pLine := lines[l]
			for _, r := range pLine {
				if r == '{' {
					parentDepth++
				} else if r == '}' {
					parentDepth--
				}
			}
			if parentDepth > 0 && strings.Contains(pLine, "@map") {
				parentType := resolveExpectedMapType(lines, l, currentFilePath)
				if parentType != "" {
					// Look up propName in parentType
					var parentMembers []CompletionItem
					if strings.HasPrefix(parentType, "@map") && strings.Contains(parentType, "{") {
						parentMembers = parseInlineMapFields(parentType)
					} else {
						parentMembers = collectInterfaceMembers(lines, parentType, currentFilePath)
					}
					for _, pm := range parentMembers {
						if pm.Label == propName {
							// Detail usually contains "propName: type"
							colonIdx := strings.Index(pm.Detail, ":")
							if colonIdx != -1 {
								return strings.TrimSpace(pm.Detail[colonIdx+1:])
							}
						}
					}
				}
				break
			}
		}
	}

	// 3. Function or method call argument: `Sys.require("<= 2.0.0", @map{` or multiline
	depthParen := 0
	foundCall := false
	callStartLine := -1
	for l := mapStartLine; l >= 0 && l >= mapStartLine-15; l-- {
		lineStr := lines[l]
		trimmedL := strings.TrimSpace(lineStr)
		if strings.HasPrefix(trimmedL, "//") {
			continue
		}
		for i := len(lineStr) - 1; i >= 0; i-- {
			r := lineStr[i]
			if r == ')' {
				depthParen--
			} else if r == '(' {
				depthParen++
				if depthParen > 0 {
					foundCall = true
					callStartLine = l
					break
				}
			}
		}
		if foundCall {
			break
		}
	}

	if foundCall && callStartLine != -1 {
		var callText strings.Builder
		for l := callStartLine; l <= mapStartLine; l++ {
			callText.WriteString(lines[l])
			callText.WriteString(" ")
		}
		callStr := callText.String()
		openParenIdx := strings.Index(callStr, "(")
		if openParenIdx != -1 {
			callerPart := strings.TrimSpace(callStr[:openParenIdx])
			reCaller := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_.]*)$`)
			if cm := reCaller.FindStringSubmatch(callerPart); len(cm) > 1 {
				callerExpr := cm[1]
				mapIdx := strings.Index(callStr, "@map")
				if mapIdx == -1 {
					mapIdx = strings.Index(callStr, "{")
				}
				if mapIdx > openParenIdx {
					argSection := callStr[openParenIdx+1 : mapIdx]
					argIndex := 0
					dBrace := 0
					dParen := 0
					inStr := false
					for _, ch := range argSection {
						if ch == '"' {
							inStr = !inStr
						}
						if !inStr {
							if ch == '(' {
								dParen++
							} else if ch == ')' {
								dParen--
							} else if ch == '{' {
								dBrace++
							} else if ch == '}' {
								dBrace--
							} else if ch == ',' && dParen == 0 && dBrace == 0 {
								argIndex++
							}
						}
					}
					if expected := resolveFunctionCallArgType(lines, callerExpr, argIndex, currentFilePath); expected != "" {
						return expected
					}
				}
			}
		}
	}

	// 4. Return statement: `return @map{`
	if strings.Contains(line, "return") {
		// Look up enclosing function
		for l := mapStartLine - 1; l >= 0; l-- {
			reFunc := regexp.MustCompile(`^\s*func\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\(.*?\)(?:\s*(?:->|:)\s*([^{]+))?`)
			if fm := reFunc.FindStringSubmatch(lines[l]); len(fm) > 1 {
				return strings.TrimSpace(fm[1])
			}
		}
	}

	return ""
}

// resolveFunctionCallArgType inspects a function or method call to determine the expected parameter type at argIndex.
func resolveFunctionCallArgType(lines []string, callerExpr string, argIndex int, currentFilePath string) string {
	callerExpr = strings.TrimSpace(callerExpr)
	if callerExpr == "" {
		return ""
	}

	var receiver, member string
	if dotIdx := strings.LastIndex(callerExpr, "."); dotIdx != -1 {
		receiver = strings.TrimSpace(callerExpr[:dotIdx])
		member = strings.TrimSpace(callerExpr[dotIdx+1:])
	} else if colonIdx := strings.LastIndex(callerExpr, "::"); colonIdx != -1 {
		receiver = strings.TrimSpace(callerExpr[:colonIdx])
		member = strings.TrimSpace(callerExpr[colonIdx+2:])
	} else {
		member = callerExpr
	}

	extractTypeFromSignature := func(sigLines []string, funcName string) string {
		reFunc := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:func\s+)?%s\s*\((.*?)\)`, regexp.QuoteMeta(funcName)))
		fullText := strings.Join(sigLines, "\n")
		matches := reFunc.FindAllStringSubmatch(fullText, -1)
		for _, m := range matches {
			if len(m) > 1 {
				params := splitParamsRespectingBraces(m[1])
				if argIndex < len(params) {
					p := params[argIndex]
					colonIdx := strings.Index(p, ":")
					if colonIdx != -1 {
						pType := strings.TrimSpace(p[colonIdx+1:])
						if eqIdx := strings.Index(pType, "="); eqIdx != -1 {
							pType = strings.TrimSpace(pType[:eqIdx])
						}
						return strings.TrimSuffix(pType, "?")
					}
					// If no colon: paramName? Type
					reNoColon := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\??\s+(.+)`)
					if nm := reNoColon.FindStringSubmatch(p); len(nm) > 1 {
						pType := strings.TrimSpace(nm[1])
						if eqIdx := strings.Index(pType, "="); eqIdx != -1 {
							pType = strings.TrimSpace(pType[:eqIdx])
						}
						return strings.TrimSuffix(pType, "?")
					}
				}
			}
		}
		return ""
	}

	curDir := ""
	if currentFilePath != "" {
		curDir = filepath.Dir(currentFilePath)
	}

	if receiver != "" {
		// 1. Check if receiver matches an import: use sys as Sys, use date, from "..." use mod as alias
		fullDoc := strings.Join(lines, "\n")
		imports := parseFileImports(fullDoc)
		targetModule := receiver
		for _, imp := range imports {
			if imp.Alias == receiver {
				targetModule = imp.Module
				break
			}
			if imp.Module == receiver {
				targetModule = imp.Module
				break
			}
		}

		// 2. Check stdlib for targetModule (e.g. sys.xru)
		if stdDir := FindStdDir(curDir); stdDir != "" {
			stdFile := filepath.Join(stdDir, strings.ToLower(targetModule)+".xru")
			if content, err := os.ReadFile(stdFile); err == nil {
				fLines := strings.Split(string(content), "\n")
				if res := extractTypeFromSignature(fLines, member); res != "" {
					return res
				}
			}
		}

		// 3. Check variable type of receiver
		if vInfo := findVariableTargetInfo(lines, receiver); vInfo != nil {
			targetType := vInfo.interfaceName
			if targetType == "" {
				targetType = vInfo.unitType
			}
			if targetType != "" {
				if stdDir := FindStdDir(curDir); stdDir != "" {
					stdEntries, sErr := os.ReadDir(stdDir)
					if sErr == nil {
						for _, entry := range stdEntries {
							if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
								c, _ := os.ReadFile(filepath.Join(stdDir, entry.Name()))
								fLines := strings.Split(string(c), "\n")
								if res := extractTypeFromSignature(fLines, member); res != "" {
									return res
								}
							}
						}
					}
				}
			}
		}
	} else {
		// Receiver is empty: top-level function in current file or std
		if res := extractTypeFromSignature(lines, member); res != "" {
			return res
		}
		if stdDir := FindStdDir(curDir); stdDir != "" {
			if stdEntries, err := os.ReadDir(stdDir); err == nil {
				for _, entry := range stdEntries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
						content, _ := os.ReadFile(filepath.Join(stdDir, entry.Name()))
						fLines := strings.Split(string(content), "\n")
						if res := extractTypeFromSignature(fLines, member); res != "" {
							return res
						}
					}
				}
			}
		}
	}

	return ""
}

// handleMapValueCompletion proposes literal values for typed map properties (e.g. "log", "silent" for ROpt.mode).
func (s *Server) handleMapValueCompletion(uri string, lines []string, pos Position) ([]CompletionItem, bool) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return nil, false
	}
	curLine := lines[pos.Line]
	col := pos.Character
	if col > len(curLine) {
		col = len(curLine)
	}
	prefixLine := curLine[:col]

	// Must be after colon on this line (not double colon ::)
	colonIdx := -1
	for i := 0; i < len(prefixLine); i++ {
		if prefixLine[i] == ':' {
			if (i > 0 && prefixLine[i-1] == ':') || (i+1 < len(prefixLine) && prefixLine[i+1] == ':') {
				continue
			}
			colonIdx = i
		}
	}
	if colonIdx == -1 {
		return nil, false
	}

	// Extract property key before colon
	reKey := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*|"(?:[^"\\]|\\.)*")\s*:`)
	match := reKey.FindStringSubmatch(curLine[:colonIdx+1])
	if len(match) < 2 {
		return nil, false
	}
	keyName := strings.Trim(match[1], "\"")

	// Verify we are inside a map literal block
	depth := 0
	mapStartLine := -1
	for l := pos.Line; l >= 0; l-- {
		line := lines[l]
		for _, r := range line {
			if r == '{' {
				depth++
			} else if r == '}' {
				depth--
			}
		}
		if depth > 0 && (strings.Contains(line, "@map") || strings.Contains(line, "=") || strings.Contains(line, "(")) {
			trimmedL := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedL, "interface ") || strings.HasPrefix(trimmedL, "type ") || strings.HasPrefix(trimmedL, "unit ") {
				return nil, false
			}
			mapStartLine = l
			break
		}
	}
	if mapStartLine == -1 {
		return nil, false
	}

	// Resolve expected type of map literal
	filePath := uriToPath(uri)
	expectedType := resolveExpectedMapType(lines, mapStartLine, filePath)
	if expectedType == "" {
		return nil, false
	}

	// Find property declaration
	propInfo := findPropertyDeclaration(lines, expectedType, keyName, uri)
	if propInfo == nil || propInfo.PropType == "" {
		return nil, false
	}

	propType := strings.TrimSpace(propInfo.PropType)
	var items []CompletionItem

	afterColon := strings.TrimSpace(prefixLine[colonIdx+1:])
	hasQuote := strings.HasPrefix(afterColon, "\"") || strings.HasPrefix(afterColon, "'")

	// Union of options: "log" or "silent" or "fatal" or "warn" or "auto-update"
	if strings.Contains(propType, " or ") {
		parts := strings.Split(propType, " or ")
		for _, part := range parts {
			val := strings.TrimSpace(part)
			if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") {
				insertText := val
				if hasQuote {
					insertText = strings.Trim(val, "\"")
				}
				items = append(items, CompletionItem{
					Label:            val,
					Kind:             CompletionItemKindValue,
					Detail:           fmt.Sprintf("option %s", val),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Allowed value for `%s.%s`", expectedType, keyName)},
					InsertText:       insertText,
					FilterText:       strings.Trim(val, "\""),
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("00_%s", val),
				})
			} else if _, err := strconv.ParseFloat(val, 64); err == nil {
				items = append(items, CompletionItem{
					Label:            val,
					Kind:             CompletionItemKindValue,
					Detail:           fmt.Sprintf("number %s", val),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Allowed number value for `%s.%s`", expectedType, keyName)},
					InsertText:       val,
					FilterText:       val,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("00_%s", val),
				})
			}
		}
	} else if propType == "bool" || propType == "boolean" {
		items = append(items,
			CompletionItem{
				Label:            "true",
				Kind:             CompletionItemKindKeyword,
				Detail:           "boolean true",
				InsertText:       "true",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "00_true",
			},
			CompletionItem{
				Label:            "false",
				Kind:             CompletionItemKindKeyword,
				Detail:           "boolean false",
				InsertText:       "false",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "00_false",
			},
		)
	}

	if len(items) > 0 {
		return items, true
	}
	return nil, false
}
