package main

import (
	"strconv"
	"strings"
)

// getTerminalArg retourne la valeur d'un argument passé au script xru.
// Si key est un nombre, c'est un argument positionnel (1-indexé).
// Sinon, c'est un flag (ex: --env prod).
func getTerminalArg(key string) string {
	if idx, err := strconv.Atoi(key); err == nil {
		if idx > 0 && idx <= len(terminalArgs) {
			return terminalArgs[idx-1]
		}
		return ""
	}

	for i := 0; i < len(terminalArgs); i++ {
		arg := terminalArgs[i]
		if arg == key {
			if i+1 < len(terminalArgs) && !strings.HasPrefix(terminalArgs[i+1], "-") {
				return terminalArgs[i+1]
			}
			return "true"
		}
		if strings.HasPrefix(arg, key+"=") {
			return strings.TrimPrefix(arg, key+"=")
		}
	}
	return ""
}
