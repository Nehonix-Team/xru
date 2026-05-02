/***************************************************************************
 * XFPM — XRU package internal utilities
 ***************************************************************************** */

package engine

import (
	"os"
	"strings"
)

// readFile is the single I/O primitive used by the package.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Interpolate replaces {VAR} placeholders with values from vars map.
func Interpolate(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// InterpolateValue recursively interpolates strings inside structured values.
func InterpolateValue(v Value, vars map[string]string) Value {
	switch val := v.(type) {
	case Literal:
		return Literal(Interpolate(string(val), vars))
	case Object:
		newObj := make(Object)
		for k, v := range val {
			newObj[k] = InterpolateValue(v, vars)
		}
		return newObj
	case Array:
		newArr := make(Array, len(val))
		for i, v := range val {
			newArr[i] = InterpolateValue(v, vars)
		}
		return newArr
	}
	return v
}
