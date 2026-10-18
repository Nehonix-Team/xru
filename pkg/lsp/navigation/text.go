package navigation

import (
	"strings"
)

// GetWordAt returns the word under the given line and character position.
func GetWordAt(text string, line, char int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	l := lines[line]
	if char < 0 || char > len(l) {
		return ""
	}

	start := char
	for start > 0 && IsWordChar(l[start-1]) {
		start--
	}
	end := char
	for end < len(l) && IsWordChar(l[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return l[start:end]
}

// IsWordChar returns true if b is an identifier character.
func IsWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// GetFullExpressionAt retrieves the member or qualified expression at line and char (e.g. prefix.word or prefix::word).
func GetFullExpressionAt(text string, line, char int) (string, string) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return "", ""
	}
	l := lines[line]
	targetWord := GetWordAt(text, line, char)
	if targetWord == "" {
		return "", ""
	}

	// Look backwards from word start to see if there is `prefix.` or `prefix::`
	wordStart := char
	for wordStart > 0 && IsWordChar(l[wordStart-1]) {
		wordStart--
	}
	if wordStart >= 2 && l[wordStart-2:wordStart] == "::" {
		prefixEnd := wordStart - 2
		prefixStart := prefixEnd
		for prefixStart > 0 && IsWordChar(l[prefixStart-1]) {
			prefixStart--
		}
		if prefixEnd > prefixStart {
			prefix := l[prefixStart:prefixEnd]
			return prefix + "::" + targetWord, targetWord
		}
	} else if wordStart > 0 && l[wordStart-1] == '.' {
		prefixEnd := wordStart - 1
		prefixStart := prefixEnd
		for prefixStart > 0 && IsWordChar(l[prefixStart-1]) {
			prefixStart--
		}
		if prefixEnd > prefixStart {
			prefix := l[prefixStart:prefixEnd]
			return prefix + "." + targetWord, targetWord
		} else if prefixEnd > 0 && (l[prefixEnd-1] == '"' || l[prefixEnd-1] == '\'') {
			return "string." + targetWord, targetWord
		} else if prefixEnd > 0 && l[prefixEnd-1] == ']' {
			return "list." + targetWord, targetWord
		}
		return "." + targetWord, targetWord
	}

	// Look forwards from word end to see if there is `.suffix` or `::suffix`
	wordEnd := char
	for wordEnd < len(l) && IsWordChar(l[wordEnd]) {
		wordEnd++
	}
	if wordEnd+2 <= len(l) && l[wordEnd:wordEnd+2] == "::" {
		suffixStart := wordEnd + 2
		suffixEnd := suffixStart
		for suffixEnd < len(l) && IsWordChar(l[suffixEnd]) {
			suffixEnd++
		}
		if suffixEnd > suffixStart {
			suffix := l[suffixStart:suffixEnd]
			return targetWord + "::" + suffix, targetWord
		}
	} else if wordEnd < len(l) && l[wordEnd] == '.' {
		suffixStart := wordEnd + 1
		suffixEnd := suffixStart
		for suffixEnd < len(l) && IsWordChar(l[suffixEnd]) {
			suffixEnd++
		}
		if suffixEnd > suffixStart {
			suffix := l[suffixStart:suffixEnd]
			return targetWord + "." + suffix, targetWord
		}
	}

	return targetWord, targetWord
}
