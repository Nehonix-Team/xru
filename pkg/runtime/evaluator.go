package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	nativeHooks    map[string]NativeMethod
	treeDispatcher TreeDispatcherFunc
	loadedModules  map[string]*UnitInstance
	loadingModules map[string]bool
	interfaces     map[string]*ast.InterfaceStatement
}

func isBuiltinUnit(name string) bool {
	switch strings.ToLower(name) {
	case "std", "net", "fs", "crypto", "sys", "cmd", "path", "local", "json", "math", "date":
		return true
	}
	return false
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		builtins:       NewBuiltinUnits(),
		nativeHooks:    NewNativeHooks(),
		loadedModules:  make(map[string]*UnitInstance),
		loadingModules: make(map[string]bool),
		interfaces:     make(map[string]*ast.InterfaceStatement),
	}
}

func (ev *Evaluator) RegisterNativeHook(name string, fn NativeMethod) {
	if ev.nativeHooks == nil {
		ev.nativeHooks = make(map[string]NativeMethod)
	}
	ev.nativeHooks[name] = fn
}

func (ev *Evaluator) SetTreeDispatcher(fn TreeDispatcherFunc) {
	ev.treeDispatcher = fn
}

func (ev *Evaluator) Eval(node ast.Node, env *Environment) (Value, error) {
	switch n := node.(type) {
	case *ast.Program:
		return ev.evalProgram(n, env)
	case *ast.VarStatement:
		if n.Type != "" && !ev.isValidType(n.Type) {
			return nil, fmt.Errorf("variable '%s': type '%s' is not defined", n.Name, n.Type)
		}
		val, err := ev.Eval(n.Value, env)
		if err != nil {
			return nil, err
		}
		if n.Type != "" && !ev.valMatchesType(val, n.Type) {
			return nil, fmt.Errorf("cannot assign value of type '%s' to variable '%s' declared as '%s'", val.Type(), n.Name, n.Type)
		}
		if mp, ok := val.(MapValue); ok && n.Type != "" {
			if mp.InterfaceName == "" {
				mp.InterfaceName = n.Type
			}
			val = mp
		}
		if n.Type != "" {
			env.SetVariableType(n.Name, n.Type)
		}
		if n.IsGlobal {
			if env.IsBlockScope() {
				return nil, fmt.Errorf("cannot declare global/export variable '%s' inside a block or function: global declarations are only permitted at module root level", n.Name)
			}
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
		targetVar := n.VarName()
		if n.Type != "" {
			if !ev.isValidType(n.Type) {
				return nil, fmt.Errorf("argument '%s': type '%s' is not defined", targetVar, n.Type)
			}
			env.SetVariableType(targetVar, n.Type)
			if targetVar != n.Name {
				env.SetVariableType(n.Name, n.Type)
			}
		}
		// S'il existe déjà dans l'environnement (fourni par les CLI args via Name ou Alias), on le valide
		if v, exists := env.Get(n.Name); exists {
			if n.Type != "" && !ev.valMatchesType(v, n.Type) {
				return nil, fmt.Errorf("argument '%s' of type '%s' received invalid value of type '%s'", n.Name, n.Type, v.Type())
			}
			if targetVar != n.Name {
				_ = env.Set(targetVar, v, false)
			}
			if n.IsGlobal {
				if env.IsBlockScope() {
					return nil, fmt.Errorf("cannot declare global/export argument '%s' inside a block or function: global declarations are only permitted at module root level", n.Name)
				}
				env.SetGlobal(n.Name, v)
				if targetVar != n.Name {
					env.SetGlobal(targetVar, v)
				}
			}
			return v, nil
		}
		if v, exists := env.Get(targetVar); exists {
			if n.Type != "" && !ev.valMatchesType(v, n.Type) {
				return nil, fmt.Errorf("argument '%s' of type '%s' received invalid value of type '%s'", targetVar, n.Type, v.Type())
			}
			if n.IsGlobal {
				if env.IsBlockScope() {
					return nil, fmt.Errorf("cannot declare global/export argument '%s' inside a block or function: global declarations are only permitted at module root level", targetVar)
				}
				env.SetGlobal(targetVar, v)
				env.SetGlobal(n.Name, v)
			}
			return v, nil
		}
		if n.DefaultValue != nil {
			val, err := ev.Eval(n.DefaultValue, env)
			if err != nil {
				return nil, err
			}
			if n.Type != "" && !ev.valMatchesType(val, n.Type) {
				return nil, fmt.Errorf("argument '%s' of type '%s' has default value of type '%s'", targetVar, n.Type, val.Type())
			}
			if n.IsGlobal {
				if env.IsBlockScope() {
					return nil, fmt.Errorf("cannot declare global/export argument '%s' inside a block or function: global declarations are only permitted at module root level", targetVar)
				}
				env.SetGlobal(targetVar, val)
				if targetVar != n.Name {
					env.SetGlobal(n.Name, val)
				}
			}
			return val, env.Set(targetVar, val, false)
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
				if existingVal, ok := env.store[alias]; ok {
					if existingUnit, isUnit := existingVal.(*UnitInstance); isUnit && strings.EqualFold(existingUnit.UnitName, bUnit.UnitName) {
						found = true
						break
					}
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
			if n.CatchVar != "" {
				_ = env.Set(n.CatchVar, StringValue{Val: fmt.Sprintf("module '%s' does not exist", unitName)}, false)
				return NilValue{}, nil
			}
			return nil, fmt.Errorf("module '%s' does not exist", unitName)
		}
		if n.CatchVar != "" {
			_ = env.Set(n.CatchVar, NilValue{}, false)
		}
		return NilValue{}, nil
	case *ast.InterfaceStatement:
		if ev.interfaces == nil {
			ev.interfaces = make(map[string]*ast.InterfaceStatement)
		}
		ev.interfaces[n.Name] = n
		for _, f := range n.Fields {
			if f.Params != nil {
				if f.ReturnType != "" && !ev.isValidType(f.ReturnType) {
					return nil, fmt.Errorf("interface '%s': type '%s' is not defined for return type of method '%s'", n.Name, f.ReturnType, f.Name)
				}
				for _, p := range f.Params {
					if p.Type != "" && !ev.isValidType(p.Type) {
						return nil, fmt.Errorf("interface '%s': type '%s' is not defined for parameter '%s' of method '%s'", n.Name, p.Type, p.Name, f.Name)
					}
				}
			} else if f.Type != "" && !ev.isValidType(f.Type) {
				return nil, fmt.Errorf("interface '%s': type '%s' is not defined for field '%s'", n.Name, f.Type, f.Name)
			}
		}
		return NilValue{}, nil
	case *ast.UnitStatement:
		// Enregistre la définition de l'unité
		unitInst := &UnitInstance{
			UnitName:    n.Name,
			Properties:  make(map[string]Value),
			Methods:     make(map[string]NativeMethod),
			Directives:  make(map[string]NativeDirective),
			MethodDefs:  n.Methods,
			IsPrototype: true,
		}
		if n.Includes != "" {
			targetMod := strings.TrimSuffix(n.Includes, "Native")
			for bName, bUnit := range ev.builtins {
				if strings.EqualFold(bName, n.Name) || strings.EqualFold(bName, targetMod) {
					for mName, mFn := range bUnit.Methods {
						unitInst.Methods[mName] = mFn
					}
					for dName, dFn := range bUnit.Directives {
						unitInst.Directives[dName] = dFn
					}
					unitInst.IsPrototype = bUnit.IsPrototype
					break
				}
			}
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
		if isBuiltinUnit(n.UnitName) {
			return nil, fmt.Errorf("built-in module '%s' cannot be instantiated with '@'. Use '%s.<method>()' directly", n.UnitName, n.UnitName)
		}
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
			UnitName:    proto.UnitName,
			Properties:  make(map[string]Value),
			Methods:     make(map[string]NativeMethod),
			Directives:  proto.Directives,
			MethodDefs:  proto.MethodDefs,
			IsPrototype: false,
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
	case *ast.InStatement:
		return ev.evalIn(n, env)
	case *ast.MatchStatement:
		return ev.evalMatch(n, env)
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
		if isTruthy(condVal) {
			return ev.Eval(n.Consequence, env)
		}
		return ev.Eval(n.Alternative, env)
	case *ast.ForStatement:
		return ev.evalFor(n, env)
	case *ast.WhileStatement:
		return ev.evalWhile(n, env)
	case *ast.RangeExpression:
		return ev.evalRange(n, env)
	case *ast.CatchExpression:
		return ev.evalCatch(n, env)
	case *ast.Identifier:
		if strings.EqualFold(n.Value, "nil") || strings.EqualFold(n.Value, "null") {
			return NilValue{}, nil
		}
		if val, ok := env.Get(n.Value); ok {
			return val, nil
		}
		for bName, bUnit := range ev.builtins {
			if strings.EqualFold(bName, n.Value) {
				return bUnit, nil
			}
		}
		return nil, fmt.Errorf("undefined identifier '%s'", n.Value)
	case *ast.PropAccess:
		if env.unitRef != nil {
			if val, ok := env.unitRef.Properties[n.Name]; ok {
				return val, nil
			}
			if fn, ok := env.unitRef.Methods[n.Name]; ok {
				return FunctionValue{Fn: fn}, nil
			}
			return nil, fmt.Errorf("undefined property or method '&%s' on unit '%s'", n.Name, env.unitRef.UnitName)
		}
		return nil, fmt.Errorf("property '&%s' cannot be accessed outside a unit definition", n.Name)
	case *ast.StringLiteral:
		res := n.Value
		spans := ast.ExtractInterpolations(n.Value)
		// Trier de droite à gauche pour que les décalages de sous-chaînes restent valides
		sort.Slice(spans, func(i, j int) bool {
			return spans[i].StartIdx > spans[j].StartIdx
		})
		for _, span := range spans {
			if span.Expr == "" {
				continue
			}
			if strings.HasPrefix(span.Expr, "&") {
				propName := strings.TrimPrefix(span.Expr, "&")
				if env.unitRef == nil {
					return NilValue{}, fmt.Errorf("property '&%s' cannot be accessed outside a unit definition", propName)
				}
				val, ok := env.unitRef.Properties[propName]
				if !ok {
					return NilValue{}, fmt.Errorf("undefined property '&%s' on unit '%s'", propName, env.unitRef.UnitName)
				}
				res = res[:span.StartIdx] + val.String() + res[span.EndIdx+1:]
				continue
			}

			// Analyse et évaluation de l'expression d'interpolation
			subLexer := lexer.New("<interpolation>", span.Expr)
			subParser := parser.New(subLexer)
			parsedExpr := subParser.ParseExpression()
			if len(subParser.Errors()) > 0 {
				return NilValue{}, fmt.Errorf("syntax error in string interpolation '{%s}': %s", span.Expr, subParser.Errors()[0])
			}

			val, err := ev.Eval(parsedExpr, env)
			if err != nil {
				return NilValue{}, fmt.Errorf("error in string interpolation '{%s}': %w", span.Expr, err)
			}
			res = res[:span.StartIdx] + val.String() + res[span.EndIdx+1:]
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
	case *ast.MapLiteral:
		pairs := make(map[string]Value)
		for _, p := range n.Pairs {
			kVal, err := ev.Eval(p.Key, env)
			if err != nil {
				return nil, err
			}
			vVal, err := ev.Eval(p.Value, env)
			if err != nil {
				return nil, err
			}
			pairs[kVal.String()] = vVal
		}
		return MapValue{Pairs: pairs}, nil
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

			modEnv := NewEnvironment(env)
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

			// Évaluation en mode module : seules les déclarations et variables de module sont évaluées.
			// Les instructions de script autonomes (appels de fonctions racine, boucles, etc.) ne sont pas exécutées.
			for _, stmt := range prog.Statements {
				switch s := stmt.(type) {
				case *ast.UseStatement, *ast.FromUseStatement, *ast.UnitStatement, *ast.FuncStatement:
					if _, err := ev.Eval(s, modEnv); err != nil {
						return nil, err
					}
				case *ast.VarStatement:
					if s.IsGlobal || isDeclarativeExpr(s.Value) {
						if _, err := ev.Eval(s, modEnv); err != nil {
							return nil, err
						}
					}
				case *ast.ArgStatement:
					if s.IsGlobal {
						if _, err := ev.Eval(s, modEnv); err != nil {
							return nil, err
						}
					}
				default:
					// Les instructions de script racine (ExpressionStatement, ForStatement, etc.) sont ignorées lors d'un import
				}
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
			targetName := n.Member
			if n.Alias != "" {
				targetName = n.Alias
			}
			unit, err := loadModuleInstance(n.Module)
			if err != nil {
				if n.CatchVar != "" {
					_ = env.Set(n.CatchVar, StringValue{Val: err.Error()}, false)
					_ = env.Set(targetName, NilValue{}, false)
					return NilValue{}, nil
				}
				return nil, err
			}
			if prop, ok := unit.Properties[n.Member]; ok {
				if n.CatchVar != "" {
					_ = env.Set(n.CatchVar, NilValue{}, false)
				}
				if err := env.Set(targetName, prop, false); err != nil {
					return nil, err
				}
				return prop, nil
			}
			if method, ok := unit.Methods[n.Member]; ok {
				if n.CatchVar != "" {
					_ = env.Set(n.CatchVar, NilValue{}, false)
				}
				fnVal := FunctionValue{Name: targetName, Fn: method}
				if err := env.Set(targetName, fnVal, false); err != nil {
					return nil, err
				}
				return fnVal, nil
			}
			if n.CatchVar != "" {
				errMsg := fmt.Sprintf("member '%s' not found on module '%s'", n.Member, n.Module)
				_ = env.Set(n.CatchVar, StringValue{Val: errMsg}, false)
				_ = env.Set(targetName, NilValue{}, false)
				return NilValue{}, nil
			}
			return nil, fmt.Errorf("member '%s' not found on module '%s'", n.Member, n.Module)
		}

		// Cas 2b: membres groupés (auth::{b, c})
		if len(n.Members) > 0 {
			unit, err := loadModuleInstance(n.Module)
			if err != nil {
				return nil, err
			}
			for _, memName := range n.Members {
				if prop, ok := unit.Properties[memName]; ok {
					if err := env.Set(memName, prop, false); err != nil {
						return nil, err
					}
					continue
				}
				if method, ok := unit.Methods[memName]; ok {
					fnVal := FunctionValue{Name: memName, Fn: method}
					if err := env.Set(memName, fnVal, false); err != nil {
						return nil, err
					}
					continue
				}
				return nil, fmt.Errorf("member '%s' not found on module '%s'", memName, n.Module)
			}
			return NilValue{}, nil
		}

		// Cas 2c: module complet sous forme de MapValue (gArgs::@map as args)
		if n.AsMap && len(n.Members) == 0 {
			unit, err := loadModuleInstance(n.Module)
			if err != nil {
				return nil, err
			}
			mapPairs := make(map[string]Value)
			for k, v := range unit.Properties {
				mapPairs[k] = v
			}
			for k, method := range unit.Methods {
				mapPairs[k] = FunctionValue{Name: k, Fn: method}
			}
			mapVal := MapValue{Pairs: mapPairs}
			targetName := n.Module
			if n.Alias != "" {
				targetName = n.Alias
			}
			if err := env.Set(targetName, mapVal, false); err != nil {
				return nil, err
			}
			return mapVal, nil
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
	case *ast.MemberExpression:
		obj, err := ev.Eval(n.Object, env)
		if err != nil {
			return nil, err
		}
		if unit, ok := obj.(*UnitInstance); ok {
			if prop, exists := unit.Properties[n.Property]; exists {
				return prop, nil
			}
			if method, exists := unit.Methods[n.Property]; exists {
				return FunctionValue{Name: n.Property, Fn: method}, nil
			}
			return nil, fmt.Errorf("property '%s' not found on module '%s'", n.Property, unit.UnitName)
		}
		if mp, ok := obj.(MapValue); ok {
			if val, exists := mp.Pairs[n.Property]; exists {
				return val, nil
			}
			ifaceName := mp.InterfaceName
			if ifaceName == "" {
				if ident, ok := n.Object.(*ast.Identifier); ok {
					ifaceName, _ = env.GetVariableType(ident.Value)
				}
			}
			if ifaceName != "" {
				cleanIface := strings.TrimSuffix(ifaceName, "?")
				cleanIface = ev.resolveType(cleanIface)
				if ev.interfaces != nil {
					if iface, ok := ev.interfaces[cleanIface]; ok {
						for _, f := range iface.Fields {
							if f.Name == n.Property {
								return NilValue{}, nil
							}
						}
						return nil, fmt.Errorf("property '%s' does not exist on type '%s'", n.Property, cleanIface)
					}
				}
				if strings.HasPrefix(cleanIface, "@map") && strings.Contains(cleanIface, "{") {
					fields := parseInlineMapFieldDefsRuntime(cleanIface)
					for _, f := range fields {
						if f.Name == n.Property {
							return NilValue{}, nil
						}
					}
					return nil, fmt.Errorf("property '%s' does not exist on type '%s'", n.Property, cleanIface)
				}
			}
			return NilValue{}, nil
		}
		return nil, fmt.Errorf("cannot access property '%s' on %s", n.Property, obj.Type())
	case *ast.ScopeAccessExpression:
		if n.IsUnitSelf {
			if env.unitRef == nil {
				return nil, fmt.Errorf("internal unit reference '&%s::%s' cannot be accessed outside a unit definition", n.Scope, n.Member)
			}
			if prop, exists := env.unitRef.Properties[n.Member]; exists {
				return prop, nil
			}
			if method, exists := env.unitRef.Methods[n.Member]; exists {
				return FunctionValue{Name: n.Member, Fn: method}, nil
			}
			for mName, mFn := range env.unitRef.Methods {
				if strings.EqualFold(mName, n.Member) {
					return FunctionValue{Name: mName, Fn: mFn}, nil
				}
			}
			return nil, fmt.Errorf("member '%s' not found on unit '%s'", n.Member, env.unitRef.UnitName)
		}
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
	if ev.interfaces == nil {
		ev.interfaces = make(map[string]*ast.InterfaceStatement)
	}
	for _, stmt := range prog.Statements {
		if iface, ok := stmt.(*ast.InterfaceStatement); ok {
			ev.interfaces[iface.Name] = iface
		}
	}
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

func (ev *Evaluator) resolveType(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return "any"
	}
	if idx := strings.Index(typeName, "["); idx != -1 && strings.HasSuffix(typeName, "]") {
		baseName := strings.TrimSpace(typeName[:idx])
		key := strings.TrimSuffix(typeName[idx+1:], "]")
		key = strings.Trim(strings.TrimSpace(key), "\"'")

		if ev.interfaces != nil {
			if iface, ok := ev.interfaces[baseName]; ok {
				for _, f := range iface.Fields {
					if f.Name == key {
						return f.Type
					}
				}
			}
		}
	}
	return typeName
}

func (ev *Evaluator) isValidType(typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "any" || typeName == "void" || typeName == "never" || typeName == "nil" || typeName == "null" {
		return true
	}
	typeName = strings.TrimSuffix(typeName, "?")

	if strings.Contains(typeName, " or ") {
		for _, part := range strings.Split(typeName, " or ") {
			if !ev.isValidType(strings.TrimSpace(part)) {
				return false
			}
		}
		return true
	}

	if strings.Contains(typeName, "|") {
		for _, part := range strings.Split(typeName, "|") {
			if !ev.isValidType(strings.TrimSpace(part)) {
				return false
			}
		}
		return true
	}

	if (strings.HasPrefix(typeName, "\"") && strings.HasSuffix(typeName, "\"")) ||
		(strings.HasPrefix(typeName, "'") && strings.HasSuffix(typeName, "'")) {
		return true
	}

	if _, err := strconv.ParseFloat(typeName, 64); err == nil {
		return true
	}

	if typeName == "true" || typeName == "false" {
		return true
	}

	if idx := strings.Index(typeName, "["); idx != -1 && strings.HasSuffix(typeName, "]") {
		baseName := strings.TrimSpace(typeName[:idx])
		key := strings.TrimSuffix(typeName[idx+1:], "]")
		key = strings.Trim(strings.TrimSpace(key), "\"'")
		if strings.EqualFold(baseName, "list") || strings.EqualFold(baseName, "task") || strings.EqualFold(baseName, "map") {
			return ev.isValidType(key)
		}
		if ev.interfaces != nil {
			if _, ok := ev.interfaces[baseName]; ok {
				return true
			}
		}
		return false
	}

	if strings.HasPrefix(typeName, "func(") || strings.HasPrefix(typeName, "@map") {
		return true
	}

	switch strings.ToLower(typeName) {
	case "string", "number", "int", "float", "bool", "boolean", "duration", "list", "map", "task", "range", "function", "unit", "document":
		return true
	}

	if ev.interfaces != nil {
		if _, ok := ev.interfaces[typeName]; ok {
			return true
		}
		for name := range ev.interfaces {
			if strings.EqualFold(name, typeName) {
				return true
			}
		}
	}

	for bName := range ev.builtins {
		if strings.EqualFold(bName, typeName) {
			return true
		}
	}

	stdInterfaces := []string{
		"Document", "SysNative", "ROpt", "FsNative", "NetNative", "CryptoNative",
		"JsonNative", "MathNative", "DateNative", "CmdNative", "PathNative", "StdNative",
	}
	for _, iface := range stdInterfaces {
		if strings.EqualFold(iface, typeName) {
			return true
		}
	}

	return false
}

type runtimeInlineField struct {
	Name     string
	Type     string
	Optional bool
}

func parseInlineMapFieldDefsRuntime(typeStr string) []runtimeInlineField {
	var fields []runtimeInlineField
	braceIdx := strings.Index(typeStr, "{")
	closeIdx := strings.LastIndex(typeStr, "}")
	if braceIdx == -1 || closeIdx == -1 || closeIdx <= braceIdx {
		return fields
	}
	body := strings.TrimSpace(typeStr[braceIdx+1 : closeIdx])
	parts := strings.Split(body, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		colonIdx := strings.Index(p, ":")
		if colonIdx == -1 {
			continue
		}
		namePart := strings.TrimSpace(p[:colonIdx])
		typePart := strings.TrimSpace(p[colonIdx+1:])
		optional := false
		if strings.HasSuffix(namePart, "?") {
			optional = true
			namePart = strings.TrimSpace(strings.TrimSuffix(namePart, "?"))
		}
		if strings.HasSuffix(typePart, "?") {
			optional = true
			typePart = strings.TrimSpace(strings.TrimSuffix(typePart, "?"))
		}
		fields = append(fields, runtimeInlineField{
			Name:     namePart,
			Type:     typePart,
			Optional: optional,
		})
	}
	return fields
}

func (ev *Evaluator) valMatchesType(val Value, expected string) bool {
	if expected == "" || expected == "any" {
		return true
	}
	if val == nil {
		return false
	}
	expected = strings.TrimSpace(expected)

	if strings.HasSuffix(expected, "?") {
		if val == (NilValue{}) {
			return true
		}
		expected = strings.TrimSuffix(expected, "?")
	} else if val == (NilValue{}) {
		return false
	}

	expected = ev.resolveType(expected)

	if strings.Contains(expected, " or ") {
		parts := strings.Split(expected, " or ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if ev.valMatchesType(val, part) {
				return true
			}
		}
		return false
	}

	if strings.Contains(expected, "|") {
		parts := strings.Split(expected, "|")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if ev.valMatchesType(val, part) {
				return true
			}
		}
		return false
	}

	if (strings.HasPrefix(expected, "\"") && strings.HasSuffix(expected, "\"")) ||
		(strings.HasPrefix(expected, "'") && strings.HasSuffix(expected, "'")) {
		expectedLit := strings.Trim(expected, "\"'")
		if sVal, ok := val.(StringValue); ok {
			return sVal.Val == expectedLit
		}
		return false
	}

	if nVal, ok := val.(NumberValue); ok {
		if expectedNum, err := strconv.ParseFloat(expected, 64); err == nil {
			return nVal.Val == expectedNum
		}
	}

	if ev.interfaces != nil {
		if iface, ok := ev.interfaces[expected]; ok {
			mp, isMap := val.(MapValue)
			if !isMap {
				return false
			}
			for _, field := range iface.Fields {
				fieldVal, exists := mp.Pairs[field.Name]
				isOpt := field.Optional || strings.HasSuffix(field.Type, "?")
				if !exists || fieldVal == (NilValue{}) {
					if !isOpt {
						return false
					}
					continue
				}
				cleanFieldType := strings.TrimSuffix(field.Type, "?")
				if !ev.valMatchesType(fieldVal, cleanFieldType) {
					return false
				}
			}
			return true
		}
	}

	if strings.HasPrefix(expected, "@map") && strings.Contains(expected, "{") {
		mp, isMap := val.(MapValue)
		if !isMap {
			return false
		}
		fields := parseInlineMapFieldDefsRuntime(expected)
		for _, field := range fields {
			fieldVal, exists := mp.Pairs[field.Name]
			isOpt := field.Optional || strings.HasSuffix(field.Type, "?")
			if !exists || fieldVal == (NilValue{}) {
				if !isOpt {
					return false
				}
				continue
			}
			cleanFieldType := strings.TrimSuffix(field.Type, "?")
			if !ev.valMatchesType(fieldVal, cleanFieldType) {
				return false
			}
		}
		return true
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
	case "void":
		return val == (NilValue{})
	default:
		if strings.HasPrefix(expected, "list") && val.Type() == ValList {
			return true
		}
		if strings.HasPrefix(expected, "func") && val.Type() == "function" {
			return true
		}
		if strings.HasPrefix(expected, "task") && val.Type() == ValTask {
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
		return false
	}
}

func isDeclarativeExpr(expr ast.Expression) bool {
	if expr == nil {
		return true
	}
	switch expr.(type) {
	case *ast.StringLiteral, *ast.NumberLiteral, *ast.BoolLiteral, *ast.DurationLiteral,
		*ast.ListLiteral, *ast.MapLiteral, *ast.LambdaExpression:
		return true
	}
	return false
}
