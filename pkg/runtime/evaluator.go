package runtime

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

type Environment struct {
	parent    *Environment
	store     map[string]Value
	constants map[string]bool
	unitRef   *UnitInstance // Pointeur vers l'unité courante si dans #UNIT
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		parent:    parent,
		store:     make(map[string]Value),
		constants: make(map[string]bool),
	}
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
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

type Evaluator struct {
	builtins map[string]*UnitInstance
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		builtins: NewBuiltinUnits(),
	}
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
	case *ast.CallExpression:
		return ev.evalCall(n, env)
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
		for k, v := range env.store {
			placeholder := fmt.Sprintf("{%s}", k)
			res = strings.ReplaceAll(res, placeholder, v.String())
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
		return method(args)
	}
	return NilValue{}, nil
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
