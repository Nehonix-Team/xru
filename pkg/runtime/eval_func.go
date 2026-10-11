package runtime

import (
	"fmt"
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
		if strVal, ok := objVal.(StringValue); ok {
			return ev.evalStringMethod(c, strVal, call.Arguments, env)
		}
		if listVal, ok := objVal.(ListValue); ok {
			return ev.evalListMethod(c, listVal, call.Arguments, env)
		}
		unit, ok := objVal.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an executable unit", c.Object.String())
		}
		if unit.IsPrototype {
			return nil, fmt.Errorf("cannot call method '%s' directly on unit definition '%s'. You must first instantiate it with '@%s()'", c.Property, unit.UnitName, unit.UnitName)
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
	case *ast.ScopeAccessExpression:
		scopeVal, ok := env.Get(c.Scope)
		if !ok {
			return nil, fmt.Errorf("module or scope '%s' not found", c.Scope)
		}
		unit, ok := scopeVal.(*UnitInstance)
		if !ok {
			return nil, fmt.Errorf("'%s' is not an executable module", c.Scope)
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
	expectedCount := len(fn.Stmt.Parameters)
	actualCount := len(args)
	if actualCount != expectedCount {
		return NilValue{}, fmt.Errorf("function '%s' expects %d arguments, but got %d", fn.Stmt.Name, expectedCount, actualCount)
	}

	fnEnv := NewEnvironment(fn.Env)
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
		argVal := args[i]
		if param.Type != "" && param.Type != "any" {
			if !valMatchesType(argVal, param.Type) {
				return NilValue{}, fmt.Errorf("argument %d ('%s') of function '%s' expects type '%s', got '%s'", i+1, param.Name, fn.Stmt.Name, param.Type, argVal.Type())
			}
		}
		_ = fnEnv.Set(param.Name, argVal, false)
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
		if !hasReturn {
			return NilValue{}, fmt.Errorf("function '%s' declared return type '%s' but did not return a value", fn.Stmt.Name, expectedRet)
		}
		if !valMatchesType(retVal, expectedRet) {
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
	lambdaEnv := NewEnvironment(l.Env)
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
		return NumberValue{Val: float64(len(s))}, nil
	case "trim":
		return StringValue{Val: strings.TrimSpace(s)}, nil
	case "trimleft":
		return StringValue{Val: strings.TrimLeft(s, " \t\r\n")}, nil
	case "trimright":
		return StringValue{Val: strings.TrimRight(s, " \t\r\n")}, nil
	case "toupper", "uppercase":
		return StringValue{Val: strings.ToUpper(s)}, nil
	case "tolower", "lowercase":
		return StringValue{Val: strings.ToLower(s)}, nil
	case "contains":
		sub := ""
		if len(args) > 0 {
			sub = args[0].String()
		}
		return BoolValue{Val: strings.Contains(s, sub)}, nil
	case "startswith":
		pfx := ""
		if len(args) > 0 {
			pfx = args[0].String()
		}
		return BoolValue{Val: strings.HasPrefix(s, pfx)}, nil
	case "endswith":
		sfx := ""
		if len(args) > 0 {
			sfx = args[0].String()
		}
		return BoolValue{Val: strings.HasSuffix(s, sfx)}, nil
	case "split":
		sep := ""
		if len(args) > 0 {
			sep = args[0].String()
		}
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
		oldStr := ""
		newStr := ""
		if len(args) > 0 {
			oldStr = args[0].String()
		}
		if len(args) > 1 {
			newStr = args[1].String()
		}
		return StringValue{Val: strings.ReplaceAll(s, oldStr, newStr)}, nil
	case "slice":
		start := 0
		end := len(s)
		if len(args) > 0 {
			if n, ok := args[0].(NumberValue); ok {
				start = int(n.Val)
			}
		}
		if len(args) > 1 {
			if n, ok := args[1].(NumberValue); ok {
				end = int(n.Val)
			}
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
		return NumberValue{Val: float64(len(elems))}, nil
	case "join":
		sep := ""
		if len(args) > 0 {
			sep = args[0].String()
		}
		var parts []string
		for _, el := range elems {
			parts = append(parts, el.String())
		}
		return StringValue{Val: strings.Join(parts, sep)}, nil
	case "contains":
		if len(args) == 0 {
			return BoolValue{Val: false}, nil
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
	case "push":
		if len(args) > 0 {
			newList := ListValue{Elements: append(elems, args[0])}
			if ident, ok := c.Object.(*ast.Identifier); ok {
				_ = env.Set(ident.Value, newList, false)
			}
			return newList, nil
		}
		return listVal, nil
	case "pop":
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
		var rev []Value
		for i := len(elems) - 1; i >= 0; i-- {
			rev = append(rev, elems[i])
		}
		return ListValue{Elements: rev}, nil
	case "slice":
		start := 0
		end := len(elems)
		if len(args) > 0 {
			if n, ok := args[0].(NumberValue); ok {
				start = int(n.Val)
			}
		}
		if len(args) > 1 {
			if n, ok := args[1].(NumberValue); ok {
				end = int(n.Val)
			}
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
