package runtime

import (
	"regexp"
	"strings"
)

var reTerminalTag = regexp.MustCompile(`</?[a-zA-Z0-9_\-]*>`)

var terminalColorCodes = map[string]string{
	// Standard foreground colors
	"red":            "\033[31m",
	"green":          "\033[32m",
	"yellow":         "\033[33m",
	"blue":           "\033[34m",
	"magenta":        "\033[35m",
	"purple":         "\033[35m",
	"cyan":           "\033[36m",
	"white":          "\033[37m",
	"gray":           "\033[90m",
	"grey":           "\033[90m",
	"black":          "\033[30m",

	// Bright foreground colors
	"bright_red":     "\033[91m",
	"light_red":      "\033[91m",
	"bright_green":   "\033[92m",
	"light_green":    "\033[92m",
	"bright_yellow":  "\033[93m",
	"light_yellow":   "\033[93m",
	"bright_blue":    "\033[94m",
	"light_blue":     "\033[94m",
	"bright_magenta": "\033[95m",
	"light_magenta":  "\033[95m",
	"bright_cyan":    "\033[96m",
	"light_cyan":     "\033[96m",
	"bright_white":   "\033[97m",

	// Styles
	"bold":           "\033[1m",
	"b":              "\033[1m",
	"dim":            "\033[2m",
	"italic":         "\033[3m",
	"i":              "\033[3m",
	"underline":      "\033[4m",
	"u":              "\033[4m",

	// Backgrounds
	"bg_red":         "\033[41m",
	"bg_green":       "\033[42m",
	"bg_yellow":      "\033[43m",
	"bg_blue":        "\033[44m",
	"bg_magenta":     "\033[45m",
	"bg_cyan":        "\033[46m",
	"bg_white":       "\033[47m",
	"bg_black":       "\033[40m",
}

func formatTerminalColors(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}

	matches := reTerminalTag.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return s
	}

	var sb strings.Builder
	var stack []string // holds active tags
	lastIdx := 0

	for _, loc := range matches {
		start, end := loc[0], loc[1]
		sb.WriteString(s[lastIdx:start])
		lastIdx = end

		tagText := s[start:end]
		inner := strings.ToLower(tagText[1 : len(tagText)-1])

		if inner == "/" {
			// Generic close tag: </>
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			sb.WriteString("\033[0m")
			for _, t := range stack {
				if code, ok := terminalColorCodes[t]; ok {
					sb.WriteString(code)
				}
			}
		} else if strings.HasPrefix(inner, "/") {
			// Named close tag: </blue>, </red>, </bold>, etc.
			tagName := strings.TrimPrefix(inner, "/")
			tagName = strings.ReplaceAll(tagName, "-", "_")

			foundIdx := -1
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i] == tagName {
					foundIdx = i
					break
				}
			}
			if foundIdx != -1 {
				stack = append(stack[:foundIdx], stack[foundIdx+1:]...)
			}

			sb.WriteString("\033[0m")
			for _, t := range stack {
				if code, ok := terminalColorCodes[t]; ok {
					sb.WriteString(code)
				}
			}
		} else {
			// Opening tag: <blue>, <red>, etc.
			tagName := strings.ReplaceAll(inner, "-", "_")
			if code, ok := terminalColorCodes[tagName]; ok {
				stack = append(stack, tagName)
				sb.WriteString(code)
			} else {
				sb.WriteString(tagText)
			}
		}
	}

	sb.WriteString(s[lastIdx:])
	if len(stack) > 0 {
		sb.WriteString("\033[0m")
	}

	return sb.String()
}
