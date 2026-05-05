package engine

import "strings"

const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
	ColorWhite   = "\033[97m"
)

func Colorify(s string) string {
	tags := []string{"red", "green", "yellow", "blue", "magenta", "cyan", "gray", "white"}
	colors := []string{ColorRed, ColorGreen, ColorYellow, ColorBlue, ColorMagenta, ColorCyan, ColorGray, ColorWhite}

	for i, tag := range tags {
		s = strings.ReplaceAll(s, "<"+tag+">", colors[i])
		s = strings.ReplaceAll(s, "</"+tag+">", ColorReset)
	}
	s = strings.ReplaceAll(s, "</>", ColorReset)
	return s
}

func colorify(s string) string { return Colorify(s) }

