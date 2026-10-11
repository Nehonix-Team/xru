package ast

import "strings"

// InterpSpan represents a single interpolation block {...} inside a string literal.
type InterpSpan struct {
	Raw      string // Raw text including braces, e.g. "{Crypto.GetRandomBytes(12)}"
	Expr     string // Extracted expression, e.g. "Crypto.GetRandomBytes(12)"
	StartIdx int    // 0-based index of '{' in the string literal content
	EndIdx   int    // 0-based index of '}' in the string literal content
}

// ExtractInterpolations scans a string content and extracts all valid `{...}` interpolation blocks.
// It ignores escaped braces (\{, \}) and double braces ({{...}} used in file templates).
func ExtractInterpolations(s string) []InterpSpan {
	var spans []InterpSpan
	n := len(s)
	i := 0
	for i < n {
		if s[i] == '\\' {
			i += 2
			continue
		}
		if s[i] == '{' && i+1 < n && s[i+1] == '{' {
			idx := strings.Index(s[i+2:], "}}")
			if idx != -1 {
				i = i + 2 + idx + 2
			} else {
				i += 2
			}
			continue
		}
		if s[i] == '{' {
			start := i
			braceCount := 1
			inQuote := byte(0)
			j := i + 1
			for j < n && braceCount > 0 {
				ch := s[j]
				if inQuote != 0 {
					if ch == '\\' {
						j += 2
						continue
					}
					if ch == inQuote {
						inQuote = 0
					}
				} else {
					if ch == '"' || ch == '\'' || ch == '`' {
						inQuote = ch
					} else if ch == '{' {
						braceCount++
					} else if ch == '}' {
						braceCount--
					}
				}
				j++
			}
			if braceCount == 0 {
				raw := s[start:j]
				expr := strings.TrimSpace(s[start+1 : j-1])
				spans = append(spans, InterpSpan{
					Raw:      raw,
					Expr:     expr,
					StartIdx: start,
					EndIdx:   j - 1,
				})
				i = j
				continue
			}
		}
		i++
	}
	return spans
}
