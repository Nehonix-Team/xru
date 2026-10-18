package runtime

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalCall(call *ast.CallExpression, env *Environment) (Value, error) {
	switch c := call.Caller.(type) {
	case *ast.MemberExpression:
		objVal, err := ev.Eval(c.Object, env)
		if err != nil {
			return nil, err
		}
		if task, ok := objVal.(*TaskValue); ok {
			switch strings.ToLower(c.Property) {
			case "wait", "await", "result":
				return task.Wait()
			case "isdone", "done":
				return task.IsDone(), nil
			case "cancel":
				return task.Cancel(), nil
			default:
				return nil, fmt.Errorf("method '%s' not found on task", c.Property)
			}
		}
		propLower := strings.ToLower(c.Property)
		if (propLower == "type" || propLower == "typeof") && len(call.Arguments) == 0 {
			return StringValue{Val: string(objVal.Type())}, nil
		}
		if strVal, ok := objVal.(StringValue); ok {
			return ev.evalStringMethod(c, strVal, call.Arguments, env)
		}
		if numVal, ok := objVal.(NumberValue); ok {
			return ev.evalNumberMethod(c, numVal, call.Arguments, env)
		}
		if listVal, ok := objVal.(ListValue); ok {
			return ev.evalListMethod(c, listVal, call.Arguments, env)
		}
		if mapVal, ok := objVal.(MapValue); ok {
			return ev.evalMapMethod(c, mapVal, call.Arguments, env)
		}
		unit, ok := objVal.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an executable unit", c.Object.String())
		}
		if unit.IsPrototype {
			return nil, fmt.Errorf("cannot call method '%s' directly on unit definition '%s'. You must first instantiate it with '@%s()'", c.Property, unit.UnitName, unit.UnitName)
		}
		if env != nil && env.CurrentDir() != "" {
			unit.WorkingDir = env.CurrentDir()
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
			resolveFsPathArgs(c.Property, args, env.CurrentDir())
		}
		return method(args)
	case *ast.ScopeAccessExpression:
		if c.IsUnitSelf {
			if env.unitRef == nil {
				return nil, fmt.Errorf("internal unit reference '&%s::%s' cannot be called outside a unit definition", c.Scope, c.Member)
			}
			if c.Scope != "self" && !strings.EqualFold(c.Scope, env.unitRef.UnitName) {
				return nil, fmt.Errorf("cannot reference '&%s::%s' inside unit '%s'; use '&%s::%s' or '&self::%s'", c.Scope, c.Member, env.unitRef.UnitName, env.unitRef.UnitName, c.Member, c.Member)
			}
			method, ok := env.unitRef.Methods[c.Member]
			if !ok {
				for mName, mFn := range env.unitRef.Methods {
					if strings.EqualFold(mName, c.Member) {
						method = mFn
						ok = true
						break
					}
				}
			}
			if !ok {
				return nil, fmt.Errorf("method '%s' not found on unit '%s'", c.Member, env.unitRef.UnitName)
			}
			var args []Value
			for _, arg := range call.Arguments {
				a, err := ev.Eval(arg, env)
				if err != nil {
					return nil, err
				}
				args = append(args, a)
			}
			if env.unitRef.UnitName == "Fs" && env != nil && env.CurrentDir() != "" {
				resolveFsPathArgs(c.Member, args, env.CurrentDir())
			}
			return method(args)
		}
		scopeVal, ok := env.Get(c.Scope)
		if !ok {
			return nil, fmt.Errorf("module or scope '%s' not found", c.Scope)
		}
		unit, ok := scopeVal.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an executable module", c.Scope)
		}
		if env != nil && env.CurrentDir() != "" {
			unit.WorkingDir = env.CurrentDir()
		}
		method, ok := unit.Methods[c.Member]
		if !ok {
			for mName, mFn := range unit.Methods {
				if strings.EqualFold(mName, c.Member) {
					method = mFn
					ok = true
					break
				}
			}
		}
		if !ok {
			return nil, fmt.Errorf("method '%s' not found on module '%s'", c.Member, c.Scope)
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
			resolveFsPathArgs(c.Member, args, env.CurrentDir())
		}
		return method(args)
	case *ast.PropAccess:
		unitName := "Unit"
		if env.unitRef != nil {
			unitName = env.unitRef.UnitName
		}
		return nil, fmt.Errorf("cannot call '&%s()' directly. The '&' symbol is strictly reserved for accessing variables declared via 'declare'. To call unit methods, use '&%s::%s()' or '&self::%s()'", c.Name, unitName, c.Name, c.Name)
	case *ast.Identifier:
		val, ok := env.GetLocal(c.Value)
		if !ok {
			if env.unitRef != nil {
				if _, isMethod := env.unitRef.Methods[c.Value]; isMethod {
					return nil, fmt.Errorf("cannot call unit method '%s' directly: use '&%s::%s()' or '&self::%s()'", c.Value, env.unitRef.UnitName, c.Value, c.Value)
				}
			}
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
		if lambdaVal, ok := val.(LambdaValue); ok {
			return ev.CallLambda(lambdaVal, args)
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
	if lambdaVal, ok := val.(LambdaValue); ok {
		return ev.CallLambda(lambdaVal, args)
	}
	return nil, fmt.Errorf("global '%s' is not callable", call.Function)
}

func (ev *Evaluator) evalFunc(fn *ast.FuncStatement, env *Environment) (Value, error) {
	if fn.ReturnType != "" && !ev.isValidType(fn.ReturnType) {
		return nil, fmt.Errorf("function '%s': return type '%s' is not defined", fn.Name, fn.ReturnType)
	}
	for _, p := range fn.Parameters {
		if p.Type != "" && !ev.isValidType(p.Type) {
			return nil, fmt.Errorf("function '%s': type '%s' is not defined for parameter '%s'", fn.Name, p.Type, p.Name)
		}
	}
	userFn := UserFunction{Stmt: fn, Env: env}
	if fn.IsGlobal {
		if env.IsBlockScope() {
			return nil, fmt.Errorf("cannot declare global/export function '%s' inside a block or function: global declarations are only permitted at module root level", fn.Name)
		}
		env.SetGlobal(fn.Name, userFn)
	} else {
		if err := env.Set(fn.Name, userFn, false); err != nil {
			return nil, err
		}
	}
	return userFn, nil
}

func (ev *Evaluator) callUserFunction(fn UserFunction, args []Value, callerEnv *Environment) (Value, error) {
	if fn.Stmt != nil && fn.Stmt.NativeHook != "" {
		if ev.nativeHooks != nil {
			if hook, ok := ev.nativeHooks[fn.Stmt.NativeHook]; ok {
				return hook(args)
			}
		}
		return nil, fmt.Errorf("native hook '%s' is not registered", fn.Stmt.NativeHook)
	}

	minArgs := 0
	hasVariadic := false
	for _, p := range fn.Stmt.Parameters {
		if p.IsVariadic {
			hasVariadic = true
		} else if !p.Optional && !strings.HasSuffix(p.Type, "?") {
			minArgs++
		}
	}

	actualCount := len(args)
	maxArgs := len(fn.Stmt.Parameters)
	if hasVariadic {
		if actualCount < minArgs {
			return NilValue{}, fmt.Errorf("function '%s' expects at least %d arguments, but got %d", fn.Stmt.Name, minArgs, actualCount)
		}
	} else if minArgs == maxArgs {
		if actualCount != minArgs {
			return NilValue{}, fmt.Errorf("function '%s' expects %d arguments, but got %d", fn.Stmt.Name, minArgs, actualCount)
		}
	} else {
		if actualCount < minArgs || actualCount > maxArgs {
			return NilValue{}, fmt.Errorf("function '%s' expects between %d and %d arguments, but got %d", fn.Stmt.Name, minArgs, maxArgs, actualCount)
		}
	}

	fnEnv := NewBlockEnvironment(fn.Env)
	if fn.Env != nil && fn.Env.unitRef != nil {
		fnEnv.unitRef = fn.Env.unitRef
	} else if callerEnv != nil && callerEnv.unitRef != nil {
		fnEnv.unitRef = callerEnv.unitRef
	}
	if callerEnv != nil {
		fnEnv.SetCurrentDir(callerEnv.CurrentDir())
		if dropVal, ok := callerEnv.Get("Drop"); ok {
			_ = fnEnv.Set("Drop", dropVal, false)
		}
		if dropVal, ok := callerEnv.Get("drop"); ok {
			_ = fnEnv.Set("drop", dropVal, false)
		}
		if keepVal, ok := callerEnv.Get("Keep"); ok {
			_ = fnEnv.Set("Keep", keepVal, false)
		}
		if keepVal, ok := callerEnv.Get("keep"); ok {
			_ = fnEnv.Set("keep", keepVal, false)
		}
		if uvVal, ok := callerEnv.Get("UseVariant"); ok {
			_ = fnEnv.Set("UseVariant", uvVal, false)
		}
		if locVal, ok := callerEnv.Get("Local"); ok {
			_ = fnEnv.Set("Local", locVal, false)
		}
	}
	for i, param := range fn.Stmt.Parameters {
		if param.IsVariadic {
			var varList []Value
			if i < len(args) {
				varList = append(varList, args[i:]...)
			}
			_ = fnEnv.Set(param.Name, ListValue{Elements: varList}, false)
			break
		}
		var argVal Value = NilValue{}
		if i < len(args) {
			argVal = args[i]
		}
		isOpt := param.Optional || strings.HasSuffix(param.Type, "?")
		if i >= len(args) || argVal == (NilValue{}) {
			if !isOpt {
				return NilValue{}, fmt.Errorf("missing required argument %d ('%s') of function '%s'", i+1, param.Name, fn.Stmt.Name)
			}
		} else if param.Type != "" && param.Type != "any" {
			cleanType := strings.TrimSuffix(param.Type, "?")
			if !ev.valMatchesType(argVal, cleanType) {
				return NilValue{}, fmt.Errorf("argument %d ('%s') of function '%s' expects type '%s', got '%s'", i+1, param.Name, fn.Stmt.Name, param.Type, argVal.Type())
			}
		}
		if mp, ok := argVal.(MapValue); ok {
			cleanType := strings.TrimSuffix(param.Type, "?")
			if mp.InterfaceName == "" {
				mp.InterfaceName = cleanType
			}
			argVal = mp
		}
		_ = fnEnv.Set(param.Name, argVal, false)
		if param.Type != "" {
			fnEnv.SetVariableType(param.Name, param.Type)
		}
	}

	var retVal Value = NilValue{}
	hasReturn := false
	for _, stmt := range fn.Stmt.Body {
		res, err := ev.Eval(stmt, fnEnv)
		if err != nil {
			return NilValue{}, err
		}
		if ret, ok := res.(ReturnValue); ok {
			retVal = ret.Val
			hasReturn = true
			break
		}
	}

	expectedRet := fn.Stmt.ReturnType
	if expectedRet != "" && expectedRet != "void" {
		expectedRet = ev.resolveType(expectedRet)
		if !hasReturn {
			return NilValue{}, fmt.Errorf("function '%s' declared return type '%s' but did not return a value", fn.Stmt.Name, expectedRet)
		}
		if !ev.valMatchesType(retVal, expectedRet) {
			return NilValue{}, fmt.Errorf("function '%s' returned value of type '%s', expected '%s'", fn.Stmt.Name, retVal.Type(), expectedRet)
		}
		return retVal, nil
	} else if expectedRet == "void" {
		if hasReturn && retVal != (NilValue{}) {
			return NilValue{}, fmt.Errorf("function '%s' has return type 'void' but returned a value", fn.Stmt.Name)
		}
		return NilValue{}, nil
	}

	return retVal, nil
}

func (ev *Evaluator) CallLambda(l LambdaValue, args []Value) (Value, error) {
	if l.Body == nil {
		return NilValue{}, nil
	}
	lambdaEnv := NewBlockEnvironment(l.Env)
	for i, param := range l.Params {
		if i < len(args) {
			_ = lambdaEnv.Set(param.Name, args[i], false)
		} else {
			_ = lambdaEnv.Set(param.Name, NilValue{}, false)
		}
	}
	return ev.Eval(l.Body, lambdaEnv)
}

func (ev *Evaluator) evalStringMethod(c *ast.MemberExpression, strVal StringValue, rawArgs []ast.Expression, env *Environment) (Value, error) {
	var args []Value
	for _, arg := range rawArgs {
		a, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	s := strVal.Val
	switch strings.ToLower(c.Property) {
	case "len", "length":
		if len(args) != 0 {
			return nil, fmt.Errorf("method '%s' expects 0 arguments, but got %d", c.Property, len(args))
		}
		return NumberValue{Val: float64(len(s))}, nil
	case "trim":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'trim' expects 0 arguments, but got %d", len(args))
		}
		return StringValue{Val: strings.TrimSpace(s)}, nil
	case "trimleft":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'trimLeft' expects 0 arguments, but got %d", len(args))
		}
		return StringValue{Val: strings.TrimLeft(s, " \t\r\n")}, nil
	case "trimright":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'trimRight' expects 0 arguments, but got %d", len(args))
		}
		return StringValue{Val: strings.TrimRight(s, " \t\r\n")}, nil
	case "toupper", "uppercase":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'toUpper' expects 0 arguments, but got %d", len(args))
		}
		return StringValue{Val: strings.ToUpper(s)}, nil
	case "tolower", "lowercase":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'toLower' expects 0 arguments, but got %d", len(args))
		}
		return StringValue{Val: strings.ToLower(s)}, nil
	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'contains' requires 1 argument (sub: string), but got %d", len(args))
		}
		return BoolValue{Val: strings.Contains(s, args[0].String())}, nil
	case "startswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'startsWith' requires 1 argument (prefix: string), but got %d", len(args))
		}
		return BoolValue{Val: strings.HasPrefix(s, args[0].String())}, nil
	case "endswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'endsWith' requires 1 argument (suffix: string), but got %d", len(args))
		}
		return BoolValue{Val: strings.HasSuffix(s, args[0].String())}, nil
	case "split":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'split' requires 1 argument (separator: string), but got %d", len(args))
		}
		sep := args[0].String()
		var parts []string
		if sep == "" {
			parts = strings.Split(s, "")
		} else {
			parts = strings.Split(s, sep)
		}
		var elems []Value
		for _, p := range parts {
			elems = append(elems, StringValue{Val: p})
		}
		return ListValue{Elements: elems}, nil
	case "replace":
		if len(args) != 2 {
			return nil, fmt.Errorf("method 'replace' requires 2 arguments (old: string, new: string), but got %d", len(args))
		}
		oldStr := args[0].String()
		newStr := args[1].String()
		return StringValue{Val: strings.ReplaceAll(s, oldStr, newStr)}, nil
	case "slice":
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("method 'slice' requires 1 to 2 arguments (start: number, end?: number), but got %d", len(args))
		}
		n1, ok := args[0].(NumberValue)
		if !ok {
			return nil, fmt.Errorf("method 'slice' expects number for start index, got %s", args[0].Type())
		}
		start := int(n1.Val)
		end := len(s)
		if len(args) == 2 {
			n2, ok := args[1].(NumberValue)
			if !ok {
				return nil, fmt.Errorf("method 'slice' expects number for end index, got %s", args[1].Type())
			}
			end = int(n2.Val)
		}
		if start < 0 {
			start = 0
		}
		if end > len(s) {
			end = len(s)
		}
		if start > end {
			start = end
		}
		return StringValue{Val: s[start:end]}, nil
	case "repeat":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'repeat' requires 1 argument (count: number), but got %d", len(args))
		}
		n, ok := args[0].(NumberValue)
		if !ok {
			return nil, fmt.Errorf("method 'repeat' expects number for count, got %s", args[0].Type())
		}
		count := int(n.Val)
		if count < 0 {
			return nil, fmt.Errorf("method 'repeat' count must be non-negative, got %d", count)
		}
		return StringValue{Val: strings.Repeat(s, count)}, nil
	case "tostring", "to_string":
		return strVal, nil
	default:
		return nil, fmt.Errorf("method '%s' not found on string", c.Property)
	}
}

func (ev *Evaluator) evalListMethod(c *ast.MemberExpression, listVal ListValue, rawArgs []ast.Expression, env *Environment) (Value, error) {
	var args []Value
	for _, arg := range rawArgs {
		a, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	elems := listVal.Elements
	switch strings.ToLower(c.Property) {
	case "len", "length":
		if len(args) != 0 {
			return nil, fmt.Errorf("method '%s' expects 0 arguments, but got %d", c.Property, len(args))
		}
		return NumberValue{Val: float64(len(elems))}, nil
	case "join":
		if len(args) > 1 {
			return nil, fmt.Errorf("method 'join' expects at most 1 argument (separator?: string), but got %d", len(args))
		}
		sep := ""
		if len(args) == 1 {
			sep = args[0].String()
		}
		var parts []string
		for _, el := range elems {
			parts = append(parts, el.String())
		}
		return StringValue{Val: strings.Join(parts, sep)}, nil
	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'contains' requires 1 argument (item: any), but got %d", len(args))
		}
		targetStr := args[0].String()
		found := false
		for _, el := range elems {
			if el.String() == targetStr {
				found = true
				break
			}
		}
		return BoolValue{Val: found}, nil
	case "get", "at":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' requires 1 argument (index: number), but got %d", c.Property, len(args))
		}
		n, ok := args[0].(NumberValue)
		if !ok {
			return nil, fmt.Errorf("method '%s' expects number for index, got %s", c.Property, args[0].Type())
		}
		idx := int(n.Val)
		if idx < 0 {
			idx = len(elems) + idx
		}
		if idx < 0 || idx >= len(elems) {
			return NilValue{}, nil
		}
		return elems[idx], nil
	case "push":
		if len(args) != 1 {
			return nil, fmt.Errorf("method 'push' requires 1 argument (item: any), but got %d", len(args))
		}
		newList := ListValue{Elements: append(elems, args[0])}
		if ident, ok := c.Object.(*ast.Identifier); ok {
			_ = env.Set(ident.Value, newList, false)
		}
		return newList, nil
	case "pop":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'pop' expects 0 arguments, but got %d", len(args))
		}
		if len(elems) == 0 {
			return NilValue{}, nil
		}
		last := elems[len(elems)-1]
		newList := ListValue{Elements: elems[:len(elems)-1]}
		if ident, ok := c.Object.(*ast.Identifier); ok {
			_ = env.Set(ident.Value, newList, false)
		}
		return last, nil
	case "reverse":
		if len(args) != 0 {
			return nil, fmt.Errorf("method 'reverse' expects 0 arguments, but got %d", len(args))
		}
		var rev []Value
		for i := len(elems) - 1; i >= 0; i-- {
			rev = append(rev, elems[i])
		}
		return ListValue{Elements: rev}, nil
	case "slice":
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("method 'slice' requires 1 to 2 arguments (start: number, end?: number), but got %d", len(args))
		}
		n1, ok := args[0].(NumberValue)
		if !ok {
			return nil, fmt.Errorf("method 'slice' expects number for start index, got %s", args[0].Type())
		}
		start := int(n1.Val)
		end := len(elems)
		if len(args) == 2 {
			n2, ok := args[1].(NumberValue)
			if !ok {
				return nil, fmt.Errorf("method 'slice' expects number for end index, got %s", args[1].Type())
			}
			end = int(n2.Val)
		}
		if start < 0 {
			start = 0
		}
		if end > len(elems) {
			end = len(elems)
		}
		if start > end {
			start = end
		}
		return ListValue{Elements: elems[start:end]}, nil
	default:
		return nil, fmt.Errorf("method '%s' not found on list", c.Property)
	}
}

func resolveFsPathArgs(methodName string, args []Value, currentDir string) {
	if currentDir == "" || len(args) == 0 {
		return
	}
	m := strings.ToLower(methodName)
	switch m {
	case "write", "append":
		// Only args[0] is the destination file path. args[1] is the content!
		if !filepath.IsAbs(args[0].String()) {
			args[0] = StringValue{Val: filepath.Join(currentDir, args[0].String())}
		}
	case "move", "copy":
		// Both args[0] (src) and args[1] (dst) are file paths.
		for i := 0; i < len(args) && i < 2; i++ {
			if !filepath.IsAbs(args[i].String()) {
				args[i] = StringValue{Val: filepath.Join(currentDir, args[i].String())}
			}
		}
	default:
		// read, exists, isdir, isfile, size, remove, etc. - only args[0] is path.
		if !filepath.IsAbs(args[0].String()) {
			args[0] = StringValue{Val: filepath.Join(currentDir, args[0].String())}
		}
	}
}

func (ev *Evaluator) evalNumberMethod(c *ast.MemberExpression, numVal NumberValue, rawArgs []ast.Expression, env *Environment) (Value, error) {
	var args []Value
	for _, arg := range rawArgs {
		a, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	switch strings.ToLower(c.Property) {
	case "abs":
		return NumberValue{Val: math.Abs(numVal.Val)}, nil
	case "floor":
		return NumberValue{Val: math.Floor(numVal.Val)}, nil
	case "ceil":
		return NumberValue{Val: math.Ceil(numVal.Val)}, nil
	case "round":
		return NumberValue{Val: math.Round(numVal.Val)}, nil
	case "int":
		return NumberValue{Val: float64(int64(numVal.Val))}, nil
	case "tostring", "to_string":
		return StringValue{Val: fmt.Sprintf("%v", numVal.Val)}, nil
	case "fixed":
		digits := 2
		if len(args) > 0 {
			if d, ok := args[0].(NumberValue); ok {
				digits = int(d.Val)
			}
		}
		return StringValue{Val: fmt.Sprintf("%.*f", digits, numVal.Val)}, nil
	default:
		return nil, fmt.Errorf("method '%s' not found on number", c.Property)
	}
}

func (ev *Evaluator) evalMapMethod(c *ast.MemberExpression, mapVal MapValue, rawArgs []ast.Expression, env *Environment) (Value, error) {
	var args []Value
	for _, arg := range rawArgs {
		a, err := ev.Eval(arg, env)
		if err != nil {
			return nil, err
		}
		args = append(args, a)
	}
	switch strings.ToLower(c.Property) {
	case "get":
		if len(args) < 1 {
			return nil, fmt.Errorf("method 'get' requires at least 1 argument (key: string)")
		}
		key := args[0].String()
		if val, exists := mapVal.Pairs[key]; exists {
			return val, nil
		}
		if len(args) >= 2 {
			return args[1], nil
		}
		return NilValue{}, nil
	case "set":
		if len(args) < 2 {
			return nil, fmt.Errorf("method 'set' requires 2 arguments (key: string, value: any)")
		}
		key := args[0].String()
		mapVal.Pairs[key] = args[1]
		return mapVal, nil
	case "merge", "update":
		if len(args) < 1 {
			return nil, fmt.Errorf("method 'merge' requires 1 argument (other: map)")
		}
		if otherMap, ok := args[0].(MapValue); ok {
			for k, v := range otherMap.Pairs {
				mapVal.Pairs[k] = v
			}
			return mapVal, nil
		}
		return nil, fmt.Errorf("argument to 'merge' must be a map, got %s", args[0].Type())
	case "keys":
		var keys []Value
		for k := range mapVal.Pairs {
			keys = append(keys, StringValue{Val: k})
		}
		return ListValue{Elements: keys}, nil
	case "values":
		var vals []Value
		for _, v := range mapVal.Pairs {
			vals = append(vals, v)
		}
		return ListValue{Elements: vals}, nil
	case "entries":
		var entries []Value
		for k, v := range mapVal.Pairs {
			entries = append(entries, MapValue{
				Pairs: map[string]Value{
					"key":   StringValue{Val: k},
					"value": v,
				},
			})
		}
		return ListValue{Elements: entries}, nil
	case "has":
		if len(args) < 1 {
			return nil, fmt.Errorf("method 'has' requires 1 argument (key: string)")
		}
		key := args[0].String()
		_, exists := mapVal.Pairs[key]
		return BoolValue{Val: exists}, nil
	case "len", "size":
		return NumberValue{Val: float64(len(mapVal.Pairs))}, nil
	case "delete":
		if len(args) < 1 {
			return nil, fmt.Errorf("method 'delete' requires 1 argument (key: string)")
		}
		key := args[0].String()
		delete(mapVal.Pairs, key)
		return NilValue{}, nil
	case "clear":
		for k := range mapVal.Pairs {
			delete(mapVal.Pairs, k)
		}
		return NilValue{}, nil
	default:
		return nil, fmt.Errorf("method '%s' not found on map", c.Property)
	}
}

