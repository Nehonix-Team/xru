package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FindPropertyDeclaration searches for the declaration of propName in targetType across lines, local files, and stdlib.
func (n *Navigator) FindPropertyDeclaration(lines []string, targetType, propName string, uri string) *PropertyInfo {
	if targetType == "" || propName == "" {
		return nil
	}
	cleanType := strings.TrimSpace(strings.TrimSuffix(targetType, "?"))

	// 1. If targetType is an inline map type: @map{ a: string, ... }
	if strings.HasPrefix(cleanType, "@map") && strings.Contains(cleanType, "{") {
		reField := regexp.MustCompile(fmt.Sprintf(`\b(%s)(\??)(?:\s*:\s*|\s+)([^,\n}]+)`, regexp.QuoteMeta(propName)))
		if m := reField.FindStringSubmatch(cleanType); len(m) > 3 {
			pType := strings.TrimSpace(m[3])
			return &PropertyInfo{
				TypeName:   "@map",
				PropName:   propName,
				PropType:   pType,
				IsOptional: m[2] == "?",
				IsMethod:   strings.HasPrefix(pType, "func(") || strings.HasPrefix(pType, "func "),
				URI:        uri,
			}
		}
	}

	// 2. Scan active file
	localTypeFound := false
	reIface := regexp.MustCompile(fmt.Sprintf(`(?mi)^\s*(?:interface|type|unit)\s+(?:%s|%sNative)\b`, regexp.QuoteMeta(cleanType), regexp.QuoteMeta(cleanType)))
	for _, l := range lines {
		if reIface.MatchString(l) {
			localTypeFound = true
			break
		}
	}

	if info := ScanLinesForPropertyDeclaration(lines, cleanType, propName, uri); info != nil {
		return info
	}

	if localTypeFound {
		return nil
	}

	// 3. Scan workspace files in current directory
	curFilePath := uriToPath(uri)
	if curFilePath != "" {
		fileDir := filepath.Dir(curFilePath)
		entries, err := os.ReadDir(fileDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
					fp := filepath.Join(fileDir, entry.Name())
					if fp == curFilePath {
						continue
					}
					if content, rErr := os.ReadFile(fp); rErr == nil {
						fLines := strings.Split(string(content), "\n")
						for _, fl := range fLines {
							if reIface.MatchString(fl) {
								localTypeFound = true
								break
							}
						}
						if info := ScanLinesForPropertyDeclaration(fLines, cleanType, propName, pathToURI(fp)); info != nil {
							return info
						}
					}
				}
			}
		}

		if localTypeFound {
			return nil
		}

		// 4. Scan stdlib files
		if n.FindStdDir != nil {
			if stdDir := n.FindStdDir(fileDir); stdDir != "" {
				stdEntries, sErr := os.ReadDir(stdDir)
				if sErr == nil {
					for _, entry := range stdEntries {
						if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
							fp := filepath.Join(stdDir, entry.Name())
							if content, rErr := os.ReadFile(fp); rErr == nil {
								fLines := strings.Split(string(content), "\n")
								if info := ScanLinesForPropertyDeclaration(fLines, cleanType, propName, pathToURI(fp)); info != nil {
									return info
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// ScanLinesForPropertyDeclaration searches for the declaration of propName in cleanType inside lines.
func ScanLinesForPropertyDeclaration(lines []string, cleanType, propName, uri string) *PropertyInfo {
	reIfaceStart := regexp.MustCompile(fmt.Sprintf(`(?mi)^\s*(?:interface|type|unit)\s+(?:%s|%sNative)\b`, regexp.QuoteMeta(cleanType), regexp.QuoteMeta(cleanType)))
	reMethod := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+)?func\s+(%s)\s*\(`, regexp.QuoteMeta(propName)))
	reField := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:func\s+)?(%s)(\??)(?:\s*:\s*|\s+)([^/\n\r;]+?)(?:;|\s*//(.*))?$`, regexp.QuoteMeta(propName)))

	inTargetIface := false
	braceDepth := 0

	checkLine := func(idx int, line string) *PropertyInfo {
		// 1. Check method: func methodName(...)
		if matchLoc := reMethod.FindStringSubmatchIndex(line); matchLoc != nil && len(matchLoc) >= 4 {
			propType := "func"
			if parenIdx := strings.Index(line, "("); parenIdx != -1 {
				end := len(line)
				if cut := strings.IndexAny(line[parenIdx:], "{;"); cut != -1 {
					end = parenIdx + cut
				}
				propType = strings.TrimSpace(line[parenIdx:end])
			}
			aboveDoc := ExtractDocCommentForLine(lines, idx)
			return &PropertyInfo{
				TypeName:  cleanType,
				PropName:  propName,
				PropType:  propType,
				IsMethod:  true,
				Doc:       aboveDoc,
				Line:      idx,
				CharStart: matchLoc[2],
				CharEnd:   matchLoc[3],
				URI:       uri,
			}
		}

		// 2. Check field: propName: Type or propName?: Type
		if matchLoc := reField.FindStringSubmatchIndex(line); matchLoc != nil && len(matchLoc) >= 4 {
			m := reField.FindStringSubmatch(line)
			propType := strings.TrimSpace(m[3])
			trailingDoc := ""
			if len(m) > 4 {
				trailingDoc = strings.TrimSpace(m[4])
			}
			aboveDoc := ExtractDocCommentForLine(lines, idx)
			doc := aboveDoc
			if trailingDoc != "" {
				if doc != "" {
					doc += "\n\n" + trailingDoc
				} else {
					doc = trailingDoc
				}
			}
			return &PropertyInfo{
				TypeName:   cleanType,
				PropName:   propName,
				PropType:   propType,
				IsOptional: m[2] == "?",
				IsMethod:   strings.HasPrefix(propType, "func(") || strings.HasPrefix(propType, "func "),
				Doc:        doc,
				Line:       idx,
				CharStart:  matchLoc[2],
				CharEnd:    matchLoc[3],
				URI:        uri,
			}
		}
		return nil
	}

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		if !inTargetIface {
			if reIfaceStart.MatchString(line) {
				inTargetIface = true
				braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
				if info := checkLine(idx, line); info != nil {
					return info
				}
				if braceDepth == 0 && strings.Contains(line, "{") {
					inTargetIface = false
				}
			}
		} else {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if info := checkLine(idx, line); info != nil {
				return info
			}
			if braceDepth <= 0 {
				inTargetIface = false
			}
		}
	}
	return nil
}

// IsCursorOnMapKey checks whether pos is on a key identifier inside a @map { ... } or object literal block.
func IsCursorOnMapKey(lines []string, pos Position) (bool, string, int) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return false, "", -1
	}
	curLine := lines[pos.Line]
	trimmed := strings.TrimSpace(curLine)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false, "", -1
	}

	// 1. Must have a key-value colon on this line (excluding ::)
	colonIdx := -1
	for i := 0; i < len(curLine); i++ {
		if curLine[i] == ':' {
			if (i > 0 && curLine[i-1] == ':') || (i+1 < len(curLine) && curLine[i+1] == ':') {
				continue
			}
			colonIdx = i
			break
		}
	}
	if colonIdx == -1 {
		return false, "", -1
	}

	// 2. Cursor MUST be before or on the colon (key position)
	if pos.Character > colonIdx {
		return false, "", -1
	}

	// 3. Extract the key before colon
	reKey := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*|"(?:[^"\\]|\\.)*")\s*:`)
	matchLoc := reKey.FindStringSubmatchIndex(curLine[:colonIdx+1])
	if matchLoc == nil || len(matchLoc) < 4 {
		return false, "", -1
	}
	keyStart := matchLoc[2]
	keyEnd := matchLoc[3]
	if pos.Character < keyStart || pos.Character > keyEnd {
		return false, "", -1
	}
	keyWord := strings.Trim(curLine[keyStart:keyEnd], "\"")

	// 4. Verify we are inside a map literal block @map { ... } or { ... }
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
				return false, "", -1
			}
			mapStartLine = l
			break
		}
	}

	if mapStartLine == -1 {
		return false, "", -1
	}

	return true, keyWord, mapStartLine
}

// FindMemberInInterface returns the Location of memberName in ifaceName if found.
func (n *Navigator) FindMemberInInterface(lines []string, ifaceName, memberName string, uri string) *Location {
	if info := n.FindPropertyDeclaration(lines, ifaceName, memberName, uri); info != nil {
		return &Location{
			URI: info.URI,
			Range: Range{
				Start: Position{Line: info.Line, Character: info.CharStart},
				End:   Position{Line: info.Line, Character: info.CharEnd},
			},
		}
	}
	return nil
}

// ExtractDocCommentForLine extracts leading comments directly above a declaration line.
func ExtractDocCommentForLine(lines []string, declLine0Idx int) string {
	idx := declLine0Idx - 1
	if idx < 0 || idx >= len(lines) {
		return ""
	}

	var commentLines []string
	inBlockComment := false

	for idx >= 0 {
		rawLine := lines[idx]
		line := strings.TrimSpace(rawLine)

		if inBlockComment {
			if strings.HasPrefix(line, "/*") {
				cleaned := strings.TrimPrefix(line, "/*")
				cleaned = strings.TrimPrefix(cleaned, "*")
				cleaned = strings.TrimSpace(cleaned)
				if cleaned != "" {
					commentLines = append([]string{cleaned}, commentLines...)
				}
				break
			}
			cleaned := strings.TrimPrefix(line, "*")
			cleaned = strings.TrimSpace(cleaned)
			commentLines = append([]string{cleaned}, commentLines...)
			idx--
			continue
		}

		if strings.HasSuffix(line, "*/") {
			inBlockComment = true
			cleaned := strings.TrimSuffix(line, "*/")
			cleaned = strings.TrimPrefix(cleaned, "/*")
			cleaned = strings.TrimPrefix(cleaned, "*")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				commentLines = append([]string{cleaned}, commentLines...)
			}
			idx--
			continue
		}

		if strings.HasPrefix(line, "//") {
			cleaned := strings.TrimPrefix(line, "//")
			cleaned = strings.TrimPrefix(cleaned, "/") // for /// comments
			cleaned = strings.TrimPrefix(cleaned, " ")
			commentLines = append([]string{cleaned}, commentLines...)
			idx--
			continue
		}

		break
	}

	return strings.Join(commentLines, "\n")
}
