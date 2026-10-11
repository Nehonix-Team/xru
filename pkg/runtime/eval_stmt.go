package runtime

import (
	"context"
	"fmt"
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
	bVal, ok := condVal.(BoolValue)
	if ok && bVal.Val {
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
		ebVal, ok := eVal.(BoolValue)
		if ok && ebVal.Val {
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
		if err == nil && res != nil {
			if unitInst, ok := res.(*UnitInstance); ok && unitInst.Methods != nil {
				if completeFn, ok := unitInst.Methods["__complete"]; ok {
					res, err = completeFn(nil)
				}
			}
		}
		task.mu.Lock()
		defer task.mu.Unlock()
		task.result = res
		task.err = err
		task.isDone = true
	}()

	return task, nil
}

func (ev *Evaluator) evalWith(with *ast.WithStatement, env *Environment) (Value, error) {
	if with.Target == nil && len(with.Body) > 0 {
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

	targetVal, err := ev.Eval(with.Target, env)
	if err != nil {
		return nil, err
	}

	withEnv := NewEnvironment(env)
	if unitInst, ok := targetVal.(*UnitInstance); ok {
		withEnv.unitRef = unitInst
	}

	for _, stmt := range with.Body {
		res, err := ev.Eval(stmt, withEnv)
		if err != nil {
			return nil, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret, nil
		}
	}

	if withEnv.unitRef != nil && withEnv.unitRef.Methods != nil {
		if completeFn, ok := withEnv.unitRef.Methods["__complete"]; ok {
			return completeFn(nil)
		}
	}

	return NilValue{}, nil
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

