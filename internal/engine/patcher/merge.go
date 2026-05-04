package patcher

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func mergeStructured(content string, patch ast.Object) string {
	for k, v := range patch {
		keyPatternQuoted := fmt.Sprintf("%q:", k)
		keyPatternBare := k + ":"
		
		start := strings.Index(content, keyPatternQuoted)
		if start == -1 {
			start = strings.Index(content, keyPatternBare)
		}

		if start != -1 {
			switch val := v.(type) {
			case ast.Object:
				openOff := strings.Index(content[start:], "{")
				if openOff != -1 {
					absOpen := start + openOff
					closeAbs := matchingBrace(content, absOpen)
					if closeAbs != -1 {
						inner := content[absOpen : closeAbs+1]
						newInner := mergeStructured(inner, val)
						content = content[:absOpen] + newInner + content[closeAbs+1:]
					}
				}
			default:
				lineEnd := strings.Index(content[start:], "\n")
				if lineEnd == -1 { lineEnd = len(content) - start }
				
				absEnd := start + lineEnd
				sepPos := strings.Index(content[start:absEnd], ":")
				if sepPos != -1 {
					prefix := content[:start+sepPos+1]
					suffix := content[absEnd:]
					comma := ""
					if strings.Contains(content[start+sepPos:absEnd], ",") {
						comma = ","
					}
					content = prefix + " " + serialiseValue(v, "") + comma + suffix
				}
			}
		} else {
			content = injectAtEnd(content, k, v)
		}
	}
	return content
}

func injectAtEnd(content, key string, value ast.Value) string {
	pos := strings.LastIndex(content, "}")
	if pos == -1 {
		return content
	}

	indent := "  "
	lastNewline := strings.LastIndex(content[:pos], "\n")
	if lastNewline != -1 {
		line := content[lastNewline+1 : pos]
		spaces := 0
		for _, r := range line {
			if r == ' ' { spaces++ } else { break }
		}
		if spaces > 0 {
			indent = strings.Repeat(" ", spaces)
		}
	}

	serialised := serialiseValue(value, indent)
	prefix := strings.TrimRight(content[:pos], " \t\n\r")
	
	if !strings.HasSuffix(prefix, "{") && !strings.HasSuffix(prefix, ",") && !strings.HasSuffix(prefix, "[") {
		prefix += ","
	}
	
	entry := fmt.Sprintf("\n%s%q: %s,", indent, key, serialised)

	return prefix + entry + "\n" + content[pos:]
}
