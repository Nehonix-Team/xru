package lsp

import (
	"fmt"
	"regexp"
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

	// 3. Function call argument: `name(@map{` or `query(@map{`
	reCall := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*?)(?:@map)`)
	if m := reCall.FindStringSubmatch(line); len(m) > 2 {
		funcName := m[1]
		beforeMap := m[2]
		argIndex := strings.Count(beforeMap, ",")

		// Find funcName definition in lines
		reFunc := regexp.MustCompile(fmt.Sprintf(`^\s*func\s+%s\s*\((.*?)\)`, regexp.QuoteMeta(funcName)))
		for _, l := range lines {
			if fm := reFunc.FindStringSubmatch(l); len(fm) > 1 {
				params := splitParamsRespectingBraces(fm[1])
				if argIndex < len(params) {
					p := params[argIndex]
					colonIdx := strings.Index(p, ":")
					if colonIdx != -1 {
						pType := strings.TrimSpace(p[colonIdx+1:])
						return strings.TrimSuffix(pType, "?")
					}
				}
				break
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
