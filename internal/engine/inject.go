/***************************************************************************
 * XRU — Code Injection Engine
 *
 * Replaces `// xfpm: {{KEY}}` markers with the provided code block.
 * The marker detection is whitespace-tolerant and handles both:
 *   - `// xfpm: {{KEY}}`  (with braces)
 *   - `// xfpm: KEY`      (without braces)
 ***************************************************************************** */

package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// InjectCode replaces lines containing specific markers or matching a regex with code.
func InjectCode(content, key, code string, raw bool) string {
	if key == "" {
		return code
	}
	var re *regexp.Regexp
	var pattern string

	// Si la clé est un Regex (ex: /pattern/)
	if strings.HasPrefix(key, "/") && strings.HasSuffix(key, "/") && len(key) > 2 {
		pattern = "(?m)" + key[1:len(key)-1]
		re = regexp.MustCompile(pattern)
		return re.ReplaceAllString(content, code)
	}

	// Sinon, recherche par marqueur universel
	escaped := regexp.QuoteMeta(key)
	if strings.HasPrefix(key, "{{") && strings.HasSuffix(key, "}}") {
		escaped = regexp.QuoteMeta(key[2 : len(key)-2])
	}
	// Pattern universel avec capture de l'indentation
	pattern = fmt.Sprintf(`(?m)^([ \t]*).*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:|@[A-Z]+INJECT:)?\s*(?:\{\{)?%s(?:\}\})?.*$`, escaped)
	re = regexp.MustCompile(pattern)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		if raw {
			return code
		}
		// Extract indentation from the match
		submatches := re.FindStringSubmatch(match)
		indent := ""
		if len(submatches) > 1 {
			indent = submatches[1]
		}

		// Apply indentation to each line of the code
		lines := strings.Split(code, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) != "" {
				lines[i] = indent + line
			}
		}
		return strings.Join(lines, "\n")
	})
}

// CleanOrphans removes any remaining injection markers from the content.
func CleanOrphans(content string) string {
	// Matches any line containing an xfpm/xru marker or a {{VAR}} placeholder inside a comment.
	re := regexp.MustCompile(`(?m)^.*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:|@[A-Z]+INJECT:)\s*(?:\{\{)?[a-zA-Z0-9_\.]+(?:\}\})?.*$\n?`)
	content = re.ReplaceAllString(content, "")
	// Also clean standalone {{VAR}} markers that might be left.
	re2 := regexp.MustCompile(`(?m)^.*\{\{[a-zA-Z0-9_\.]+\}\}.*$\n?`)
	return re2.ReplaceAllString(content, "")
}
