package navigation

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FindDeclarationInFile scans a source file for declarations matching the specified word.
func FindDeclarationInFile(filePath, word string) (line, char int, found bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, false
	}
	lines := strings.Split(string(content), "\n")
	reDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare|interface|type)\s+(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reDecl.FindStringSubmatchIndex(l); loc != nil && len(loc) >= 4 {
			return lIdx, loc[2], true
		}
	}
	reMod := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*mod\s+(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reMod.FindStringSubmatchIndex(l); loc != nil && len(loc) >= 4 {
			return lIdx, loc[2], true
		}
	}
	reWord := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reWord.FindStringIndex(l); loc != nil {
			return lIdx, loc[0], true
		}
	}
	return 0, 0, false
}
