package packager

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

// BundleAndMinify bundles main.xru with all imported modules and subdescriptors inlined
// and minifies / mangles variable and function names.
func (p *Packager) BundleAndMinify() (string, error) {
	prog, err := p.BuildBundledProgram()
	if err != nil {
		return "", err
	}
	if prog == nil {
		return "", nil
	}
	mangler := NewMangler(prog)
	return mangler.Minify(prog), nil
}

// BuildBundledProgram compiles main.xru by recursively inlining all local module dependencies
// declared in _mod.xru and all subdescriptors from dispatch statements into a single unified AST.
func (p *Packager) BuildBundledProgram() (*ast.Program, error) {
	mainPath := filepath.Join(p.rootDir, "main.xru")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	l := lexer.New("main.xru", string(content))
	pr := parser.New(l)
	mainProg := pr.ParseProgram()
	if len(pr.Errors()) > 0 {
		return nil, fmt.Errorf("syntax error in main.xru: %s", pr.Errors()[0])
	}

	modFilePath := modules.FindModConfigFile(p.rootDir)
	var modCfg *modules.ModConfig
	if modFilePath != "" {
		if cfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 {
			modCfg = cfg
		}
	}

	var useStmts []ast.Statement
	var argStmts []ast.Statement
	var varStmts []ast.Statement
	var funcStmts []ast.Statement
	var unitStmts []ast.Statement
	var mainStmts []ast.Statement

	seenUses := make(map[string]bool)
	seenArgs := make(map[string]bool)
	seenFuncs := make(map[string]bool)
	visitedFiles := make(map[string]bool)

	var inlineModule func(source, modName string) error
	inlineModule = func(source, modName string) error {
		if modCfg == nil || source == "root" {
			return nil
		}
		modDetails, err := modCfg.ResolveModule(source, modName)
		if err != nil {
			return fmt.Errorf("cannot resolve module '%s' in namespace '%s': %w", modName, source, err)
		}
		absPath, _ := filepath.Abs(modDetails.FilePath)
		if visitedFiles[absPath] {
			return nil
		}
		visitedFiles[absPath] = true

		subContent, err := os.ReadFile(modDetails.FilePath)
		if err != nil {
			return fmt.Errorf("cannot read module file '%s': %w", modDetails.FilePath, err)
		}
		subLex := lexer.New(modDetails.FilePath, string(subContent))
		subParser := parser.New(subLex)
		subProg := subParser.ParseProgram()
		if len(subParser.Errors()) > 0 {
			return fmt.Errorf("syntax error in module '%s': %s", modDetails.FilePath, subParser.Errors()[0])
		}

		// First, resolve dependencies recursively
		for _, st := range subProg.Statements {
			if fus, ok := st.(*ast.FromUseStatement); ok {
				if len(fus.Modules) > 0 {
					for _, m := range fus.Modules {
						if err := inlineModule(fus.Source, m); err != nil {
							return err
						}
					}
				} else if fus.Module != "" {
					if err := inlineModule(fus.Source, fus.Module); err != nil {
						return err
					}
				}
			}
		}

		// Extract declarations
		for _, st := range subProg.Statements {
			switch s := st.(type) {
			case *ast.UseStatement:
				key := s.Module + "::" + s.Alias
				if !seenUses[key] {
					seenUses[key] = true
					useStmts = append(useStmts, s)
				}
			case *ast.ArgStatement:
				if !seenArgs[s.Name] {
					seenArgs[s.Name] = true
					argStmts = append(argStmts, s)
				}
			case *ast.FuncStatement:
				if !seenFuncs[s.Name] {
					seenFuncs[s.Name] = true
					funcStmts = append(funcStmts, s)
				}
			case *ast.VarStatement:
				if s.IsGlobal {
					varStmts = append(varStmts, s)
				}
			case *ast.UnitStatement:
				unitStmts = append(unitStmts, s)
			}
		}
		return nil
	}

	for _, stmt := range mainProg.Statements {
		switch s := stmt.(type) {
		case *ast.FromUseStatement:
			if len(s.Modules) > 0 {
				for _, m := range s.Modules {
					if err := inlineModule(s.Source, m); err != nil {
						return nil, err
					}
				}
			} else if s.Module != "" {
				if err := inlineModule(s.Source, s.Module); err != nil {
					return nil, err
				}
			}
		case *ast.UseStatement:
			key := s.Module + "::" + s.Alias
			if !seenUses[key] {
				seenUses[key] = true
				useStmts = append(useStmts, s)
			}
		case *ast.ArgStatement:
			if !seenArgs[s.Name] {
				seenArgs[s.Name] = true
				argStmts = append(argStmts, s)
			}
		case *ast.FuncStatement:
			if !seenFuncs[s.Name] {
				seenFuncs[s.Name] = true
				funcStmts = append(funcStmts, s)
			}
		case *ast.UnitStatement:
			unitStmts = append(unitStmts, s)
		case *ast.DispatchTreeStatement:
			targetDir := s.Path
			if targetDir == "" && s.PathExpr != nil {
				if strLit, ok := s.PathExpr.(*ast.StringLiteral); ok {
					targetDir = strLit.Value
				}
			}
			if targetDir == "" {
				targetDir = "./"
			}
			pattern := s.Pattern
			if pattern == "" {
				pattern = ".rules.xru"
			}
			inlinedBlocks, err := p.inlineSubdescriptors(targetDir, pattern)
			if err != nil {
				return nil, err
			}
			mainStmts = append(mainStmts, inlinedBlocks...)
		case *ast.ModStatement:
			// ignore
		default:
			mainStmts = append(mainStmts, s)
		}
	}

	var finalStmts []ast.Statement
	finalStmts = append(finalStmts, useStmts...)
	finalStmts = append(finalStmts, argStmts...)
	finalStmts = append(finalStmts, varStmts...)
	finalStmts = append(finalStmts, funcStmts...)
	finalStmts = append(finalStmts, unitStmts...)
	finalStmts = append(finalStmts, mainStmts...)

	mainProg.Statements = finalStmts
	return mainProg, nil
}

func (p *Packager) inlineSubdescriptors(targetDir, pattern string) ([]ast.Statement, error) {
	fullTarget := filepath.Join(p.rootDir, targetDir)
	var inlined []ast.Statement

	var discoveredFiles []string
	_ = filepath.WalkDir(fullTarget, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := entry.Name()
		matched, _ := filepath.Match(pattern, name)
		if matched || name == pattern {
			discoveredFiles = append(discoveredFiles, path)
		}
		return nil
	})

	sort.Strings(discoveredFiles)

	for _, path := range discoveredFiles {
		relDir, err := filepath.Rel(p.rootDir, filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		relDir = filepath.ToSlash(relDir)

		subContent, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		subLex := lexer.New(path, string(subContent))
		subParser := parser.New(subLex)
		subProg := subParser.ParseProgram()
		if len(subParser.Errors()) > 0 {
			return nil, fmt.Errorf("syntax error in %s: %s", path, subParser.Errors()[0])
		}

		var bodyStmts []ast.Statement
		for _, st := range subProg.Statements {
			if _, ok := st.(*ast.FromUseStatement); ok {
				continue
			}
			if _, ok := st.(*ast.ModStatement); ok {
				continue
			}
			if _, ok := st.(*ast.UseStatement); ok {
				continue
			}
			if _, ok := st.(*ast.ArgStatement); ok {
				continue
			}
			st = rewriteScopeCalls(st, "main")
			bodyStmts = append(bodyStmts, st)
		}

		inStmt := &ast.InStatement{
			Position: subProg.Pos(),
			Path:     &ast.StringLiteral{Value: relDir},
			Body:     bodyStmts,
		}
		inlined = append(inlined, inStmt)
	}

	return inlined, nil
}

func rewriteScopeCalls(stmt ast.Statement, scopeName string) ast.Statement {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		s.Expression = rewriteScopeExpr(s.Expression, scopeName)
		return s
	default:
		return stmt
	}
}

func rewriteScopeExpr(expr ast.Expression, scopeName string) ast.Expression {
	switch e := expr.(type) {
	case *ast.CallExpression:
		if scopeAcc, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if scopeAcc.Scope == scopeName {
				e.Caller = &ast.Identifier{Position: scopeAcc.Position, Value: scopeAcc.Member}
			}
		}
		return e
	default:
		return expr
	}
}

// Mangler renames non-reserved identifiers and minifies code
type Mangler struct {
	counter     int
	nameMap     map[string]string
	reserved    map[string]bool
	argParams   map[string]bool
	localScopes []map[string]string
}

func NewMangler(prog *ast.Program) *Mangler {
	m := &Mangler{
		nameMap:   make(map[string]string),
		reserved:  make(map[string]bool),
		argParams: make(map[string]bool),
	}

	// Keywords & built-in primitives to never mangle
	builtins := []string{
		"keep", "Keep", "drop", "Drop", "UseVariant", "useVariant", "Local",
		"file", "in", "if", "else", "elif", "for", "match", "case", "default",
		"return", "with", "true", "false", "nil", "null", "void", "arg",
		"print", "println", "len", "append", "exit", "std", "fs", "Fs", "cmd", "Cmd",
		"crypto", "Crypto", "net", "path", "exists", "exec", "getRandomBytes",
		"replace", "replaceFuzzy", "removeLine", "removeLines", "removeBalanced", "removeBetween",
		"setJson", "getJson", "content", "save",
	}
	for _, b := range builtins {
		m.reserved[b] = true
	}

	// For each arg statement: assign a mangled alias so internal variable name becomes _a, _b...
	for _, stmt := range prog.Statements {
		if argStmt, ok := stmt.(*ast.ArgStatement); ok {
			var alias string
			if argStmt.Alias != "" {
				alias = argStmt.Alias
			} else {
				alias = m.nextName()
				argStmt.Alias = alias
			}
			m.nameMap[argStmt.Name] = alias
			m.nameMap[strings.ToLower(argStmt.Name)] = alias
			m.nameMap[strings.ToUpper(argStmt.Name)] = alias
			m.nameMap[alias] = alias
			m.argParams[argStmt.Name] = true
		}
	}

	return m
}

func (m *Mangler) pushScope() {
	m.localScopes = append(m.localScopes, make(map[string]string))
}

func (m *Mangler) popScope() {
	if len(m.localScopes) > 0 {
		m.localScopes = m.localScopes[:len(m.localScopes)-1]
	}
}

func (m *Mangler) defineLocal(name string) string {
	mangled := m.nextName()
	if len(m.localScopes) > 0 {
		m.localScopes[len(m.localScopes)-1][name] = mangled
	}
	return mangled
}

func (m *Mangler) resolveName(name string) string {
	for i := len(m.localScopes) - 1; i >= 0; i-- {
		if mangled, ok := m.localScopes[i][name]; ok {
			return mangled
		}
	}
	if m.reserved[name] {
		return name
	}
	if mangled, ok := m.nameMap[name]; ok {
		return mangled
	}
	return name
}

func (m *Mangler) nextName() string {
	n := m.counter
	m.counter++
	var name string
	if n < 26 {
		name = fmt.Sprintf("_%c", 'a'+n)
	} else {
		name = fmt.Sprintf("_%c%d", 'a'+(n%26), n/26)
	}
	// Avoid collisions with reserved words
	if m.reserved[name] {
		return m.nextName()
	}
	return name
}

func (m *Mangler) getMangledName(orig string) string {
	if m.reserved[orig] {
		return orig
	}
	if mangled, ok := m.nameMap[orig]; ok {
		return mangled
	}
	mangled := m.nextName()
	m.nameMap[orig] = mangled
	return mangled
}

func (m *Mangler) Minify(prog *ast.Program) string {
	// Pass 1: register all top-level user function names so calls can resolve
	for _, stmt := range prog.Statements {
		if fn, ok := stmt.(*ast.FuncStatement); ok {
			if !m.reserved[fn.Name] {
				_ = m.getMangledName(fn.Name)
			}
		}
	}

	var topStmts []string
	for _, stmt := range prog.Statements {
		str := m.minifyStatement(stmt)
		if str != "" {
			topStmts = append(topStmts, str)
		}
	}

	return strings.Join(topStmts, ";\n") + ";\n"
}

func (m *Mangler) minifyStatement(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.ArgStatement:
		prefix := ""
		if s.IsGlobal {
			prefix = "global "
		}
		alias := s.VarName()
		if alias != s.Name {
			if s.DefaultValue != nil {
				return fmt.Sprintf("%sarg %s as %s: %s = %s", prefix, s.Name, alias, s.Type, m.minifyExpression(s.DefaultValue))
			}
			return fmt.Sprintf("%sarg %s as %s: %s", prefix, s.Name, alias, s.Type)
		}
		if s.DefaultValue != nil {
			return fmt.Sprintf("%sarg %s: %s = %s", prefix, s.Name, s.Type, m.minifyExpression(s.DefaultValue))
		}
		return fmt.Sprintf("%sarg %s: %s", prefix, s.Name, s.Type)

	case *ast.UseStatement:
		if s.Alias != "" {
			return fmt.Sprintf("use %s as %s", s.Module, s.Alias)
		}
		return fmt.Sprintf("use %s", s.Module)

	case *ast.ModStatement:
		return "" // Strip mod statement in bundled package

	case *ast.FromUseStatement:
		return "" // Strip from...use statements in bundled package

	case *ast.VarStatement:
		kw := "let"
		if s.IsConst {
			kw = "const"
		}
		varName := s.Name
		if !m.reserved[varName] {
			varName = m.getMangledName(varName)
		}
		return fmt.Sprintf("%s %s = %s", kw, varName, m.minifyExpression(s.Value))

	case *ast.FuncStatement:
		fnName := s.Name
		if !m.reserved[fnName] {
			fnName = m.getMangledName(fnName)
		}
		m.pushScope()
		var params []string
		for _, p := range s.Parameters {
			pName := m.defineLocal(p.Name)
			params = append(params, pName)
		}
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		m.popScope()
		prefix := "func"
		return fmt.Sprintf("%s %s(%s){%s}", prefix, fnName, strings.Join(params, ","), strings.Join(bodyParts, ";"))

	case *ast.InStatement:
		pathStr := m.minifyExpression(s.Path)
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		return fmt.Sprintf("in %s{%s}", pathStr, strings.Join(bodyParts, ";"))

	case *ast.FileStatement:
		pathStr := s.Path
		if s.PathExpr != nil {
			pathStr = m.minifyExpression(s.PathExpr)
		} else {
			pathStr = fmt.Sprintf("\"%s\"", pathStr)
		}

		if s.Handler != nil {
			handlerStr := m.minifyExpression(s.Handler)
			return fmt.Sprintf("file %s,%s", pathStr, handlerStr)
		}
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		return fmt.Sprintf("file %s{%s}", pathStr, strings.Join(bodyParts, ";"))

	case *ast.IfStatement:
		condStr := m.minifyExpression(s.Condition)
		var consParts []string
		for _, st := range s.Consequence {
			ms := m.minifyStatement(st)
			if ms != "" {
				consParts = append(consParts, ms)
			}
		}
		res := fmt.Sprintf("if %s{%s}", condStr, strings.Join(consParts, ";"))

		for _, elif := range s.ElifBlocks {
			elifCond := m.minifyExpression(elif.Condition)
			var elifParts []string
			for _, st := range elif.Body {
				ms := m.minifyStatement(st)
				if ms != "" {
					elifParts = append(elifParts, ms)
				}
			}
			res += fmt.Sprintf(" elif %s{%s}", elifCond, strings.Join(elifParts, ";"))
		}

		if len(s.Alternative) > 0 {
			var altParts []string
			for _, st := range s.Alternative {
				ms := m.minifyStatement(st)
				if ms != "" {
					altParts = append(altParts, ms)
				}
			}
			res += fmt.Sprintf(" else{%s}", strings.Join(altParts, ";"))
		}
		return res

	case *ast.ForStatement:
		m.pushScope()
		itemName := m.defineLocal(s.Item)
		iterStr := m.minifyExpression(s.Iterable)
		idxName := ""
		if s.Index != "" {
			idxName = m.defineLocal(s.Index)
		}
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		m.popScope()
		if idxName != "" {
			return fmt.Sprintf("for %s,%s in %s{%s}", itemName, idxName, iterStr, strings.Join(bodyParts, ";"))
		}
		return fmt.Sprintf("for %s in %s{%s}", itemName, iterStr, strings.Join(bodyParts, ";"))

	case *ast.ReassignStatement:
		target := s.Name
		if !m.reserved[target] && !strings.HasPrefix(target, "&") {
			target = m.getMangledName(target)
		}
		return fmt.Sprintf("%s=:%s", target, m.minifyExpression(s.Value))

	case *ast.ReturnStatement:
		if s.Value != nil {
			return fmt.Sprintf("return %s", m.minifyExpression(s.Value))
		}
		return "return"

	case *ast.ExpressionStatement:
		return m.minifyExpression(s.Expression)

	case *ast.BlockStatement:
		var parts []string
		for _, st := range s.Statements {
			ms := m.minifyStatement(st)
			if ms != "" {
				parts = append(parts, ms)
			}
		}
		return fmt.Sprintf("{%s}", strings.Join(parts, ";"))

	case *ast.WhileStatement:
		condStr := m.minifyExpression(s.Condition)
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		return fmt.Sprintf("while %s{%s}", condStr, strings.Join(bodyParts, ";"))

	case *ast.WithStatement:
		targetStr := ""
		if s.Target != nil {
			targetStr = m.minifyExpression(s.Target)
		}
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		if targetStr != "" {
			return fmt.Sprintf("with %s{%s}", targetStr, strings.Join(bodyParts, ";"))
		}
		return fmt.Sprintf("with{%s}", strings.Join(bodyParts, ";"))

	default:
		return s.String()
	}
}

func (m *Mangler) minifyExpression(expr ast.Expression) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		return m.resolveName(e.Value)

	case *ast.StringLiteral:
		// Handle internal string interpolation: $f{var}
		val := e.Value
		spans := ast.ExtractInterpolations(val)
		if len(spans) > 0 {
			for _, span := range spans {
				origExpr := span.Expr
				resolved := m.resolveName(origExpr)
				if resolved != origExpr {
					val = strings.Replace(val, "$f{"+origExpr+"}", "$f{"+resolved+"}", 1)
				}
			}
		}
		return fmt.Sprintf("%q", val)

	case *ast.NumberLiteral:
		return e.Value

	case *ast.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"

	case *ast.DurationLiteral:
		return e.Value

	case *ast.PrefixExpression:
		return fmt.Sprintf("%s%s", e.Operator, m.minifyExpression(e.Right))

	case *ast.InfixExpression:
		left := m.minifyExpression(e.Left)
		right := m.minifyExpression(e.Right)
		return fmt.Sprintf("%s%s%s", left, e.Operator, right)

	case *ast.CallExpression:
		caller := m.minifyExpression(e.Caller)
		var args []string
		for _, a := range e.Arguments {
			args = append(args, m.minifyExpression(a))
		}
		return fmt.Sprintf("%s(%s)", caller, strings.Join(args, ","))

	case *ast.MemberExpression:
		obj := m.minifyExpression(e.Object)
		// Property names (methods / fields) are never mangled
		return fmt.Sprintf("%s.%s", obj, e.Property)

	case *ast.PropAccess:
		return "&" + e.Name

	case *ast.ListLiteral:
		var elems []string
		for _, el := range e.Elements {
			elems = append(elems, m.minifyExpression(el))
		}
		return fmt.Sprintf("[%s]", strings.Join(elems, ","))

	case *ast.LambdaExpression:
		m.pushScope()
		var params []string
		for _, p := range e.Parameters {
			pName := m.defineLocal(p.Name)
			params = append(params, pName)
		}
		bodyStr := m.minifyExpression(e.Body)
		m.popScope()
		return fmt.Sprintf("(%s)->%s", strings.Join(params, ","), bodyStr)

	case *ast.BlockStatement:
		var parts []string
		for _, st := range e.Statements {
			ms := m.minifyStatement(st)
			if ms != "" {
				parts = append(parts, ms)
			}
		}
		return fmt.Sprintf("{\n%s\n}", strings.Join(parts, "\n"))

	case *ast.ScopeAccessExpression:
		member := e.Member
		if !m.reserved[member] {
			member = m.getMangledName(member)
		}
		return member

	case *ast.GlobalCallExpression:
		fnName := e.Function
		if !m.reserved[fnName] {
			fnName = m.getMangledName(fnName)
		}
		var args []string
		for _, a := range e.Arguments {
			args = append(args, m.minifyExpression(a))
		}
		return fmt.Sprintf("%s(%s)", fnName, strings.Join(args, ","))

	case *ast.GlobalAccessExpression:
		fnName := e.Name
		if !m.reserved[fnName] {
			fnName = m.getMangledName(fnName)
		}
		return fnName

	case *ast.WithExpression:
		if e.Target != nil {
			return fmt.Sprintf("with %s", m.minifyExpression(e.Target))
		}
		return "with"

	case *ast.CatchExpression:
		targetStr := m.minifyExpression(e.Target)
		res := targetStr + " catch"
		if e.ErrVar != "" {
			errVar := e.ErrVar
			if !m.reserved[errVar] {
				errVar = m.getMangledName(errVar)
			}
			res += " " + errVar
		}
		if e.Fallback != nil {
			if fbExpr, ok := e.Fallback.(ast.Expression); ok {
				res += fmt.Sprintf(" or %s", m.minifyExpression(fbExpr))
			} else if fbStmt, ok := e.Fallback.(ast.Statement); ok {
				res += fmt.Sprintf(" or %s", m.minifyStatement(fbStmt))
			} else {
				res += fmt.Sprintf(" or %s", e.Fallback.String())
			}
		}
		return res

	case *ast.IfExpression:
		return fmt.Sprintf("if %s -> %s else %s", m.minifyExpression(e.Condition), m.minifyExpression(e.Consequence), m.minifyExpression(e.Alternative))

	case *ast.RangeExpression:
		return fmt.Sprintf("%s..%s", m.minifyExpression(e.Start), m.minifyExpression(e.End))

	case *ast.InstantiationExpression:
		return fmt.Sprintf("@%s()", e.UnitName)

	default:
		return e.String()
	}
}
