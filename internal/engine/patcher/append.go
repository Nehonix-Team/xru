package patcher

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func appendStructured(content string, patch ast.Object) string {
	for k, v := range patch {
		keyPatternQuoted := fmt.Sprintf("%q:", k)
		keyPatternBare := k + ":"
		
		start := strings.Index(content, keyPatternQuoted)
		if start == -1 {
			start = strings.Index(content, keyPatternBare)
		}

		if start != -1 {
			openOff := strings.Index(content[start:], "[")
			if openOff != -1 {
				absOpen := start + openOff
				closeAbs := matchingBracket(content, absOpen)
				if closeAbs != -1 {
					indent := "  "
					lastNewline := strings.LastIndex(content[:closeAbs], "\n")
					if lastNewline != -1 {
						line := content[lastNewline+1 : closeAbs]
						spaces := 0
						for _, r := range line {
							if r == ' ' { spaces++ } else { break }
						}
						indent = strings.Repeat(" ", spaces+2)
					}
					
					serialised := serialiseValue(v, indent)
					prefix := strings.TrimRight(content[:closeAbs], " \t\n\r")
					separator := ","
					if strings.TrimSpace(content[absOpen+1:closeAbs]) == "" {
						separator = ""
					}
					
					entry := fmt.Sprintf("%s\n%s%s,", separator, indent, serialised)
					content = prefix + entry + "\n" + strings.Repeat(" ", strings.Count(indent, " ")-2) + content[closeAbs:]
				}
			}
		}
	}
	return content
}
