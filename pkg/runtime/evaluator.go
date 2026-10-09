package runtime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

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
		if n.Type != "" && !valMatchesType(val, n.Type) {
			return nil, fmt.Errorf("cannot assign value of type '%s' to variable '%s' declared as '%s'", val.Type(), n.Name, n.Type)
		}
		return val, env.Set(n.Name, val, n.IsConst)
	case *ast.ArgStatement:
		// S'il existe déjà dans l'environnement (fourni par les CLI args), on le valide
		if v, exists := env.Get(n.Name); exists {
			if n.Type != "" && !valMatchesType(v, n.Type) {
				return nil, fmt.Errorf("argument '%s' of type '%s' received invalid value of type '%s'", n.Name, n.Type, v.Type())
			}
			return v, nil
		}
		if n.DefaultValue != nil {
			val, err := ev.Eval(n.DefaultValue, env)
			if err != nil {
				return nil, err
			}
			if n.Type != "" && !valMatchesType(val, n.Type) {
				return nil, fmt.Errorf("argument '%s' of type '%s' has default value of type '%s'", n.Name, n.Type, val.Type())
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
		found := false
		for bName, bUnit := range ev.builtins {
			if strings.EqualFold(bName, unitName) {
				env.Set(alias, bUnit, true)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("module '%s' does not exist", unitName)
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
	case *ast.BlockStatement:
		return ev.evalBlockStatement(n, env)
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
		if strings.EqualFold(n.Value, "nil") || strings.EqualFold(n.Value, "null") {
			return NilValue{}, nil
		}
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
	case *ast.LambdaExpression:
		return LambdaValue{
			Params: n.Parameters,
			Body:   n.Body,
			Env:    env,
		}, nil
	case *ast.CustomDirectiveInvocation:
		return ev.evalCustomDirective(n, env)
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

func valMatchesType(val Value, expected string) bool {
	if expected == "" || expected == "any" {
		return true
	}
	if val == nil {
		return false
	}
	switch strings.ToLower(expected) {
	case "string":
		return val.Type() == ValString
	case "number", "int", "float":
		return val.Type() == ValNumber
	case "bool", "boolean":
		return val.Type() == ValBool
	case "duration":
		return val.Type() == ValDuration
	case "list":
		return val.Type() == ValList
	case "map":
		return val.Type() == ValMap
	case "function":
		return val.Type() == "function"
	default:
		if strings.HasPrefix(expected, "list") && val.Type() == ValList {
			return true
		}
		if strings.HasPrefix(expected, "map") && val.Type() == ValMap {
			return true
		}
		if val.Type() == ValUnit {
			if u, ok := val.(*UnitInstance); ok {
				return strings.EqualFold(u.UnitName, expected)
			}
		}
		return true
	}
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
