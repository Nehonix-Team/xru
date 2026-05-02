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

// InjectCode replaces every line containing `// xfpm: <key>` with code.
// The match is whitespace-agnostic and works whether key has `{{}}` braces or not.
func InjectCode(content, key, code string) string {
	escaped := regexp.QuoteMeta(key)
	pattern := fmt.Sprintf(`(?m)^.*//\s*xfpm:\s*%s.*$`, escaped)
	re := regexp.MustCompile(pattern)

	// If the key has braces and we didn't match, try the bare inner name
	if !re.MatchString(content) && strings.HasPrefix(key, "{{") && strings.HasSuffix(key, "}}") {
		inner := key[2 : len(key)-2]
		pattern = fmt.Sprintf(`(?m)^.*//\s*xfpm:\s*%s.*$`, regexp.QuoteMeta(inner))
		re = regexp.MustCompile(pattern)
	}

	return re.ReplaceAllString(content, code)
}

// CleanOrphans removes any remaining `// xfpm:` markers from the content.
// This is used as a final cleanup pass to ensure unselected features don't leave artifacts.
func CleanOrphans(content string) string {
	// Matches any line that contains the xfpm marker, with or without braces.
	// It handles both:
	//   - // xfpm: {{KEY}}
	//   - // xfpm: KEY
	// It also cleans the whole line and the trailing newline.
	re := regexp.MustCompile(`(?m)^.*//\s*xfpm:\s*(\{\{)?([A-Z0-9_]+)(\}\})?.*$\n?`)
	return re.ReplaceAllString(content, "")
}
