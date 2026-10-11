package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

type varTargetInfo struct {
	isTask         bool
	taskReturnType string
	unitType       string
}

func findVariableTargetInfo(lines []string, varName string) *varTargetInfo {
	reVar := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?(?:let|const)\s+%s(?:\s*:\s*([^=]+?))?(?:\s*=\s*(.+))?$`, regexp.QuoteMeta(varName)))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if m := reVar.FindStringSubmatch(line); len(m) > 1 {
			typeStr := strings.TrimSpace(m[1])
			valStr := strings.TrimSpace(m[2])

			// 1. Explicit task type
			if strings.HasPrefix(typeStr, "task[") && strings.HasSuffix(typeStr, "]") {
				inner := strings.TrimSuffix(strings.TrimPrefix(typeStr, "task["), "]")
				return &varTargetInfo{isTask: true, taskReturnType: inner}
			} else if typeStr == "task" {
				return &varTargetInfo{isTask: true, taskReturnType: "any"}
			}

			// 2. Assigned with `with <expr>`
			if strings.HasPrefix(valStr, "with ") {
				withTarget := strings.TrimSpace(strings.TrimPrefix(valStr, "with "))
				retType := "any"
				if strings.Contains(strings.ToLower(withTarget), "net.fetch") {
					retType = "string"
				} else if parenIdx := strings.Index(withTarget, "("); parenIdx != -1 {
					memberPart := withTarget[:parenIdx]
					if dot := strings.LastIndex(memberPart, "."); dot != -1 {
						callObj := strings.TrimSpace(memberPart[:dot])
						callMethod := strings.TrimSpace(memberPart[dot+1:])
						retType = inferUnitMethodReturnType(lines, callObj, callMethod)
					}
				}
				if typeStr != "" && typeStr != "any" {
					retType = typeStr
				}
				return &varTargetInfo{isTask: true, taskReturnType: retType}
			}

			// 3. Instantiated via @UnitName()
			if strings.HasPrefix(valStr, "@") {
				unitName := strings.TrimPrefix(valStr, "@")
				if parenIdx := strings.Index(unitName, "("); parenIdx != -1 {
					unitName = unitName[:parenIdx]
				}
				return &varTargetInfo{unitType: strings.TrimSpace(unitName)}
			}

			// 4. Custom unit type in annotation
			if typeStr != "" && typeStr != "string" && typeStr != "number" && typeStr != "bool" && typeStr != "duration" && typeStr != "any" {
				return &varTargetInfo{unitType: typeStr}
			}
		}
	}
	return nil
}

func inferUnitMethodReturnType(lines []string, objName, methodName string) string {
	unitName := objName
	if varInfo := findVariableTargetInfo(lines, objName); varInfo != nil && varInfo.unitType != "" {
		unitName = varInfo.unitType
	}

	// 1. Built-in units
	if unitDoc, ok := BuiltinUnitsDocs[strings.ToLower(unitName)]; ok {
		if mDoc, ok := unitDoc.Methods[strings.ToLower(methodName)]; ok {
			if arrowIdx := strings.Index(mDoc.Detail, "->"); arrowIdx != -1 {
				return strings.TrimSpace(mDoc.Detail[arrowIdx+2:])
			}
		}
	}

	// 2. Local unit declaration in document
	reUnitStart := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\b`, regexp.QuoteMeta(unitName)))
	reFunc := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?func\s+%s\s*\(.*?\)(?:\s*->\s*([a-zA-Z0-9_\[\]]+))?`, regexp.QuoteMeta(methodName)))

	inUnit := false
	depth := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !inUnit {
			if reUnitStart.MatchString(l) {
				inUnit = true
				depth += strings.Count(l, "{") - strings.Count(l, "}")
			}
			continue
		}

		depth += strings.Count(l, "{") - strings.Count(l, "}")
		if depth <= 0 {
			break
		}

		if m := reFunc.FindStringSubmatch(l); len(m) > 0 {
			if len(m) > 1 && m[1] != "" {
				return strings.TrimSpace(m[1])
			}
			return "void"
		}
	}

	return "any"
}
