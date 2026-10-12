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
	"github.com/Nehonix-Team/xru/pkg/parser"
)

// BundleAndMinify bundles main.xru with all subdescriptors inlined into in "..." { ... }
// and minifies / mangles variable and function names.
func (p *Packager) BundleAndMinify() (string, error) {
	mainPath := filepath.Join(p.rootDir, "main.xru")
	content, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	l := lexer.New("main.xru", string(content))
	pr := parser.New(l)
	prog := pr.ParseProgram()
	if len(pr.Errors()) > 0 {
		return "", fmt.Errorf("syntax error in main.xru: %s", pr.Errors()[0])
	}

	// 1. Bundle: inlining dispatch statements with in "<relDir>" { ... }
	var bundledStmts []ast.Statement
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
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
				return "", err
			}
			bundledStmts = append(bundledStmts, inlinedBlocks...)
		default:
			bundledStmts = append(bundledStmts, stmt)
		}
	}
	prog.Statements = bundledStmts

	// 2. Mangling & Minification
	mangler := NewMangler(prog)
	minified := mangler.Minify(prog)
	return minified, nil
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

	// Preserve declared #ARG and arg names
	for _, stmt := range prog.Statements {
		if argStmt, ok := stmt.(*ast.ArgStatement); ok {
			m.reserved[argStmt.Name] = true
			m.reserved[strings.ToLower(argStmt.Name)] = true
			m.reserved[strings.ToUpper(argStmt.Name)] = true
			m.argParams[argStmt.Name] = true
			m.argParams[strings.ToLower(argStmt.Name)] = true
			m.argParams[strings.ToUpper(argStmt.Name)] = true
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
	var sb strings.Builder

	// Pass 1: register all top-level user function names so calls can resolve
	for _, stmt := range prog.Statements {
		if fn, ok := stmt.(*ast.FuncStatement); ok {
			if !m.reserved[fn.Name] {
				_ = m.getMangledName(fn.Name)
			}
		}
	}

	for _, stmt := range prog.Statements {
		str := m.minifyStatement(stmt)
		if str != "" {
			sb.WriteString(str)
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

func (m *Mangler) minifyStatement(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.ArgStatement:
		// Preserve arg declarations
		if s.DefaultValue != nil {
			return fmt.Sprintf("arg %s: %s = %s", s.Name, s.Type, m.minifyExpression(s.DefaultValue))
		}
		return fmt.Sprintf("arg %s: %s", s.Name, s.Type)

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
		if s.IsGlobal {
			prefix = "global func"
		}
		return fmt.Sprintf("%s %s(%s){\n%s\n}", prefix, fnName, strings.Join(params, ","), strings.Join(bodyParts, "\n"))

	case *ast.InStatement:
		pathStr := m.minifyExpression(s.Path)
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		return fmt.Sprintf("in %s{\n%s\n}", pathStr, strings.Join(bodyParts, "\n"))

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
		return fmt.Sprintf("file %s{\n%s\n}", pathStr, strings.Join(bodyParts, "\n"))

	case *ast.IfStatement:
		condStr := m.minifyExpression(s.Condition)
		var consParts []string
		for _, st := range s.Consequence {
			ms := m.minifyStatement(st)
			if ms != "" {
				consParts = append(consParts, ms)
			}
		}
		res := fmt.Sprintf("if %s{\n%s\n}", condStr, strings.Join(consParts, "\n"))

		for _, elif := range s.ElifBlocks {
			elifCond := m.minifyExpression(elif.Condition)
			var elifParts []string
			for _, st := range elif.Body {
				ms := m.minifyStatement(st)
				if ms != "" {
					elifParts = append(elifParts, ms)
				}
			}
			res += fmt.Sprintf(" elif %s{\n%s\n}", elifCond, strings.Join(elifParts, "\n"))
		}

		if len(s.Alternative) > 0 {
			var altParts []string
			for _, st := range s.Alternative {
				ms := m.minifyStatement(st)
				if ms != "" {
					altParts = append(altParts, ms)
				}
			}
			res += fmt.Sprintf(" else{\n%s\n}", strings.Join(altParts, "\n"))
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
			return fmt.Sprintf("for %s,%s in %s{\n%s\n}", itemName, idxName, iterStr, strings.Join(bodyParts, "\n"))
		}
		return fmt.Sprintf("for %s in %s{\n%s\n}", itemName, iterStr, strings.Join(bodyParts, "\n"))

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

	default:
		return e.String()
	}
}
