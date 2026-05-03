/***************************************************************************
 * XFPM — XRU package internal utilities
 ***************************************************************************** */

package engine

import (
	"os"
	"regexp"
)

var varRegex = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)

// VarProvider is an interface for looking up variables and tracking usage.
type VarProvider interface {
	Get(name string) (string, bool)
}

// readFile is the single I/O primitive used by the package.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

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
