package patcher

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func deepen(path string, val ast.Value) ast.Object {
	parts := strings.Split(path, ".")
	res := make(ast.Object)
	curr := res
	for i := 0; i < len(parts)-1; i++ {
		next := make(ast.Object)
		curr[parts[i]] = next
		curr = next
	}
	curr[parts[len(parts)-1]] = val
	return res
}

func serialiseValue(v ast.Value, indent string) string {
	switch val := v.(type) {
	case ast.Object:
		if len(val) == 0 { return "{}" }
		res := "{\n"
		for k, v := range val {
			res += fmt.Sprintf("%s  %q: %s,\n", indent, k, serialiseValue(v, indent+"  "))
		}
		res += indent + "}"
		return res
	case ast.Array:
		if len(val) == 0 { return "[]" }
		res := "[\n"
		for _, v := range val {
			res += fmt.Sprintf("%s  %s,\n", indent, serialiseValue(v, indent+"  "))
		}
		res += indent + "]"
		return res
	case ast.Literal:
		s := string(val)
		if s == "true" || s == "false" || s == "null" {
			return s
		}
		var n float64
		if _, err := fmt.Sscanf(s, "%f", &n); err == nil && !strings.Contains(s, " ") {
			return s
		}
		b, _ := json.Marshal(s)
		return string(b)
	}
	return ""
}

func matchingBracket(s string, openIdx int) int {
	depth := 0
	inString := false
	var quote rune

	runes := []rune(s)
	for i := openIdx; i < len(runes); i++ {
		r := runes[i]
		if inString {
			if r == quote && runes[i-1] != '\\' {
				inString = false
			}
			continue
		}

		if r == '"' || r == '\'' {
			inString = true
			quote = r
			continue
		}

		if r == '[' {
			depth++
		} else if r == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func matchingBrace(s string, openIdx int) int {
	depth := 0
	inString := false
	var quote rune

	runes := []rune(s)
	for i := openIdx; i < len(runes); i++ {
		r := runes[i]
		if inString {
			if r == quote && runes[i-1] != '\\' {
				inString = false
			}
			continue
		}

		if r == '"' || r == '\'' {
			inString = true
			quote = r
			continue
		}

		if r == '{' {
			depth++
		} else if r == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
