package ast

import "strings"

// InterpSpan represents a single interpolation block $f{...} inside a string literal.
type InterpSpan struct {
	Raw      string // Raw text including sigil and braces, e.g. "$f{Crypto.GetRandomBytes(12)}"
	Expr     string // Extracted expression, e.g. "Crypto.GetRandomBytes(12)"
	StartIdx int    // 0-based index of '$' in the string literal content
	EndIdx   int    // 0-based index of '}' in the string literal content
}

// ExtractInterpolations scans a string content and extracts all valid `$f{...}` interpolation blocks.
// Plain `{...}`, `${...}`, or escaped `\$f{...}` are treated as literal text.
func ExtractInterpolations(s string) []InterpSpan {
	var spans []InterpSpan
	n := len(s)
	i := 0
	for i < n {
		if s[i] == '\\' {
			i += 2
			continue
		}
		// Détection exclusive du token $f{
		if i+2 < n && s[i] == '$' && s[i+1] == 'f' && s[i+2] == '{' {
			start := i
			braceCount := 1
			inQuote := byte(0)
			j := i + 3
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
				expr := strings.TrimSpace(s[start+3 : j-1])
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
