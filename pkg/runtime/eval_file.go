package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalFile(fileStmt *ast.FileStatement, env *Environment) (Value, error) {
	// 1. Résolution dynamique du chemin cible
	rawPath := fileStmt.Path
	if fileStmt.PathExpr != nil {
		pathVal, err := ev.Eval(fileStmt.PathExpr, env)
		if err != nil {
			return nil, err
		}
		rawPath = pathVal.String()
	} else {
		for k, v := range env.All() {
			rawPath = strings.ReplaceAll(rawPath, fmt.Sprintf("{%s}", k), v.String())
		}
	}

	targetPath := rawPath
	if !filepath.IsAbs(targetPath) && env.CurrentDir() != "" {
		targetPath = filepath.Join(env.CurrentDir(), targetPath)
	}

	// 2. Lecture du fichier
	var readErr error
	var content string
	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		readErr = err
	} else {
		content = string(contentBytes)
	}

	modified := false

	// Fonctions utilitaires internes pour la synchronisation
	syncLines := func() ListValue {
		rawLines := strings.Split(content, "\n")
		var elems []Value
		for _, l := range rawLines {
			elems = append(elems, StringValue{Val: l})
		}
		return ListValue{Elements: elems}
	}

	// Création de l'instance Multi-Vues Document
	docUnit := &UnitInstance{
		UnitName: "Document",
		Properties: map[string]Value{
			"path":       StringValue{Val: targetPath},
			"content":    StringValue{Val: content},
			"lines":      syncLines(),
			"length":     NumberValue{Val: float64(len(content))},
			"isModified": BoolValue{Val: false},
		},
		Methods:    make(map[string]NativeMethod),
		Directives: make(map[string]NativeDirective),
	}

	updateContent := func(newContent string) {
		content = newContent
		modified = true
		docUnit.Properties["content"] = StringValue{Val: content}
		docUnit.Properties["lines"] = syncLines()
		docUnit.Properties["length"] = NumberValue{Val: float64(len(content))}
		docUnit.Properties["isModified"] = BoolValue{Val: true}
	}

	// === VUE 1 : CONTENU BRUT ===
	docUnit.Methods["getContent"] = func(args []Value) (Value, error) {
		return StringValue{Val: content}, nil
	}
	docUnit.Methods["setContent"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("setContent requires 1 string argument")
		}
		updateContent(args[0].String())
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["append"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("append requires 1 string argument")
		}
		updateContent(content + args[0].String())
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["prepend"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("prepend requires 1 string argument")
		}
		updateContent(args[0].String() + content)
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["contains"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("contains requires 1 string argument")
		}
		return BoolValue{Val: strings.Contains(content, args[0].String())}, nil
	}
	docUnit.Methods["size"] = func(args []Value) (Value, error) {
		return NumberValue{Val: float64(len(content))}, nil
	}

	// === VUE 2 : LIGNES (LINEWISE) ===
	docUnit.Methods["getLines"] = func(args []Value) (Value, error) {
		return syncLines(), nil
	}
	docUnit.Methods["setLines"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("setLines requires 1 list argument")
		}
		listVal, ok := args[0].(ListValue)
		if !ok {
			return NilValue{}, fmt.Errorf("setLines expects list argument")
		}
		var newLines []string
		for _, el := range listVal.Elements {
			newLines = append(newLines, el.String())
		}
		updateContent(strings.Join(newLines, "\n"))
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["getLine"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("getLine requires line number (1-indexed)")
		}
		n, ok := args[0].(NumberValue)
		if !ok {
			return NilValue{}, fmt.Errorf("getLine expects number argument")
		}
		lines := strings.Split(content, "\n")
		idx := int(n.Val) - 1
		if idx < 0 || idx >= len(lines) {
			return StringValue{Val: ""}, nil
		}
		return StringValue{Val: lines[idx]}, nil
	}
	docUnit.Methods["setLine"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("setLine requires line number (1-indexed) and replacement text")
		}
		n, ok := args[0].(NumberValue)
		if !ok {
			return NilValue{}, fmt.Errorf("setLine expects number for line number")
		}
		lines := strings.Split(content, "\n")
		idx := int(n.Val) - 1
		if idx < 0 || idx >= len(lines) {
			return NilValue{}, fmt.Errorf("line index %d out of bounds (1..%d)", int(n.Val), len(lines))
		}
		lines[idx] = args[1].String()
		updateContent(strings.Join(lines, "\n"))
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["removeLine"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("removeLine requires line number (1-indexed) or pattern string")
		}
		if n, ok := args[0].(NumberValue); ok {
			lines := strings.Split(content, "\n")
			idx := int(n.Val) - 1
			if idx >= 0 && idx < len(lines) {
				lines = append(lines[:idx], lines[idx+1:]...)
				updateContent(strings.Join(lines, "\n"))
			}
			return BoolValue{Val: true}, nil
		}
		pattern := args[0].String()
		lines := strings.Split(content, "\n")
		for idx, l := range lines {
			if strings.Contains(l, pattern) {
				lines = append(lines[:idx], lines[idx+1:]...)
				updateContent(strings.Join(lines, "\n"))
				break
			}
		}
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["removeLines"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("removeLines requires pattern string or startLine/endLine (1-indexed)")
		}
		cleanGap := false
		if len(args) == 2 {
			if b, ok := args[1].(BoolValue); ok {
				cleanGap = b.Val
			}
		}
		// If 1 arg, or 2 args where 2nd is cleanGap bool
		if len(args) == 1 || (len(args) == 2 && cleanGap) {
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			var newLines []string
			for _, l := range lines {
				if !strings.Contains(l, pattern) {
					newLines = append(newLines, l)
				}
			}
			newContent := strings.Join(newLines, "\n")
			if cleanGap {
				reGap := regexp.MustCompile(`(\r?\n[ \t]*){3,}`)
				newContent = reGap.ReplaceAllString(newContent, "\n\n")
			}
			updateContent(newContent)
			return BoolValue{Val: true}, nil
		}
		startN, ok1 := args[0].(NumberValue)
		endN, ok2 := args[1].(NumberValue)
		if !ok1 || !ok2 {
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			var newLines []string
			for _, l := range lines {
				if !strings.Contains(l, pattern) {
					newLines = append(newLines, l)
				}
			}
			newContent := strings.Join(newLines, "\n")
			updateContent(newContent)
			return BoolValue{Val: true}, nil
		}
		lines := strings.Split(content, "\n")
		startIdx := int(startN.Val) - 1
		endIdx := int(endN.Val) - 1
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx >= len(lines) {
			endIdx = len(lines) - 1
		}
		if startIdx <= endIdx && startIdx < len(lines) {
			lines = append(lines[:startIdx], lines[endIdx+1:]...)
			newContent := strings.Join(lines, "\n")
			if len(args) > 2 {
				if b, ok := args[2].(BoolValue); ok && b.Val {
					reGap := regexp.MustCompile(`(\r?\n[ \t]*){3,}`)
					newContent = reGap.ReplaceAllString(newContent, "\n\n")
				}
			}
			updateContent(newContent)
		}
		return BoolValue{Val: true}, nil
	}

	// === VUE 3 : JSON STRUCTURÉ ===
	docUnit.Methods["getJson"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("getJson requires keyPath argument")
		}
		keyPath := args[0].String()
		var data any
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return NilValue{}, fmt.Errorf("invalid json in %s: %w", targetPath, err)
		}
		keys := strings.Split(keyPath, ".")
		cur := data
		for _, k := range keys {
			if m, ok := cur.(map[string]any); ok {
				cur = m[k]
			} else {
				return NilValue{}, nil
			}
		}
		return goValueToXruValue(cur), nil
	}
	docUnit.Methods["setJson"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("setJson requires keyPath and value arguments")
		}
		keyPath := args[0].String()
		val := xruValueToGoValue(args[1])
		var data map[string]any
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			data = make(map[string]any)
		}
		keys := strings.Split(keyPath, ".")
		setNestedJson(data, keys, val)
		updatedBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return NilValue{}, err
		}
		updateContent(string(updatedBytes))
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["removeJson"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("removeJson requires keyPath argument")
		}
		keyPath := args[0].String()
		var data map[string]any
		if err := json.Unmarshal([]byte(content), &data); err != nil {
			return NilValue{}, err
		}
		keys := strings.Split(keyPath, ".")
		removeNestedJson(data, keys)
		updatedBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return NilValue{}, err
		}
		updateContent(string(updatedBytes))
		return BoolValue{Val: true}, nil
	}

	// === VUE 4 : OPÉRATIONS CHIRURGICALES & SÉMANTIQUES ===
	docUnit.Methods["replace"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("replace requires target string and replacement")
		}
		target := args[0].String()
		repl := args[1].String()
		if strings.Contains(content, target) {
			updateContent(strings.ReplaceAll(content, target, repl))
		}
		return BoolValue{Val: true}, nil
	}

	docUnit.Methods["replaceFuzzy"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("replaceFuzzy requires target pattern and replacement")
		}
		target := args[0].String()
		repl := args[1].String()
		re, err := makeFuzzyRegex(target)
		if err != nil {
			return NilValue{}, err
		}
		if re.MatchString(content) {
			updateContent(re.ReplaceAllString(content, repl))
		}
		return BoolValue{Val: true}, nil
	}

	docUnit.Methods["remove"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("remove requires target string")
		}
		target := args[0].String()
		if strings.Contains(content, target) {
			updateContent(strings.ReplaceAll(content, target, ""))
		}
		return BoolValue{Val: true}, nil
	}

	docUnit.Methods["removeFuzzy"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("removeFuzzy requires target pattern")
		}
		target := args[0].String()
		re, err := makeFuzzyRegex(target)
		if err != nil {
			return NilValue{}, err
		}
		if re.MatchString(content) {
			updateContent(re.ReplaceAllString(content, ""))
		}
		return BoolValue{Val: true}, nil
	}

	docUnit.Methods["insertAfter"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("insertAfter requires anchor and content to insert")
		}
		anchor := args[0].String()
		toInsert := args[1].String()
		idx := strings.Index(content, anchor)
		if idx != -1 {
			pos := idx + len(anchor)
			updateContent(content[:pos] + toInsert + content[pos:])
		}
		return BoolValue{Val: true}, nil
	}

	docUnit.Methods["insertBefore"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("insertBefore requires anchor and content to insert")
		}
		anchor := args[0].String()
		toInsert := args[1].String()
		idx := strings.Index(content, anchor)
		if idx != -1 {
			updateContent(content[:idx] + toInsert + content[idx:])
		}
		return BoolValue{Val: true}, nil
	}

	// removeBetween(startPattern, endPattern, includeBounds = true, cleanGap = false)
	docUnit.Methods["removeBetween"] = func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("removeBetween requires startPattern and endPattern")
		}
		startPattern := args[0].String()
		endPattern := args[1].String()
		includeBounds := true
		cleanGap := false

		if len(args) > 2 {
			if b, ok := args[2].(BoolValue); ok {
				includeBounds = b.Val
			} else if mv, ok := args[2].(MapValue); ok {
				if inc, exists := mv.Pairs["includeBounds"]; exists {
					if b, ok := inc.(BoolValue); ok {
						includeBounds = b.Val
					}
				}
				if cg, exists := mv.Pairs["cleanGap"]; exists {
					if b, ok := cg.(BoolValue); ok {
						cleanGap = b.Val
					}
				}
			}
		}
		if len(args) > 3 {
			if b, ok := args[3].(BoolValue); ok {
				cleanGap = b.Val
			}
		}

		startRe, err := makeFuzzyRegex(startPattern)
		if err != nil {
			return NilValue{}, err
		}
		endRe, err := makeFuzzyRegex(endPattern)
		if err != nil {
			return NilValue{}, err
		}

		startLoc := startRe.FindStringIndex(content)
		if startLoc == nil {
			return BoolValue{Val: false}, nil
		}

		// Recherche endPattern après la position de startPattern
		subContent := content[startLoc[1]:]
		endLoc := endRe.FindStringIndex(subContent)
		if endLoc == nil {
			return BoolValue{Val: false}, nil
		}

		actualEndStart := startLoc[1] + endLoc[0]
		actualEndEnd := startLoc[1] + endLoc[1]

		if cleanGap {
			startIndex := startLoc[0]
			endIndex := actualEndEnd
			if !includeBounds {
				startIndex = startLoc[1]
				endIndex = actualEndStart
			}

			// Consommer les espaces, tabs, virgules, points-virgules et saut de ligne après endPattern
			for endIndex < len(content) && (content[endIndex] == ';' || content[endIndex] == ',' || content[endIndex] == ' ' || content[endIndex] == '\t') {
				endIndex++
			}
			if endIndex < len(content) && content[endIndex] == '\r' {
				endIndex++
			}
			if endIndex < len(content) && content[endIndex] == '\n' {
				endIndex++
			}

			// Si la ligne avant startIndex ne contient que des espaces d'indentation, reculer
			lineStart := startIndex
			for lineStart > 0 && (content[lineStart-1] == ' ' || content[lineStart-1] == '\t') {
				lineStart--
			}
			if lineStart == 0 || content[lineStart-1] == '\n' {
				startIndex = lineStart
			}

			newContent := content[:startIndex] + content[endIndex:]
			reGap := regexp.MustCompile(`(\r?\n[ \t]*){3,}`)
			newContent = reGap.ReplaceAllString(newContent, "\n\n")
			updateContent(newContent)
		} else {
			if includeBounds {
				updateContent(content[:startLoc[0]] + content[actualEndEnd:])
			} else {
				updateContent(content[:startLoc[1]] + content[actualEndStart:])
			}
		}
		return BoolValue{Val: true}, nil
	}

	// removeBalanced(startAnchor, openChar = "{", closeChar = "}")
	docUnit.Methods["removeBalanced"] = func(args []Value) (Value, error) {
		if len(args) < 1 {
			return NilValue{}, fmt.Errorf("removeBalanced requires startAnchor")
		}
		startAnchor := args[0].String()
		openChar := "{"
		closeChar := "}"
		cleanGap := false
		if len(args) > 1 {
			openChar = args[1].String()
		}
		if len(args) > 2 {
			closeChar = args[2].String()
		}
		if len(args) > 3 {
			if b, ok := args[3].(BoolValue); ok {
				cleanGap = b.Val
			}
		}

		anchorRe, err := makeFuzzyRegex(startAnchor)
		if err != nil {
			return NilValue{}, err
		}
		loc := anchorRe.FindStringIndex(content)
		if loc == nil {
			return BoolValue{Val: false}, nil
		}

		startIndex := loc[0]
		var curr int
		if idx := strings.Index(content[loc[0]:loc[1]], openChar); idx != -1 {
			curr = loc[0] + idx
		} else {
			openIndex := strings.Index(content[loc[1]:], openChar)
			if openIndex == -1 {
				return BoolValue{Val: false}, nil
			}
			curr = loc[1] + openIndex
		}

		// Algorithme d'appariement équilibré ignorant les guillemets
		depth := 1
		i := curr + len(openChar)
		inString := byte(0)
		endIndex := -1

		for i < len(content) {
			ch := content[i]
			if inString != 0 {
				if ch == '\\' && i+1 < len(content) {
					i += 2
					continue
				}
				if ch == inString {
					inString = 0
				}
				i++
				continue
			}

			if ch == '"' || ch == '\'' || ch == '`' {
				inString = ch
				i++
				continue
			}

			if strings.HasPrefix(content[i:], openChar) {
				depth++
				i += len(openChar)
				continue
			}
			if strings.HasPrefix(content[i:], closeChar) {
				depth--
				if depth == 0 {
					endIndex = i + len(closeChar)
					break
				}
				i += len(closeChar)
				continue
			}
			i++
		}

		if endIndex != -1 {
			// Englober le point-virgule, virgule ou espaces qui suivent
			for endIndex < len(content) && (content[endIndex] == ';' || content[endIndex] == ',' || content[endIndex] == ' ' || content[endIndex] == '\t') {
				endIndex++
			}
			if endIndex < len(content) && content[endIndex] == '\r' {
				endIndex++
			}
			if endIndex < len(content) && content[endIndex] == '\n' {
				endIndex++
			}

			// Si la ligne avant startIndex ne contient que des espaces d'indentation, reculer
			lineStart := startIndex
			for lineStart > 0 && (content[lineStart-1] == ' ' || content[lineStart-1] == '\t') {
				lineStart--
			}
			if lineStart == 0 || content[lineStart-1] == '\n' {
				startIndex = lineStart
			}

			newContent := content[:startIndex] + content[endIndex:]
			if cleanGap {
				reGap := regexp.MustCompile(`(\r?\n[ \t]*){3,}`)
				newContent = reGap.ReplaceAllString(newContent, "\n\n")
			}
			updateContent(newContent)
			return BoolValue{Val: true}, nil
		}

		return BoolValue{Val: false}, nil
	}

	docUnit.Methods["cleanGaps"] = func(args []Value) (Value, error) {
		reGap := regexp.MustCompile(`(\r?\n[ \t]*){3,}`)
		cleaned := reGap.ReplaceAllString(content, "\n\n")
		reEmptyLine := regexp.MustCompile(`\n[ \t]+\n`)
		cleaned = reEmptyLine.ReplaceAllString(cleaned, "\n\n")
		updateContent(cleaned)
		return BoolValue{Val: true}, nil
	}
	docUnit.Methods["cleanGap"] = docUnit.Methods["cleanGaps"]

	docUnit.Methods["save"] = func(args []Value) (Value, error) {
		err := os.WriteFile(targetPath, []byte(content), 0644)
		if err != nil {
			return NilValue{}, err
		}
		modified = false
		docUnit.Properties["isModified"] = BoolValue{Val: false}
		return BoolValue{Val: true}, nil
	}

	// 3. Exécution moderne via Handler Lambda (doc, err) -> { ... }
	if fileStmt.Handler != nil {
		var errVal Value = NilValue{}
		if readErr != nil {
			errVal = StringValue{Val: readErr.Error()}
		}

		// Si l'utilisateur n'a déclaré qu'un paramètre (doc) et qu'il y a une erreur de lecture -> crash
		executeHandler := func(fnVal Value, callArgs []Value) error {
			if lambda, ok := fnVal.(LambdaValue); ok {
				_, err := ev.CallLambda(lambda, callArgs)
				return err
			}
			if fn, ok := fnVal.(FunctionValue); ok {
				_, err := fn.Fn(callArgs)
				return err
			}
			return fmt.Errorf("file handler must be a function or lambda, got %s", fnVal.Type())
		}

		if lambda, ok := fileStmt.Handler.(*ast.LambdaExpression); ok {
			if len(lambda.Parameters) < 2 && readErr != nil {
				return NilValue{}, fmt.Errorf("failed to open file '%s': %w", targetPath, readErr)
			}
			fnVal, err := ev.Eval(fileStmt.Handler, env)
			if err != nil {
				return nil, err
			}

			isErrFirst := false
			if len(lambda.Parameters) >= 2 {
				p0Name := strings.ToLower(lambda.Parameters[0].Name)
				p1Name := strings.ToLower(lambda.Parameters[1].Name)
				p1Type := lambda.Parameters[1].Type
				if p1Type == "Document" || ((p0Name == "e" || p0Name == "err" || p0Name == "error") && (p1Name == "data" || p1Name == "d" || p1Name == "doc" || p1Name == "content" || p1Type == "Document")) {
					isErrFirst = true
				}
			}

			var callArgs []Value
			if len(lambda.Parameters) == 1 {
				p0Name := strings.ToLower(lambda.Parameters[0].Name)
				if p0Name == "e" || p0Name == "err" || p0Name == "error" {
					callArgs = append(callArgs, errVal)
				} else {
					callArgs = append(callArgs, docUnit)
				}
			} else if len(lambda.Parameters) >= 2 {
				if isErrFirst {
					callArgs = append(callArgs, errVal, docUnit)
				} else {
					callArgs = append(callArgs, docUnit, errVal)
				}
			} else {
				callArgs = append(callArgs, docUnit)
			}

			if err := executeHandler(fnVal, callArgs); err != nil {
				return nil, err
			}
		} else {
			fnVal, err := ev.Eval(fileStmt.Handler, env)
			if err != nil {
				return nil, err
			}
			if err := executeHandler(fnVal, []Value{errVal, docUnit}); err != nil {
				return nil, err
			}
		}

		// Auto-sauvegarde à la sortie si modifié et sans erreur
		if modified && readErr == nil {
			_ = os.WriteFile(targetPath, []byte(content), 0644)
		}
		return docUnit, nil
	}

	// 4. Rétrocompatibilité : Ancien bloc direct file "path" { ... }
	if len(fileStmt.Body) > 0 {
		if readErr != nil {
			return NilValue{}, fmt.Errorf("failed to open file '%s': %w", targetPath, readErr)
		}
		fileEnv := NewEnvironment(env)
		fileEnv.SetCurrentDir(filepath.Dir(targetPath))

		// Injection des méthodes avec alias PascalCase et minuscules
		for mName, mFn := range docUnit.Methods {
			fn := mFn
			fileEnv.Set(mName, FunctionValue{Name: mName, Fn: fn}, false)
			fileEnv.Set(strings.ToLower(mName), FunctionValue{Name: strings.ToLower(mName), Fn: fn}, false)
			fileEnv.Set(strings.Title(mName), FunctionValue{Name: strings.Title(mName), Fn: fn}, false)
		}

		fileEnv.Set("RemoveBlock", FunctionValue{
			Name: "RemoveBlock",
			Fn: func(args []Value) (Value, error) {
				return docUnit.Methods["removeBetween"](args)
			},
		}, false)
		fileEnv.Set("removeBlock", FunctionValue{
			Name: "removeBlock",
			Fn: func(args []Value) (Value, error) {
				return docUnit.Methods["removeBetween"](args)
			},
		}, false)
		fileEnv.Set("RemoveLines", FunctionValue{
			Name: "RemoveLines",
			Fn: func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("RemoveLines requires pattern")
				}
				pat := args[0].String()
				lines := strings.Split(content, "\n")
				var rem []string
				for _, l := range lines {
					if !strings.Contains(l, pat) {
						rem = append(rem, l)
					}
				}
				updateContent(strings.Join(rem, "\n"))
				return BoolValue{Val: true}, nil
			},
		}, false)

		for _, stmt := range fileStmt.Body {
			if _, err := ev.Eval(stmt, fileEnv); err != nil {
				return nil, err
			}
		}

		if modified {
			_ = os.WriteFile(targetPath, []byte(content), 0644)
		}
		return docUnit, nil
	}

	return docUnit, nil
}

// makeFuzzyRegex transforme une chaîne en regex tolérante aux espaces multiples et retours à la ligne
func makeFuzzyRegex(pattern string) (*regexp.Regexp, error) {
	tokens := strings.Fields(pattern)
	if len(tokens) == 0 {
		return regexp.Compile("")
	}
	var escapedTokens []string
	for _, tok := range tokens {
		escapedTokens = append(escapedTokens, regexp.QuoteMeta(tok))
	}
	fuzzyPattern := strings.Join(escapedTokens, `\s+`)
	return regexp.Compile(fuzzyPattern)
}

func setNestedJson(data map[string]any, keys []string, value any) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		data[keys[0]] = value
		return
	}
	next, exists := data[keys[0]]
	if !exists {
		nextMap := make(map[string]any)
		data[keys[0]] = nextMap
		setNestedJson(nextMap, keys[1:], value)
		return
	}
	if nextMap, ok := next.(map[string]any); ok {
		setNestedJson(nextMap, keys[1:], value)
	} else {
		nextMap := make(map[string]any)
		data[keys[0]] = nextMap
		setNestedJson(nextMap, keys[1:], value)
	}
}

func removeNestedJson(data map[string]any, keys []string) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		delete(data, keys[0])
		return
	}
	if nextMap, ok := data[keys[0]].(map[string]any); ok {
		removeNestedJson(nextMap, keys[1:])
	}
}
