/***************************************************************************
 * XFPM — XRU package internal utilities
 ***************************************************************************** */

package engine

import (
	// "os"
	"regexp"
	"strings"
)

var varRegex = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)

// VarProvider is an interface for looking up variables and tracking usage.
type VarProvider interface {
	Get(name string) (string, bool)
}

// // readFile is the single I/O primitive used by the package.
// func readFile(path string) ([]byte, error) {
// 	return os.ReadFile(path)
// }

// Interpolate replaces {VAR} placeholders with values from a VarProvider.
func Interpolate(s string, provider VarProvider) string {
	// Detect unclosed braces first
	inBrace := false
	for i, r := range s {
		if r == '{' {
			inBrace = true
		} else if r == '}' {
			inBrace = false
		}
		if i == len(s)-1 && inBrace {
			return "[SYNTAX_ERROR: UNCLOSED_BRACE]"
		}
	}

	// Handle escaping: \{ becomes {
	s = strings.ReplaceAll(s, "\\{", "{")

	if provider == nil {
		return s
	}
	return varRegex.ReplaceAllStringFunc(s, func(m string) string {
		name := m[1 : len(m)-1]
		if val, ok := provider.Get(name); ok {
			return val
		}
		return "[ERROR: UNDEFINED_VAR]"
	})
}

// InterpolateValue recursively interpolates strings inside structured values.
func InterpolateValue(v Value, provider VarProvider) Value {
	if provider == nil {
		return v
	}
	switch val := v.(type) {
	case Literal:
		return Literal(Interpolate(string(val), provider))
	case Object:
		newObj := make(Object)
		for k, v := range val {
			newObj[k] = InterpolateValue(v, provider)
		}
		return newObj
	case Array:
		newArr := make(Array, len(val))
		for i, v := range val {
			newArr[i] = InterpolateValue(v, provider)
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
			// Check if the prefix is indeed indentation
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
