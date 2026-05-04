package patcher

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func removeStructured(content string, patch ast.Value) string {
	switch p := patch.(type) {
	case ast.Object:
		for k, v := range p {
			keyPatternQuoted := fmt.Sprintf("%q:", k)
			keyPatternBare := k + ":"
			
			start := strings.Index(content, keyPatternQuoted)
			if start == -1 {
				start = strings.Index(content, keyPatternBare)
			}

			if start != -1 {
				if nestedPatch, ok := v.(ast.Object); ok {
					openOff := strings.Index(content[start:], "{")
					if openOff != -1 {
						absOpen := start + openOff
						closeAbs := matchingBrace(content, absOpen)
						if closeAbs != -1 {
							inner := content[absOpen : closeAbs+1]
							newInner := removeStructured(inner, nestedPatch)
							content = content[:absOpen] + newInner + content[closeAbs+1:]
						}
					}
				} else {
					lineStart := strings.LastIndex(content[:start], "\n")
					if lineStart == -1 { lineStart = 0 } else { lineStart++ }
					
					sepPos := strings.Index(content[start:], ":")
					if sepPos != -1 {
						afterSep := content[start+sepPos+1:]
						p := 0
						for p < len(afterSep) && (afterSep[p] == ' ' || afterSep[p] == '\t' || afterSep[p] == '\r') {
							p++
						}
						if p < len(afterSep) && afterSep[p] == '{' {
							absOpen := start + sepPos + 1 + p
							closeAbs := matchingBrace(content, absOpen)
							if closeAbs != -1 {
								lineEnd := strings.Index(content[closeAbs:], "\n")
								if lineEnd == -1 { lineEnd = len(content[closeAbs:]) }
								absEnd := closeAbs + lineEnd
								content = content[:lineStart] + content[absEnd+1:]
								continue
							}
						}
					}

					lineEnd := strings.Index(content[start:], "\n")
					if lineEnd == -1 { lineEnd = len(content) - start }
					absEnd := start + lineEnd
					content = content[:lineStart] + content[absEnd+1:]
				}
			}
		}
	case ast.Array:
		for _, v := range p {
			if s, ok := v.(ast.Literal); ok {
				content = removeStructured(content, ast.Object{string(s): ast.Literal("")})
			}
		}
	}
	return content
}
