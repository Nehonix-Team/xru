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

// InjectCode replaces lines containing specific markers with the provided code block.
// It supports multiple comment styles (//, #, --, /*, <!--) and triggers (--> , xru:, xfpm:).
func InjectCode(content, key, code string) string {
	escaped := regexp.QuoteMeta(key)
	if strings.HasPrefix(key, "{{") && strings.HasSuffix(key, "}}") {
		escaped = regexp.QuoteMeta(key[2 : len(key)-2])
	}

	// Pattern universel :
	// 1. Détecte le début de ligne
	// 2. Détecte un préfixe de commentaire (//, #, --, /*, <!--)
	// 3. Détecte un déclencheur optionnel (--> , xru:, xfpm:)
	// 4. Détecte la clé (avec ou sans accolades {{}})
	pattern := fmt.Sprintf(`(?m)^.*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:)?\s*(?:\{\{)?%s(?:\}\})?.*$`, escaped)
	re := regexp.MustCompile(pattern)

	return re.ReplaceAllString(content, code)
}

// CleanOrphans removes any remaining injection markers from the content.
func CleanOrphans(content string) string {
	// Matches any universal marker line and its trailing newline.
	re := regexp.MustCompile(`(?m)^.*(?://|#|--|/\*|<!--)\s*(?:-->|xru:|xfpm:)?\s*(?:\{\{)?[A-Z0-9_]+(?:\}\})?.*$\n?`)
	return re.ReplaceAllString(content, "")
}
