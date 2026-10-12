package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalBlockStatement(block *ast.BlockStatement, env *Environment) (Value, error) {
	var result Value = NilValue{}
	for _, stmt := range block.Statements {
		res, err := ev.Eval(stmt, env)
		if err != nil {
			return nil, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret, nil
		}
		result = res
	}
	return result, nil
}

func (ev *Evaluator) evalIf(ifStmt *ast.IfStatement, env *Environment) (Value, error) {
	condVal, err := ev.Eval(ifStmt.Condition, env)
	if err != nil {
		return nil, err
	}
	if isTruthy(condVal) {
		for _, s := range ifStmt.Consequence {
			res, err := ev.Eval(s, env)
			if err != nil {
				return nil, err
			}
			if ret, ok := res.(ReturnValue); ok {
				return ret, nil
			}
		}
		return NilValue{}, nil
	}

	for _, elif := range ifStmt.ElifBlocks {
		eVal, err := ev.Eval(elif.Condition, env)
		if err != nil {
			return nil, err
		}
		if isTruthy(eVal) {
			for _, s := range elif.Body {
				res, err := ev.Eval(s, env)
				if err != nil {
					return nil, err
				}
				if ret, ok := res.(ReturnValue); ok {
					return ret, nil
				}
			}
			return NilValue{}, nil
		}
	}

	for _, s := range ifStmt.Alternative {
		res, err := ev.Eval(s, env)
		if err != nil {
			return nil, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret, nil
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalWithExpression(w *ast.WithExpression, env *Environment) (Value, error) {
	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx
	task := &TaskValue{
		done:   make(chan struct{}),
		cancel: cancel,
	}

	asyncEnv := NewEnvironment(env)

	go func() {
		defer close(task.done)
		res, err := ev.Eval(w.Target, asyncEnv)
		task.mu.Lock()
		defer task.mu.Unlock()
		task.result = res
		task.err = err
		task.isDone = true
	}()

	return task, nil
}

func (ev *Evaluator) evalWith(with *ast.WithStatement, env *Environment) (Value, error) {
	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx
	task := &TaskValue{
		done:   make(chan struct{}),
		cancel: cancel,
	}
	asyncEnv := NewEnvironment(env)
	go func() {
		defer close(task.done)
		var res Value = NilValue{}
		var err error
		if with.Target != nil {
			res, err = ev.Eval(with.Target, asyncEnv)
		}
		for _, stmt := range with.Body {
			r, e := ev.Eval(stmt, asyncEnv)
			if e != nil {
				err = e
				break
			}
			if ret, ok := r.(ReturnValue); ok {
				res = ret.Val
				break
			}
			res = r
		}
		task.mu.Lock()
		defer task.mu.Unlock()
		task.result = res
		task.err = err
		task.isDone = true
	}()
	return task, nil
}

func (ev *Evaluator) evalCustomDirective(dir *ast.CustomDirectiveInvocation, env *Environment) (Value, error) {
	var keyVal Value = NilValue{}
	var err error
	if dir.Key != nil {
		if ident, ok := dir.Key.(*ast.Identifier); ok {
			keyVal = StringValue{Val: ident.Value}
		} else {
			keyVal, err = ev.Eval(dir.Key, env)
			if err != nil {
				return nil, err
			}
		}
	}
	var valVal Value = NilValue{}
	if dir.Value != nil {
		valVal, err = ev.Eval(dir.Value, env)
		if err != nil {
			return nil, err
		}
	}
	if env.unitRef != nil && env.unitRef.Directives != nil {
		dirName := dir.Directive
		for dKey, handler := range env.unitRef.Directives {
			if strings.EqualFold(dKey, dirName) {
				return NilValue{}, handler(keyVal, valVal)
			}
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalDispatchTree(dt *ast.DispatchTreeStatement, env *Environment) (Value, error) {
	if dt.PathExpr != nil {
		pathVal, err := ev.Eval(dt.PathExpr, env)
		if err != nil {
			return NilValue{}, err
		}
		dt.Path = pathVal.String()
	}

	if dt.PatternExpr != nil {
		patVal, err := ev.Eval(dt.PatternExpr, env)
		if err != nil {
			return NilValue{}, err
		}
		dt.Pattern = patVal.String()
	} else if dt.Pattern == "" {
		dt.Pattern = ".rules.xru"
	}

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

func (ev *Evaluator) evalIn(inStmt *ast.InStatement, env *Environment) (Value, error) {
	pathVal, err := ev.Eval(inStmt.Path, env)
	if err != nil {
		return NilValue{}, err
	}
	targetPath := pathVal.String()

	baseDir := env.CurrentDir()
	if baseDir == "" {
		baseDir = "."
	}
	var fullDir string
	if filepath.IsAbs(targetPath) {
		fullDir = targetPath
	} else {
		fullDir = filepath.Clean(filepath.Join(baseDir, targetPath))
	}

	scopedEnv := NewEnvironment(env)
	scopedEnv.SetCurrentDir(fullDir)

	InjectSandboxedPrimitives(fullDir, scopedEnv)

	for k, v := range env.All() {
		_ = scopedEnv.Set(strings.ToLower(k), v, false)
		_ = scopedEnv.Set(strings.ToUpper(k), v, false)
	}

	for _, stmt := range inStmt.Body {
		res, err := ev.Eval(stmt, scopedEnv)
		if err != nil {
			return NilValue{}, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret, nil
		}
	}
	return BoolValue{Val: true}, nil
}

func InjectSandboxedPrimitives(dir string, env *Environment) {
	keepMethod := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		keepList, ok := args[0].(ListValue)
		if !ok {
			return NilValue{}, fmt.Errorf("keep requires a list of file names")
		}

		keepMap := make(map[string]bool)
		for _, el := range keepList.Elements {
			keepMap[el.String()] = true
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return NilValue{}, err
		}

		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".xru") {
				continue
			}
			if !keepMap[name] {
				targetPath := filepath.Join(dir, name)
				if err := validateSandbox(dir, targetPath); err != nil {
					return NilValue{}, err
				}
				_ = os.RemoveAll(targetPath)
			}
		}
		return BoolValue{Val: true}, nil
	}

	dropMethod := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		dropList, ok := args[0].(ListValue)
		if !ok {
			return NilValue{}, fmt.Errorf("drop requires a list of file names/patterns")
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			return NilValue{}, err
		}

		baseDir := filepath.Base(dir)

		for _, item := range dropList.Elements {
			pattern := item.String()
			cleanPattern := filepath.Clean(pattern)

			// Si le pattern cible le dossier courant lui-même (ex: ".", "./", "servers", "./servers/", "./servers")
			if pattern == "." || pattern == "./" || cleanPattern == "." || cleanPattern == baseDir || pattern == "./"+baseDir || pattern == "./"+baseDir+"/" {
				_ = os.RemoveAll(dir)
				continue
			}

			for _, entry := range entries {
				name := entry.Name()
				if strings.HasSuffix(name, ".xru") {
					continue
				}
				matched, _ := filepath.Match(pattern, name)
				if pattern == "*" || matched || pattern == name || cleanPattern == name {
					targetPath := filepath.Join(dir, name)
					if err := validateSandbox(dir, targetPath); err != nil {
						return NilValue{}, err
					}
					_ = os.RemoveAll(targetPath)
				}
			}
		}
		return BoolValue{Val: true}, nil
	}

	useVariantMethod := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("UseVariant requires target and variant source")
		}
		target := args[0].String()
		variant := args[1].String()

		targetPath := filepath.Join(dir, target)
		variantPath := filepath.Join(dir, variant)

		if err := validateSandbox(dir, targetPath); err != nil {
			return NilValue{}, err
		}
		if err := validateSandbox(dir, variantPath); err != nil {
			return NilValue{}, err
		}

		if _, err := os.Stat(variantPath); os.IsNotExist(err) {
			return NilValue{}, fmt.Errorf("[ERR_TARGET_NOT_FOUND] Variant source '%s' does not exist in %s", variant, dir)
		}

		data, err := os.ReadFile(variantPath)
		if err != nil {
			return NilValue{}, err
		}
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return NilValue{}, err
		}
		_ = os.Remove(variantPath)
		return BoolValue{Val: true}, nil
	}

	scopeUnit := &UnitInstance{
		UnitName:   "LocalScope",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Keep":       keepMethod,
			"Drop":       dropMethod,
			"UseVariant": useVariantMethod,
		},
		Directives: make(map[string]NativeDirective),
	}
	_ = env.Set("Local", scopeUnit, true)
	_ = env.Set("Keep", FunctionValue{Name: "Keep", Fn: keepMethod}, true)
	_ = env.Set("keep", FunctionValue{Name: "keep", Fn: keepMethod}, true)
	_ = env.Set("Drop", FunctionValue{Name: "Drop", Fn: dropMethod}, true)
	_ = env.Set("drop", FunctionValue{Name: "drop", Fn: dropMethod}, true)
	_ = env.Set("UseVariant", FunctionValue{Name: "UseVariant", Fn: useVariantMethod}, true)
	_ = env.Set("useVariant", FunctionValue{Name: "useVariant", Fn: useVariantMethod}, true)
}

func validateSandbox(allowedDir, targetPath string) error {
	rel, err := filepath.Rel(allowedDir, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("[ERR_SANDBOX_VIOLATION] Path '%s' escapes local sandbox '%s'", targetPath, allowedDir)
	}
	return nil
}


func (ev *Evaluator) evalRange(r *ast.RangeExpression, env *Environment) (Value, error) {
	startVal, err := ev.Eval(r.Start, env)
	if err != nil {
		return nil, err
	}
	endVal, err := ev.Eval(r.End, env)
	if err != nil {
		return nil, err
	}
	startNum, ok1 := startVal.(NumberValue)
	endNum, ok2 := endVal.(NumberValue)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("range operator '..' requires numbers, got '%s' and '%s'", startVal.Type(), endVal.Type())
	}
	return RangeValue{Start: int64(startNum.Val), End: int64(endNum.Val)}, nil
}

func (ev *Evaluator) evalFor(f *ast.ForStatement, env *Environment) (Value, error) {
	iterVal, err := ev.Eval(f.Iterable, env)
	if err != nil {
		return nil, err
	}

	loopEnv := NewEnvironment(env)

	switch it := iterVal.(type) {
	case RangeValue:
		start := it.Start
		end := it.End
		step := int64(1)
		if start > end {
			step = int64(-1)
		}
		idx := 0
		for current := start; ; current += step {
			_ = loopEnv.Set(f.Item, NumberValue{Val: float64(current)}, false)
			if f.Index != "" {
				_ = loopEnv.Set(f.Index, NumberValue{Val: float64(idx)}, false)
			}
			for _, stmt := range f.Body {
				res, err := ev.Eval(stmt, loopEnv)
				if err != nil {
					return nil, err
				}
				if ret, ok := res.(ReturnValue); ok {
					return ret, nil
				}
			}
			if current == end {
				break
			}
			idx++
		}
	case ListValue:
		for idx, item := range it.Elements {
			_ = loopEnv.Set(f.Item, item, false)
			if f.Index != "" {
				_ = loopEnv.Set(f.Index, NumberValue{Val: float64(idx)}, false)
			}
			for _, stmt := range f.Body {
				res, err := ev.Eval(stmt, loopEnv)
				if err != nil {
					return nil, err
				}
				if ret, ok := res.(ReturnValue); ok {
					return ret, nil
				}
			}
		}
	default:
		return nil, fmt.Errorf("cannot iterate over '%s', expected list or range", iterVal.Type())
	}

	return NilValue{}, nil
}

func (ev *Evaluator) evalMatch(m *ast.MatchStatement, env *Environment) (Value, error) {
	targetVal, err := ev.Eval(m.Target, env)
	if err != nil {
		return NilValue{}, err
	}

	for _, kase := range m.Cases {
		patternVal, err := ev.Eval(kase.Pattern, env)
		if err != nil {
			return NilValue{}, err
		}

		matched := false
		if targetVal != nil && patternVal != nil {
			if targetVal.String() == patternVal.String() {
				matched = true
			}
		}

		if matched {
			caseEnv := NewEnvironment(env)
			for _, stmt := range kase.Body {
				res, err := ev.Eval(stmt, caseEnv)
				if err != nil {
					return NilValue{}, err
				}
				if ret, ok := res.(ReturnValue); ok {
					return ret, nil
				}
			}
			return NilValue{}, nil
		}
	}

	if len(m.DefaultBlock) > 0 {
		defEnv := NewEnvironment(env)
		for _, stmt := range m.DefaultBlock {
			res, err := ev.Eval(stmt, defEnv)
			if err != nil {
				return NilValue{}, err
			}
			if ret, ok := res.(ReturnValue); ok {
				return ret, nil
			}
		}
	}

	return NilValue{}, nil
}

