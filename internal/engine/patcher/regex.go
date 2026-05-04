package patcher

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func applyRegex(content string, patch ast.Object) string {
	for k, v := range patch {
		if val, ok := v.(ast.Literal); ok {
			re, err := regexp.Compile(k)
			if err != nil {
				continue
			}
			content = re.ReplaceAllString(content, string(val))
		}
	}
	return content
}

func renameKeyStructured(content string, patch ast.Object) string {
	for oldK, v := range patch {
		if newK, ok := v.(ast.Literal); ok {
			keyPatternQuoted := fmt.Sprintf("%q:", oldK)
			keyPatternBare := oldK + ":"

			start := strings.Index(content, keyPatternQuoted)
			if start == -1 {
				start = strings.Index(content, keyPatternBare)
			}
			if start != -1 {
				if strings.HasPrefix(content[start:], keyPatternQuoted) {
					content = content[:start] + fmt.Sprintf("%q:", string(newK)) + content[start+len(keyPatternQuoted):]
				} else {
					content = content[:start] + string(newK) + ":" + content[start+len(keyPatternBare):]
				}
			}
		}
	}
	return content
}
