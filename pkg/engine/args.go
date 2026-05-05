package engine

import (
	"strconv"
	"strings"
)

// getTerminalArg retourne la valeur d'un argument passé au script xru.
// Si key est un nombre, c'est un argument positionnel (1-indexé).
// Sinon, c'est un flag (ex: --env prod).
func getTerminalArg(key string, r *Runner) string {
	if idx, err := strconv.Atoi(key); err == nil {
		if idx > 0 && idx <= len(r.TerminalArgs) {
			return r.TerminalArgs[idx-1]
		}
		return ""
	}

	for i := 0; i < len(r.TerminalArgs); i++ {
		arg := r.TerminalArgs[i]
		if arg == key {
			if i+1 < len(r.TerminalArgs) && !strings.HasPrefix(r.TerminalArgs[i+1], "-") {
				return r.TerminalArgs[i+1]
			}
			return "true"
		}
		if strings.HasPrefix(arg, key+"=") {
			return strings.TrimPrefix(arg, key+"=")
		}
	}
	return ""
}

