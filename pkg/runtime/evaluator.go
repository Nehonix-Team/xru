package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

type TreeDispatcherFunc func(dt *ast.DispatchTreeStatement, eval *Evaluator, env *Environment) error

type Evaluator struct {
	builtins       map[string]*UnitInstance
	treeDispatcher TreeDispatcherFunc
	loadedModules  map[string]*UnitInstance
	loadingModules map[string]bool
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		builtins:       NewBuiltinUnits(),
		loadedModules:  make(map[string]*UnitInstance),
		loadingModules: make(map[string]bool),
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
		if n.IsGlobal {
			if n.IsConst {
				env.SetGlobalConst(n.Name)
			}
			env.SetGlobal(n.Name, val)
			return val, nil
		}
		if _, exists := env.GetLocal(n.Name); exists {
			return nil, fmt.Errorf("variable '%s' is already declared in this scope", n.Name)
		}
		if val, exists := env.Get(n.Name); exists {
			if _, isUnit := val.(*UnitInstance); isUnit {
				return nil, fmt.Errorf("variable '%s' collides with imported module '%s'", n.Name, n.Name)
			}
		}
		return val, env.Set(n.Name, val, n.IsConst)
	case *ast.ReassignStatement:
		val, err := ev.Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		if err := env.Reassign(n.Name, val); err != nil {
			return nil, err
		}
		return val, nil
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
				if _, exists := env.Get(alias); exists {
					return nil, fmt.Errorf("module alias '%s' collides with an existing variable or import", alias)
				}
				if err := env.Set(alias, bUnit, true); err != nil {
					return nil, err
				}
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
			MethodDefs: n.Methods,
		}
		for _, prop := range n.Properties {
			pVal, _ := ev.Eval(prop.Value, env)
			unitInst.Properties[prop.Name] = pVal
		}
		for _, method := range n.Methods {
			capturedMethod := method
			unitInst.Methods[method.Name] = func(args []Value) (Value, error) {
				methodEnv := NewEnvironment(env)
				methodEnv.unitRef = unitInst
				return ev.callUserFunction(UserFunction{Stmt: capturedMethod, Env: methodEnv}, args, methodEnv)
			}
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
			Methods:    make(map[string]NativeMethod),
			Directives: proto.Directives,
			MethodDefs: proto.MethodDefs,
		}
		for k, v := range proto.Properties {
			clone.Properties[k] = v
		}
		for _, method := range proto.MethodDefs {
			capturedMethod := method
			clone.Methods[method.Name] = func(args []Value) (Value, error) {
				methodEnv := NewEnvironment(env)
				methodEnv.unitRef = clone
				return ev.callUserFunction(UserFunction{Stmt: capturedMethod, Env: methodEnv}, args, methodEnv)
			}
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
	case *ast.WithExpression:
		return ev.evalWithExpression(n, env)
	case *ast.IfStatement:
		return ev.evalIf(n, env)
	case *ast.IfExpression:
		condVal, err := ev.Eval(n.Condition, env)
		if err != nil {
			return nil, err
		}
		if bVal, ok := condVal.(BoolValue); ok && bVal.Val {
			return ev.Eval(n.Consequence, env)
		}
		return ev.Eval(n.Alternative, env)
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
			return nil, fmt.Errorf("undefined property '&%s' on unit '%s'", n.Name, env.unitRef.UnitName)
		}
		return nil, fmt.Errorf("property '&%s' cannot be accessed outside a unit definition", n.Name)
	case *ast.StringLiteral:
		// Interpolation {var} ou {module::prop}
		res := n.Value
		for k, v := range env.All() {
			placeholder := fmt.Sprintf("{%s}", k)
			res = strings.ReplaceAll(res, placeholder, v.String())
			if unit, ok := v.(*UnitInstance); ok {
				for pName, pVal := range unit.Properties {
					scopedPlaceholder := fmt.Sprintf("{%s::%s}", k, pName)
					res = strings.ReplaceAll(res, scopedPlaceholder, pVal.String())
				}
			}
		}
		if env.unitRef != nil {
			for pName, pVal := range env.unitRef.Properties {
				placeholder := fmt.Sprintf("{&%s}", pName)
				res = strings.ReplaceAll(res, placeholder, pVal.String())
			}
			if matches := reUnitPropInterpolation.FindAllStringSubmatch(res, -1); len(matches) > 0 {
				return NilValue{}, fmt.Errorf("undefined property '&%s' on unit '%s'", matches[0][1], env.unitRef.UnitName)
			}
		} else {
			if matches := reUnitPropInterpolation.FindAllStringSubmatch(res, -1); len(matches) > 0 {
				return NilValue{}, fmt.Errorf("property '&%s' cannot be accessed outside a unit definition", matches[0][1])
			}
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
	case *ast.DurationLiteral:
		dur, err := time.ParseDuration(n.Value)
		if err != nil {
			return nil, err
		}
		return DurationValue{Val: dur}, nil
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
	case *ast.PrefixExpression:
		return ev.evalPrefix(n, env)
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
	case *ast.MapStatement:
		return NilValue{}, nil
	case *ast.ModStatement:
		return NilValue{}, nil
	case *ast.FromUseStatement:
		modFilePath := modules.FindModConfigFile(env.CurrentDir())
		if modFilePath == "" {
			return nil, fmt.Errorf("no '_mod.xru' file found in workspace")
		}
		modCfg, errs := modules.LoadModConfig(modFilePath)
		if len(errs) > 0 {
			return nil, fmt.Errorf("error in '_mod.xru': %s", errs[0])
		}

		loadModuleInstance := func(modName string) (*UnitInstance, error) {
			modDetails, err := modCfg.ResolveModule(n.Source, modName)
			if err != nil {
				return nil, err
			}
			absPath, _ := filepath.Abs(modDetails.FilePath)
			if ev.loadingModules[absPath] {
				return nil, fmt.Errorf("circular module dependency detected: '%s' (from '%s') is already being loaded", modName, n.Source)
			}
			if cached, ok := ev.loadedModules[absPath]; ok {
				return cached, nil
			}

			ev.loadingModules[absPath] = true
			defer func() {
				delete(ev.loadingModules, absPath)
			}()

			modEnv := NewEnvironment(nil)
			modEnv.SetCurrentDir(filepath.Dir(modDetails.FilePath))
			content, err := os.ReadFile(modDetails.FilePath)
			if err != nil {
				return nil, err
			}
			l := lexer.New(modDetails.FilePath, string(content))
			p := parser.New(l)
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				return nil, fmt.Errorf("syntax error in %s:\n  • %s", modDetails.FilePath, strings.Join(p.Errors(), "\n  • "))
			}

			// Validation sémantique stricte du module (ex: use std requis si std est utilisé)
			c := checker.New()
			c.SetCurrentFile(modDetails.FilePath)
			c.Check(prog)
			if len(c.Errors()) > 0 {
				return nil, fmt.Errorf("validation error in %s:\n  • %s", modDetails.FilePath, strings.Join(c.Errors(), "\n  • "))
			}

			_, err = ev.Eval(prog, modEnv)
			if err != nil {
				return nil, err
			}

			unitInst := &UnitInstance{
				UnitName:   modName,
				Properties: make(map[string]Value),
				Methods:    make(map[string]NativeMethod),
				Directives: make(map[string]NativeDirective),
			}
			for expName, exp := range modDetails.Exports {
				if exp.Kind == "func" {
					if fnVal, ok := modEnv.Get(expName); ok {
						if uFn, ok := fnVal.(UserFunction); ok {
							capturedFn := uFn
							unitInst.Methods[expName] = func(args []Value) (Value, error) {
								return ev.callUserFunction(capturedFn, args, env)
							}
						}
					}
				} else {
					if val, ok := modEnv.Get(expName); ok {
						unitInst.Properties[expName] = val
					}
				}
			}
			ev.loadedModules[absPath] = unitInst
			return unitInst, nil
		}

		// Cas 1: multi-modules (auth, cdn)
		if len(n.Modules) > 0 {
			for _, mName := range n.Modules {
				unit, err := loadModuleInstance(mName)
				if err != nil {
					return nil, err
				}
				if err := env.Set(mName, unit, true); err != nil {
					return nil, err
				}
			}
			return NilValue{}, nil
		}

		// Cas 2: membre individuel (auth::b)
		if n.Member != "" {
			unit, err := loadModuleInstance(n.Module)
			if err != nil {
				return nil, err
			}
			if prop, ok := unit.Properties[n.Member]; ok {
				if err := env.Set(n.Member, prop, false); err != nil {
					return nil, err
				}
				return prop, nil
			}
			if method, ok := unit.Methods[n.Member]; ok {
				fnVal := FunctionValue{Name: n.Member, Fn: method}
				if err := env.Set(n.Member, fnVal, false); err != nil {
					return nil, err
				}
				return fnVal, nil
			}
			return nil, fmt.Errorf("member '%s' not found on module '%s'", n.Member, n.Module)
		}

		// Cas 3: module complet (auth as a ou auth)
		alias := n.Alias
		if alias == "" {
			alias = n.Module
		}
		unit, err := loadModuleInstance(n.Module)
		if err != nil {
			return nil, err
		}
		if err := env.Set(alias, unit, true); err != nil {
			return nil, err
		}
		return unit, nil
	case *ast.ScopeAccessExpression:
		scopeVal, ok := env.Get(n.Scope)
		if !ok {
			return nil, fmt.Errorf("module or scope '%s' not found", n.Scope)
		}
		if unit, ok := scopeVal.(*UnitInstance); ok {
			if prop, exists := unit.Properties[n.Member]; exists {
				return prop, nil
			}
			if method, exists := unit.Methods[n.Member]; exists {
				return FunctionValue{Name: n.Member, Fn: method}, nil
			}
			return nil, fmt.Errorf("member '%s' not found on module '%s'", n.Member, n.Scope)
		}
		return nil, fmt.Errorf("'%s' is not a module", n.Scope)
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

var (
	reInterpolationIdentifier = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	reUnitPropInterpolation   = regexp.MustCompile(`\{&([a-zA-Z_][a-zA-Z0-9_]*)\}`)
)

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
