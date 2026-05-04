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
	// Detect unclosed braces first
	inBrace := false
	for i, r := range s {
		if r == '{' && (i == 0 || s[i-1] != '\\') {
			inBrace = true
		} else if r == '}' && (i == 0 || s[i-1] != '\\') {
			inBrace = false
		}
		if i == len(s)-1 && inBrace {
			return "[SYNTAX_ERROR: UNCLOSED_BRACE]"
		}
	}

	// Handle escaping: \{ becomes {
	s = strings.ReplaceAll(s, "\\{", "{")
	s = strings.ReplaceAll(s, "\\}", "}")

	if provider == nil {
		return s
	}

	resultLines := strings.Split(s, "\n")
	// Recursive interpolation (max 10 levels) to support {A.{B}}
	for pass := 0; pass < 10; pass++ {
		interpolatedSomething := false
		var nextResultLines []string

		for _, line := range resultLines {
			// Detect if this line only contains a single {VAR} (after indentation)
			trimmed := strings.TrimSpace(line)
			matches := varRegex.FindAllString(trimmed, -1)
			isOnlyVar := len(matches) == 1 && matches[0] == trimmed

			if isOnlyVar {
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

				val := ""
				if rawVal == nil {
					val = ""
				} else {
					switch v := rawVal.(type) {
					case string:
						val = v
					case ast.Literal:
						val = string(v)
					default:
						val = stringify(rawVal)
					}
				}

				if val == "" {
					continue
				}

				interpolatedSomething = true
				val = Dedent(val)

				valLines := strings.Split(val, "\n")
				for j := range valLines {
					if valLines[j] != "" {
						valLines[j] = indent + valLines[j]
					}
				}
				nextResultLines = append(nextResultLines, valLines...)
			} else {
				if varRegex.MatchString(line) {
					interpolatedSomething = true
				}
				interpolated := varRegex.ReplaceAllStringFunc(line, func(m string) string {
					name := m[1 : len(m)-1]
					if rawVal, ok := provider.Get(name); ok {
						switch v := rawVal.(type) {
						case string:
							return v
						case ast.Literal:
							return string(v)
						default:
							return stringify(rawVal)
						}
					}
					return "[ERROR: UNDEFINED_" + name + "]"
				})
				nextResultLines = append(nextResultLines, interpolated)
			}
		}

		resultLines = nextResultLines
		if !interpolatedSomething {
			break
		}
	}

	return strings.Join(resultLines, "\n")
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
		return Interpolate(val, provider)
	case ast.Literal:
		return ast.Literal(Interpolate(string(val), provider))
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
