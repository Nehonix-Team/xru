package main

import "strings"

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
	colorWhite   = "\033[97m"
)

func colorify(s string) string {
	s = strings.ReplaceAll(s, "<red>", colorRed)
	s = strings.ReplaceAll(s, "<green>", colorGreen)
	s = strings.ReplaceAll(s, "<yellow>", colorYellow)
	s = strings.ReplaceAll(s, "<blue>", colorBlue)
	s = strings.ReplaceAll(s, "<magenta>", colorMagenta)
	s = strings.ReplaceAll(s, "<cyan>", colorCyan)
	s = strings.ReplaceAll(s, "<gray>", colorGray)
	s = strings.ReplaceAll(s, "<white>", colorWhite)
	s = strings.ReplaceAll(s, "</>", colorReset)
	return s
}
