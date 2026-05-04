/***************************************************************************
 * XFPM — XRU package internal utilities
 ***************************************************************************** */

package util

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

var varRegex = regexp.MustCompile(`\{[a-zA-Z0-9_\.]+\}`)

// VarProvider is an interface for looking up variables and tracking usage.
type VarProvider interface {
	Get(name string) (interface{}, bool)
}

// Interpolate replaces {VAR} placeholders with values from a VarProvider.
func Interpolate(s string, provider VarProvider) string {
	if provider == nil {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\\{", "{"), "\\}", "}")
	}

	resultLines := strings.Split(s, "\n")
	for pass := 0; pass < 10; pass++ {
		interpolatedSomething := false
		var nextResultLines []string

		for _, line := range resultLines {
			trimmed := strings.TrimSpace(line)
			matches := varRegex.FindAllString(trimmed, -1)
			isOnlyVar := len(matches) == 1 && matches[0] == trimmed

			if isOnlyVar {
				// Get indentation
				indent := ""
				for _, r := range line {
					if r == ' ' || r == '\t' {
						indent += string(r)
					} else {
						break
					}
				}

				name := trimmed[1 : len(trimmed)-1]
				rawVal, ok := provider.Get(name)
				if !ok {
					rawVal = "[ERROR: UNDEFINED_" + name + "]"
				}

				if rawVal == nil {
					nextResultLines = append(nextResultLines, "")
					interpolatedSomething = true
					continue
				}

				val := fmt.Sprint(rawVal)
				lines := strings.Split(val, "\n")
				for _, l := range lines {
					nextResultLines = append(nextResultLines, indent+l)
				}
				interpolatedSomething = true
			} else {
				// Inline interpolation
				safety := 0
				for {
					safety++
					if safety > 100 {
						break
					}
					match := varRegex.FindStringIndex(line)
					if match == nil {
						break
					}
					start, end := match[0], match[1]

					name := line[start+1 : end-1]
					rawVal, ok := provider.Get(name)
					if !ok {
						rawVal = "[ERROR: UNDEFINED_" + name + "]"
					}
					
					replacement := fmt.Sprint(rawVal)
					line = line[:start] + replacement + line[end:]
					interpolatedSomething = true
				}
				nextResultLines = append(nextResultLines, line)
			}
		}

		resultLines = nextResultLines
		if !interpolatedSomething {
			break
		}
	}

	final := strings.Join(resultLines, "\n")
	final = strings.ReplaceAll(final, "\\{", "{")
	final = strings.ReplaceAll(final, "\\}", "}")
	return final
}

func stringify(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case ast.Literal:
		return string(val)
	case ast.Object, ast.Array, map[string]interface{}, []interface{}:
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// InterpolateValue recursively interpolates strings inside structured values.
func InterpolateValue(v interface{}, provider VarProvider) interface{} {
	if provider == nil {
		return v
	}
	switch val := v.(type) {
	case string:
		trimmed := strings.TrimSpace(val)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			name := trimmed[1 : len(trimmed)-1]
			if !strings.ContainsAny(name, "{} ") {
				if obj, ok := provider.Get(name); ok {
					return obj
				}
			}
		}
		return Interpolate(val, provider)
	case ast.Literal:
		valStr := string(val)
		trimmed := strings.TrimSpace(valStr)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			name := trimmed[1 : len(trimmed)-1]
			if !strings.ContainsAny(name, "{} ") {
				if obj, ok := provider.Get(name); ok {
					return obj
				}
			}
		}
		return ast.Literal(Interpolate(valStr, provider))
	case ast.Object:
		newObj := make(ast.Object)
		for k, v := range val {
			newObj[k] = InterpolateValue(v, provider).(ast.Value)
		}
		return newObj
	case ast.Array:
		newArr := make(ast.Array, len(val))
		for i, v := range val {
			newArr[i] = InterpolateValue(v, provider).(ast.Value)
		}
		return newArr
	}
	return v
}

// Dedent removes the common leading whitespace from all lines in s.
func Dedent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return s
	}

	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := 0
		for _, r := range line {
			if r == ' ' || r == '\t' {
				indent++
			} else {
				break
			}
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent <= 0 {
		return strings.TrimSuffix(s, "\n")
	}

	var result []string
	for _, line := range lines {
		if len(line) >= minIndent {
			isIndented := true
			for i := 0; i < minIndent; i++ {
				if line[i] != ' ' && line[i] != '\t' {
					isIndented = false
					break
				}
			}
			if isIndented {
				result = append(result, line[minIndent:])
			} else {
				result = append(result, strings.TrimSpace(line))
			}
		} else {
			result = append(result, strings.TrimSpace(line))
		}
	}

	return strings.Join(result, "\n")
}
