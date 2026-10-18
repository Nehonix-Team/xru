package navigation

import (
	"fmt"
	"regexp"
	"strings"
)

// resolveLocalDefinition searches for local declarations in the current document (let, const, arg, func, unit, declare, interface, type, lambdas, func params, for loops, mod).
func resolveLocalDefinition(lines []string, word, uri string) *Location {
	reLocalDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare(?:\s+(?:public|private))?|interface|type)\s+(%s)\b`, regexp.QuoteMeta(word)))
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if loc := reLocalDecl.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}
	}

	reLambda := regexp.MustCompile(`\(([^)]*)\)\s*(?:->|=>)`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reLambda.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			paramSub := line[locs[2]:locs[3]]
			paramRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if pLoc := paramRe.FindStringIndex(paramSub); pLoc != nil {
				startCol := locs[2] + pLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reFuncParams := regexp.MustCompile(`func\s+[a-zA-Z0-9_]+\s*\(([^)]*)\)`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reFuncParams.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			paramSub := line[locs[2]:locs[3]]
			paramRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if pLoc := paramRe.FindStringIndex(paramSub); pLoc != nil {
				startCol := locs[2] + pLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reForLoop := regexp.MustCompile(`^\s*for\s+([a-zA-Z0-9_,\s]+)\s+in\b`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reForLoop.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			varSub := line[locs[2]:locs[3]]
			varRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if vLoc := varRe.FindStringIndex(varSub); vLoc != nil {
				startCol := locs[2] + vLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reMod := regexp.MustCompile(fmt.Sprintf(`^\s*mod\s+(%s)\b`, regexp.QuoteMeta(word)))
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if loc := reMod.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}
	}

	return nil
}
