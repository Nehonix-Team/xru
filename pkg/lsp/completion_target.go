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
	isString       bool
	isList         bool
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

			// 4. String type
			if typeStr == "string" || strings.HasPrefix(valStr, "\"") || strings.HasPrefix(valStr, "'") {
				return &varTargetInfo{isString: true}
			}

			// 5. List type
			if strings.HasPrefix(typeStr, "list") || strings.HasPrefix(valStr, "[") {
				return &varTargetInfo{isList: true}
			}

			// 6. Custom unit type in annotation
			if typeStr != "" && typeStr != "number" && typeStr != "bool" && typeStr != "duration" && typeStr != "any" {
				return &varTargetInfo{unitType: typeStr}
			}
		}
	}

	// 7. Check file statement parameters (doc: Document, err: string)
	reFileStmt := regexp.MustCompile(`^\s*file\s+.*?(?:,\s*)?\((.*?)\)\s*(?:->|=>)`)
	for _, line := range lines {
		if m := reFileStmt.FindStringSubmatch(line); len(m) > 1 {
			params := strings.Split(m[1], ",")
			if len(params) > 0 && strings.TrimSpace(params[0]) == varName {
				return &varTargetInfo{unitType: "Document"}
			}
			if len(params) > 1 && strings.TrimSpace(params[1]) == varName {
				return &varTargetInfo{isString: true}
			}
		}
	}

	// 8. Check func parameters
	reFunc := regexp.MustCompile(`^\s*(?:global\s+)?func\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\((.*?)\)`)
	for _, line := range lines {
		if m := reFunc.FindStringSubmatch(line); len(m) > 1 {
			for _, p := range strings.Split(m[1], ",") {
				parts := strings.Split(p, ":")
				if len(parts) > 0 && strings.TrimSpace(parts[0]) == varName {
					pType := "any"
					if len(parts) > 1 {
						pType = strings.TrimSpace(parts[1])
					}
					if pType == "string" {
						return &varTargetInfo{isString: true}
					}
					if strings.HasPrefix(pType, "list") {
						return &varTargetInfo{isList: true}
					}
					if pType != "" && pType != "any" && pType != "number" && pType != "bool" && pType != "void" {
						return &varTargetInfo{unitType: pType}
					}
				}
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
