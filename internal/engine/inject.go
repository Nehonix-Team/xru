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
func InjectCode(content, key, code string) string {
	if key == "" {
		return code
	}
	var re *regexp.Regexp
	var pattern string

	// Si la clé est un Regex (ex: /pattern/)
	if strings.HasPrefix(key, "/") && strings.HasSuffix(key, "/") && len(key) > 2 {
		pattern = "(?m)" + key[1:len(key)-1]
		re = regexp.MustCompile(pattern)
	} else {
		// Sinon, recherche par marqueur universel
		escaped := regexp.QuoteMeta(key)
		if strings.HasPrefix(key, "{{") && strings.HasSuffix(key, "}}") {
			escaped = regexp.QuoteMeta(key[2 : len(key)-2])
		}
		// Pattern universel avec support des commentaires et déclencheurs
		pattern = fmt.Sprintf(`(?m)^.*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:|@[A-Z]+INJECT:)?\s*(?:\{\{)?%s(?:\}\})?.*$`, escaped)
		re = regexp.MustCompile(pattern)
	}

	return re.ReplaceAllString(content, code)
}

// CleanOrphans removes any remaining injection markers from the content.
func CleanOrphans(content string) string {
	// Matches any universal marker line and its trailing newline.
	re := regexp.MustCompile(`(?m)^.*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:)?\s*(?:\{\{)?[A-Z0-9_]+(?:\}\})?.*$\n?`)
	return re.ReplaceAllString(content, "")
}
