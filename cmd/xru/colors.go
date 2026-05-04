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
	tags := []string{"red", "green", "yellow", "blue", "magenta", "cyan", "gray", "white"}
	colors := []string{colorRed, colorGreen, colorYellow, colorBlue, colorMagenta, colorCyan, colorGray, colorWhite}

	for i, tag := range tags {
		s = strings.ReplaceAll(s, "<"+tag+">", colors[i])
		s = strings.ReplaceAll(s, "</"+tag+">", colorReset)
	}
	s = strings.ReplaceAll(s, "</>", colorReset)
	return s
}
