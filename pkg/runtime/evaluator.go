package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

type Environment struct {
	parent     *Environment
	store      map[string]Value
	globals    map[string]Value
	constants  map[string]bool
	unitRef    *UnitInstance // Pointeur vers l'unité courante si dans #UNIT
	currentDir string
}

func NewEnvironment(parent *Environment) *Environment {
	var g map[string]Value
	if parent != nil {
		g = parent.globals
	} else {
		g = make(map[string]Value)
	}
	return &Environment{
		parent:    parent,
		store:     make(map[string]Value),
		globals:   g,
		constants: make(map[string]bool),
	}
}

func (e *Environment) SetGlobal(name string, val Value) {
	e.globals[name] = val
}

func (e *Environment) SetCurrentDir(dir string) {
	e.currentDir = dir
}

func (e *Environment) CurrentDir() string {
	if e.currentDir != "" {
		return e.currentDir
	}
	if e.parent != nil {
		return e.parent.CurrentDir()
	}
	return ""
}

func (e *Environment) All() map[string]Value {
	res := make(map[string]Value)
	for k, v := range e.globals {
		res[k] = v
	}
	if e.parent != nil {
		for k, v := range e.parent.All() {
			res[k] = v
		}
	}
	for k, v := range e.store {
		res[k] = v
	}
	return res
}

func (e *Environment) Set(name string, val Value, isConst bool) error {
	if e.constants[name] {
		return fmt.Errorf("cannot reassign constant '%s'", name)
	}
	e.store[name] = val
	if isConst {
		e.constants[name] = true
	}
	return nil
}

func (e *Environment) Get(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if val, ok := e.globals[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

func (e *Environment) GetLocal(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.GetLocal(name)
	}
	return nil, false
}

func (e *Environment) GetGlobal(name string) (Value, bool) {
	if val, ok := e.globals[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.GetGlobal(name)
	}
	return nil, false
}

type TreeDispatcherFunc func(dt *ast.DispatchTreeStatement, eval *Evaluator, env *Environment) error

type Evaluator struct {
	builtins       map[string]*UnitInstance
	treeDispatcher TreeDispatcherFunc
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		builtins: NewBuiltinUnits(),
	}
}

func (ev *Evaluator) SetTreeDispatcher(fn TreeDispatcherFunc) {
	ev.treeDispatcher = fn
}

func (ev *Evaluator) Eval(node ast.Node, env *Environment) (Value, error) {
	switch n := node.(type) {
	case *ast.Program:
		return ev.evalProgram(n, env)
	case *ast.VarStatement:
		val, err := ev.Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		return val, env.Set(n.Name, val, n.IsConst)
	case *ast.ArgStatement:
		// S'il existe déjà dans l'environnement (fourni par les CLI args), on le garde
		if _, exists := env.Get(n.Name); !exists && n.DefaultValue != nil {
			val, err := ev.Eval(n.DefaultValue, env)
			if err != nil {
				return nil, err
			}
			return val, env.Set(n.Name, val, false)
		}
		return NilValue{}, nil
	case *ast.UseStatement:
		// Enregistre l'unité native avec son alias
		unitName := n.Module
		alias := n.Alias
		if alias == "" {
			alias = unitName
		}
		// Recherche insensible à la casse dans les builtins
		for bName, bUnit := range ev.builtins {
			if strings.EqualFold(bName, unitName) {
				env.Set(alias, bUnit, true)
				break
			}
		}
		return NilValue{}, nil
	case *ast.UnitStatement:
		// Enregistre la définition de l'unité
		unitInst := &UnitInstance{
			UnitName:   n.Name,
			Properties: make(map[string]Value),
			Methods:    make(map[string]NativeMethod),
			Directives: make(map[string]NativeDirective),
		}
		for _, prop := range n.Properties {
			pVal, _ := ev.Eval(prop.Value, env)
			unitInst.Properties[prop.Name] = pVal
		}
		env.Set(n.Name, unitInst, true)
		return unitInst, nil
	case *ast.InstantiationExpression:
		// @UnitName()
		val, ok := env.Get(n.UnitName)
		if !ok {
			return nil, fmt.Errorf("undefined unit '%s'", n.UnitName)
		}
		proto, ok := val.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not a unit", n.UnitName)
		}
		// Cloner l'instance
		clone := &UnitInstance{
			UnitName:   proto.UnitName,
			Properties: make(map[string]Value),
			Methods:    proto.Methods,
			Directives: proto.Directives,
		}
		for k, v := range proto.Properties {
			clone.Properties[k] = v
		}
		return clone, nil
	case *ast.ExpressionStatement:
		return ev.Eval(n.Expression, env)
	case *ast.FileStatement:
		return ev.evalFile(n, env)
	case *ast.FuncStatement:
		return ev.evalFunc(n, env)
	case *ast.ReturnStatement:
		val, err := ev.Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		return ReturnValue{Val: val}, nil
	case *ast.DispatchTreeStatement:
		return ev.evalDispatchTree(n, env)
	case *ast.CallExpression:
		return ev.evalCall(n, env)
	case *ast.GlobalCallExpression:
		return ev.evalGlobalCall(n, env)
	case *ast.GlobalAccessExpression:
		if val, ok := env.GetGlobal(n.Name); ok {
			return val, nil
		}
		return nil, fmt.Errorf("undefined global variable '%s'", n.Name)
	case *ast.WithStatement:
		return ev.evalWith(n, env)
	case *ast.IfStatement:
		return ev.evalIf(n, env)
	case *ast.Identifier:
		if val, ok := env.Get(n.Value); ok {
			return val, nil
		}
		return nil, fmt.Errorf("undefined identifier '%s'", n.Value)
	case *ast.PropAccess:
		if env.unitRef != nil {
			if val, ok := env.unitRef.Properties[n.Name]; ok {
				return val, nil
			}
		}
		return nil, fmt.Errorf("undefined property '&%s'", n.Name)
	case *ast.StringLiteral:
		// Interpolation {var}
		res := n.Value
		for k, v := range env.All() {
			placeholder := fmt.Sprintf("{%s}", k)
			res = strings.ReplaceAll(res, placeholder, v.String())
		}
		if unk := findUndefinedInterpolations(res); len(unk) > 0 {
			return NilValue{}, fmt.Errorf("undefined variable '{%s}' in string interpolation: \"%s\"", strings.Join(unk, ", {"), n.Value)
		}
		return StringValue{Val: res}, nil
	case *ast.NumberLiteral:
		num, err := strconv.ParseFloat(n.Value, 64)
		if err != nil {
			return nil, err
		}
		return NumberValue{Val: num}, nil
	case *ast.BoolLiteral:
		return BoolValue{Val: n.Value}, nil
	case *ast.ListLiteral:
		var elements []Value
		for _, el := range n.Elements {
			val, err := ev.Eval(el, env)
			if err != nil {
				return nil, err
			}
			elements = append(elements, val)
		}
		return ListValue{Elements: elements}, nil
	case *ast.InfixExpression:
		return ev.evalInfix(n, env)
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalProgram(prog *ast.Program, env *Environment) (Value, error) {
	var result Value = NilValue{}
	for _, stmt := range prog.Statements {
		res, err := ev.Eval(stmt, env)
		if err != nil {
			return nil, err
		}
		result = res
	}
	return result, nil
}

func (ev *Evaluator) evalCall(call *ast.CallExpression, env *Environment) (Value, error) {
	switch c := call.Caller.(type) {
	case *ast.MemberExpression:
		objVal, err := ev.Eval(c.Object, env)
		if err != nil {
			return nil, err
		}
		unit, ok := objVal.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an executable unit", c.Object.String())
		}
		method, ok := unit.Methods[c.Property]
		if !ok {
			for mName, mFn := range unit.Methods {
				if strings.EqualFold(mName, c.Property) {
					method = mFn
					ok = true
					break
				}
			}
		}
		if !ok {
			return nil, fmt.Errorf("method '%s' not found on unit '%s'", c.Property, unit.UnitName)
		}
		var args []Value
		for _, arg := range call.Arguments {
			a, err := ev.Eval(arg, env)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		}
		if unit.UnitName == "Fs" && env != nil && env.CurrentDir() != "" {
			for i := 0; i < len(args); i++ {
				strVal := args[i].String()
				if !filepath.IsAbs(strVal) {
					args[i] = StringValue{Val: filepath.Join(env.CurrentDir(), strVal)}
				}
			}
		}
		return method(args)
	case *ast.Identifier:
		val, ok := env.GetLocal(c.Value)
		if !ok {
			if _, isGlobal := env.GetGlobal(c.Value); isGlobal {
				return nil, fmt.Errorf("cannot call global function '%s' as local; use '::%s(...)' or 'global::%s(...)'", c.Value, c.Value, c.Value)
			}
			return nil, fmt.Errorf("undefined function '%s'", c.Value)
		}
		var args []Value
		for _, arg := range call.Arguments {
			a, err := ev.Eval(arg, env)
			if err != nil {
				return nil, err
			}
			args = append(args, a)
		}
		if fnVal, ok := val.(FunctionValue); ok {
			return fnVal.Fn(args)
		}
		if userFn, ok := val.(UserFunction); ok {
			return ev.callUserFunction(userFn, args, env)
		}
		return nil, fmt.Errorf("'%s' is not callable", c.Value)
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalGlobalCall(call *ast.GlobalCallExpression, env *Environment) (Value, error) {
	val, ok := env.GetGlobal(call.Function)
	if !ok {
		return nil, fmt.Errorf("undefined global function '%s'", call.Function)
	}
	var args []Value
	for _, arg := range call.Arguments {
		a, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	if fnVal, ok := val.(FunctionValue); ok {
		return fnVal.Fn(args)
	}
	if userFn, ok := val.(UserFunction); ok {
		return ev.callUserFunction(userFn, args, env)
	}
	return nil, fmt.Errorf("global '%s' is not callable", call.Function)
}

func (ev *Evaluator) evalWith(with *ast.WithStatement, env *Environment) (Value, error) {
	for _, stmt := range with.Body {
		_, err := ev.Eval(stmt, env)
		if err != nil {
			return nil, err
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalIf(ifStmt *ast.IfStatement, env *Environment) (Value, error) {
	condVal, err := ev.Eval(ifStmt.Condition, env)
	if err != nil {
		return nil, err
	}
	bVal, ok := condVal.(BoolValue)
	if ok && bVal.Val {
		for _, s := range ifStmt.Consequence {
			if _, err := ev.Eval(s, env); err != nil {
				return nil, err
			}
		}
		return NilValue{}, nil
	}

	for _, s := range ifStmt.Alternative {
		if _, err := ev.Eval(s, env); err != nil {
			return nil, err
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalInfix(infix *ast.InfixExpression, env *Environment) (Value, error) {
	left, err := ev.Eval(infix.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := ev.Eval(infix.Right, env)
	if err != nil {
		return nil, err
	}

	switch infix.Operator {
	case "==":
		return BoolValue{Val: left.String() == right.String()}, nil
	case "!=":
		return BoolValue{Val: left.String() != right.String()}, nil
	}
	return BoolValue{Val: false}, nil
}

func (ev *Evaluator) evalFile(fileStmt *ast.FileStatement, env *Environment) (Value, error) {
	// 1. Interpolation des variables dans le chemin du fichier
	rawPath := fileStmt.Path
	for k, v := range env.All() {
		rawPath = strings.ReplaceAll(rawPath, fmt.Sprintf("{%s}", k), v.String())
	}

	targetPath := rawPath
	if !filepath.IsAbs(targetPath) && env.CurrentDir() != "" {
		targetPath = filepath.Join(env.CurrentDir(), targetPath)
	}

	contentBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return NilValue{}, fmt.Errorf("failed to open file '%s': %w", targetPath, err)
	}
	content := string(contentBytes)
	modified := false

	// 2. Création de l'environnement confiné au fichier avec les opérations chirurgicales
	fileEnv := NewEnvironment(env)
	fileEnv.SetCurrentDir(filepath.Dir(targetPath))

	// Replace(target, replacement)
	fileEnv.Set("Replace", FunctionValue{
		Name: "Replace",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("Replace requires target string and replacement")
			}
			target := args[0].String()
			repl := args[1].String()
			if strings.Contains(content, target) {
				content = strings.ReplaceAll(content, target, repl)
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Remove(target)
	fileEnv.Set("Remove", FunctionValue{
		Name: "Remove",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Remove requires target string")
			}
			target := args[0].String()
			if strings.Contains(content, target) {
				content = strings.ReplaceAll(content, target, "")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveBlock(startPattern, endPattern)
	fileEnv.Set("RemoveBlock", FunctionValue{
		Name: "RemoveBlock",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("RemoveBlock requires start and end patterns")
			}
			startPat := args[0].String()
			endPat := args[1].String()

			sIdx := strings.Index(content, startPat)
			if sIdx != -1 {
				lineStart := strings.LastIndex(content[:sIdx], "\n")
				if lineStart == -1 {
					lineStart = 0
				} else {
					lineStart += 1
				}

				eRel := strings.Index(content[sIdx:], endPat)
				if eRel != -1 {
					eIdx := sIdx + eRel + len(endPat)
					lineEnd := strings.Index(content[eIdx:], "\n")
					if lineEnd == -1 {
						eIdx = len(content)
					} else {
						eIdx = eIdx + lineEnd + 1
					}
					content = content[:lineStart] + content[eIdx:]
					modified = true
				}
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveUntil(startPattern, untilPattern)
	fileEnv.Set("RemoveUntil", FunctionValue{
		Name: "RemoveUntil",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("RemoveUntil requires start pattern and until pattern")
			}
			startPat := args[0].String()
			untilPat := args[1].String()

			sIdx := strings.Index(content, startPat)
			if sIdx != -1 {
				lineStart := strings.LastIndex(content[:sIdx], "\n")
				if lineStart == -1 {
					lineStart = 0
				} else {
					lineStart += 1
				}

				uRel := strings.Index(content[sIdx:], untilPat)
				if uRel != -1 {
					untilIdx := sIdx + uRel
					lineUntil := strings.LastIndex(content[:untilIdx], "\n")
					if lineUntil == -1 || lineUntil < lineStart {
						lineUntil = untilIdx
					} else {
						lineUntil += 1
					}
					content = content[:lineStart] + content[lineUntil:]
					modified = true
				}
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// RemoveLines(pattern)
	fileEnv.Set("RemoveLines", FunctionValue{
		Name: "RemoveLines",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("RemoveLines requires pattern string")
			}
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			var kept []string
			changed := false
			for _, line := range lines {
				if strings.Contains(line, pattern) {
					changed = true
					continue
				}
				kept = append(kept, line)
			}
			if changed {
				content = strings.Join(kept, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// ReplaceLine(pattern, replacement)
	fileEnv.Set("ReplaceLine", FunctionValue{
		Name: "ReplaceLine",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("ReplaceLine requires pattern string and replacement")
			}
			pattern := args[0].String()
			repl := args[1].String()
			lines := strings.Split(content, "\n")
			changed := false
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					lines[i] = repl
					changed = true
				}
			}
			if changed {
				content = strings.Join(lines, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// CommentOut(pattern)
	fileEnv.Set("CommentOut", FunctionValue{
		Name: "CommentOut",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("CommentOut requires pattern string")
			}
			pattern := args[0].String()
			lines := strings.Split(content, "\n")
			changed := false
			for i, line := range lines {
				if strings.Contains(line, pattern) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
					lines[i] = "// " + line
					changed = true
				}
			}
			if changed {
				content = strings.Join(lines, "\n")
				modified = true
			}
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Append(text)
	fileEnv.Set("Append", FunctionValue{
		Name: "Append",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Append requires string")
			}
			content += "\n" + args[0].String()
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// Prepend(text)
	fileEnv.Set("Prepend", FunctionValue{
		Name: "Prepend",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("Prepend requires string")
			}
			content = args[0].String() + "\n" + content
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// JsonSet(keyPath, val)
	fileEnv.Set("JsonSet", FunctionValue{
		Name: "JsonSet",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 2 {
				return NilValue{}, fmt.Errorf("JsonSet requires key path and value")
			}
			keyPath := args[0].String()
			val := args[1].String()

			var data map[string]any
			if err := json.Unmarshal([]byte(content), &data); err != nil {
				return NilValue{}, fmt.Errorf("invalid json in %s: %w", targetPath, err)
			}
			keys := strings.Split(keyPath, ".")
			setNestedJson(data, keys, val)

			updatedBytes, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return NilValue{}, err
			}
			content = string(updatedBytes) + "\n"
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// JsonRemove(key)
	fileEnv.Set("JsonRemove", FunctionValue{
		Name: "JsonRemove",
		Fn: func(args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, fmt.Errorf("JsonRemove requires key name")
			}
			key := args[0].String()

			var data map[string]any
			if err := json.Unmarshal([]byte(content), &data); err != nil {
				return NilValue{}, fmt.Errorf("invalid json in %s: %w", targetPath, err)
			}
			delete(data, key)

			updatedBytes, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return NilValue{}, err
			}
			content = string(updatedBytes) + "\n"
			modified = true
			return BoolValue{Val: true}, nil
		},
	}, false)

	// 3. Exécution de toutes les instructions dans le scope du fichier
	for _, s := range fileStmt.Body {
		if _, err := ev.Eval(s, fileEnv); err != nil {
			return NilValue{}, err
		}
	}

	// 4. Écriture si le fichier a été modifié
	if modified {
		if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
			return NilValue{}, fmt.Errorf("failed to write file '%s': %w", targetPath, err)
		}
	}

	return BoolValue{Val: true}, nil
}

func setNestedJson(data map[string]any, keys []string, val any) {
	if len(keys) == 1 {
		data[keys[0]] = val
		return
	}
	currentKey := keys[0]
	subMap, exists := data[currentKey].(map[string]any)
	if !exists {
		subMap = make(map[string]any)
		data[currentKey] = subMap
	}
	setNestedJson(subMap, keys[1:], val)
}

func (ev *Evaluator) evalFunc(fn *ast.FuncStatement, env *Environment) (Value, error) {
	userFn := UserFunction{Stmt: fn, Env: env}
	if fn.IsGlobal {
		env.SetGlobal(fn.Name, userFn)
	} else {
		if err := env.Set(fn.Name, userFn, false); err != nil {
			return nil, err
		}
	}
	return userFn, nil
}

func (ev *Evaluator) callUserFunction(fn UserFunction, args []Value, callerEnv *Environment) (Value, error) {
	fnEnv := NewEnvironment(fn.Env)
	if callerEnv != nil {
		fnEnv.SetCurrentDir(callerEnv.CurrentDir())
		if dropVal, ok := callerEnv.Get("Drop"); ok {
			_ = fnEnv.Set("Drop", dropVal, false)
		}
		if keepVal, ok := callerEnv.Get("Keep"); ok {
			_ = fnEnv.Set("Keep", keepVal, false)
		}
		if uvVal, ok := callerEnv.Get("UseVariant"); ok {
			_ = fnEnv.Set("UseVariant", uvVal, false)
		}
		if locVal, ok := callerEnv.Get("Local"); ok {
			_ = fnEnv.Set("Local", locVal, false)
		}
	}
	for i, param := range fn.Stmt.Parameters {
		if i < len(args) {
			_ = fnEnv.Set(param.Name, args[i], false)
		}
	}
	for _, stmt := range fn.Stmt.Body {
		res, err := ev.Eval(stmt, fnEnv)
		if err != nil {
			return NilValue{}, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret.Val, nil
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalDispatchTree(dt *ast.DispatchTreeStatement, env *Environment) (Value, error) {
	treeEnv := NewEnvironment(env)
	for k, expr := range dt.PassArgs {
		val, err := ev.Eval(expr, env)
		if err != nil {
			return NilValue{}, err
		}
		treeEnv.Set(k, val, false)
	}

	if ev.treeDispatcher != nil {
		if err := ev.treeDispatcher(dt, ev, treeEnv); err != nil {
			return NilValue{}, err
		}
	}
	return BoolValue{Val: true}, nil
}

var reInterpolationIdentifier = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

func findUndefinedInterpolations(str string) []string {
	// Masquer temporairement les accolades doubles {{...}} (placeholders de fichiers)
	masked := strings.ReplaceAll(str, "{{", "\x00\x00")
	masked = strings.ReplaceAll(masked, "}}", "\x01\x01")

	matches := reInterpolationIdentifier.FindAllStringSubmatch(masked, -1)
	var undefinedVars []string
	for _, m := range matches {
		if len(m) > 1 {
			undefinedVars = append(undefinedVars, m[1])
		}
	}
	return undefinedVars
}



