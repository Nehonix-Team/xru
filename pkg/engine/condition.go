package engine

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/util" 
)

// evalCondition évalue une condition interpolée et retourne true/false.
func evalCondition(cond string, scope *Scope, cb string, r *Runner) bool {
	cond = util.Interpolate(cond, scope)
	cond = strings.Trim(cond, "\"' ")

	negate := false
	if strings.HasPrefix(cond, "!") {
		negate = true
		cond = strings.TrimSpace(cond[1:])
	}

	result := evalRaw(cond, cb)

	if negate {
		return !result
	}
	return result
}

func evalRaw(cond, cb string) bool {
	if strings.HasPrefix(cond, "exists(") && strings.HasSuffix(cond, ")") {
		path := strings.Trim(cond[7:len(cond)-1], "\"' ")
		absPath := filepath.Join(cb, path)
		_, err := os.Stat(absPath)
		return err == nil
	}
	if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		return strings.Trim(parts[0], "\"' ") == strings.Trim(parts[1], "\"' ")
	}
	if strings.Contains(cond, "!=") {
		parts := strings.SplitN(cond, "!=", 2)
		return strings.Trim(parts[0], "\"' ") != strings.Trim(parts[1], "\"' ")
	}
	return cond == "true"
}

