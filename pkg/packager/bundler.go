package packager

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

// BundleAndMinify bundles main.xru with all imported modules and subdescriptors inlined
// and minifies / mangles variable and function names.
func (p *Packager) BundleAndMinify() (string, error) {
	return p.BundleAndMinifyWithEntry("main.xru")
}

// BundleAndMinifyWithEntry bundles a given entry .xru file with all imported modules and subdescriptors inlined
// and minifies / mangles variable and function names into a single standalone module.
func (p *Packager) BundleAndMinifyWithEntry(entryFile string) (string, error) {
	prog, err := p.BuildBundledProgramWithEntry(entryFile)
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
	return p.BuildBundledProgramWithEntry("main.xru")
}

// BuildBundledProgramWithEntry compiles a given entry .xru file by recursively inlining all local module dependencies
// declared in _mod.xru and all subdescriptors from dispatch statements into a single unified AST.
func (p *Packager) BuildBundledProgramWithEntry(entryFile string) (*ast.Program, error) {
	if entryFile == "" {
		entryFile = "main.xru"
	}
	var entryPath string
	if filepath.IsAbs(entryFile) {
		entryPath = entryFile
	} else {
		entryPath = filepath.Join(p.rootDir, entryFile)
	}

	content, err := os.ReadFile(entryPath)
	if err != nil {
		if os.IsNotExist(err) && entryFile == "main.xru" {
			return nil, nil
		}
		return nil, err
	}

	l := lexer.New(filepath.Base(entryPath), string(content))
	pr := parser.New(l)
	mainProg := pr.ParseProgram()
	if len(pr.Errors()) > 0 {
		return nil, fmt.Errorf("syntax error in %s: %s", filepath.Base(entryPath), pr.Errors()[0])
	}

	modFilePath := modules.FindModConfigFile(filepath.Dir(entryPath))
	if modFilePath == "" {
		modFilePath = modules.FindModConfigFile(p.rootDir)
	}
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
	knownArgs := make(map[string]bool)
	visitedFiles := make(map[string]bool)

	moduleAliases := make(map[string]string)
	importedMembers := make(map[string]string)

	for _, st := range mainProg.Statements {
		if s, ok := st.(*ast.ArgStatement); ok {
			knownArgs[s.Name] = true
			if s.Alias != "" {
				knownArgs[s.Alias] = true
			}
		}
	}

	registerImportedMember := func(mod, member, alias string) {
		target := member
		if alias != "" {
			target = alias
		}
		if knownArgs[member] {
			importedMembers[target] = member
		} else {
			importedMembers[target] = mod + "__" + member
		}
	}

	var inlineModule func(source, modName, alias string) error
	inlineModule = func(source, modName, alias string) error {
		if alias != "" {
			moduleAliases[alias] = modName
		}
		moduleAliases[modName] = modName

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

		for _, st := range subProg.Statements {
			if s, ok := st.(*ast.ArgStatement); ok {
				knownArgs[s.Name] = true
				if s.Alias != "" {
					knownArgs[s.Alias] = true
				}
			}
		}

		// First, resolve dependencies recursively
		for _, st := range subProg.Statements {
			if fus, ok := st.(*ast.FromUseStatement); ok {
				if len(fus.Modules) > 0 {
					for _, m := range fus.Modules {
						if err := inlineModule(fus.Source, m, ""); err != nil {
							return err
						}
					}
				} else if fus.Module != "" {
					if err := inlineModule(fus.Source, fus.Module, fus.Alias); err != nil {
						return err
					}
					if len(fus.Members) > 0 {
						for _, member := range fus.Members {
							registerImportedMember(fus.Module, member, "")
						}
					} else if fus.Member != "" {
						registerImportedMember(fus.Module, fus.Member, fus.Alias)
					}
				}
			}
		}

		// Collect all top-level symbols declared in this module
		declNames := make(map[string]bool)
		for _, st := range subProg.Statements {
			switch s := st.(type) {
			case *ast.FuncStatement:
				declNames[s.Name] = true
			case *ast.VarStatement:
				declNames[s.Name] = true
			case *ast.UnitStatement:
				declNames[s.Name] = true
			}
		}

		// Rename all declared symbols inside subProg to avoid collisions across modules
		for _, st := range subProg.Statements {
			switch s := st.(type) {
			case *ast.FuncStatement:
				s.Name = modName + "__" + s.Name
			case *ast.VarStatement:
				s.Name = modName + "__" + s.Name
			case *ast.UnitStatement:
				s.Name = modName + "__" + s.Name
			case *ast.ReassignStatement:
				if declNames[s.Name] {
					s.Name = modName + "__" + s.Name
				}
			}
		}

		// Transform all internal identifier references in subProg that point to declared symbols
		for i, st := range subProg.Statements {
			subProg.Statements[i] = transformStmt(st, func(expr ast.Expression) ast.Expression {
				if sa, ok := expr.(*ast.ScopeAccessExpression); ok && !sa.IsUnitSelf {
					if knownArgs[sa.Member] {
						return &ast.Identifier{Position: sa.Position, Value: sa.Member}
					}
					if targetMod, ok := moduleAliases[sa.Scope]; ok {
						return &ast.Identifier{Position: sa.Position, Value: targetMod + "__" + sa.Member}
					}
					return &ast.Identifier{Position: sa.Position, Value: sa.Scope + "__" + sa.Member}
				}
				if inst, ok := expr.(*ast.InstantiationExpression); ok {
					if declNames[inst.UnitName] {
						inst.UnitName = modName + "__" + inst.UnitName
					} else if target, ok := importedMembers[inst.UnitName]; ok {
						inst.UnitName = target
					}
					return inst
				}
				if id, ok := expr.(*ast.Identifier); ok {
					if knownArgs[id.Value] {
						return id
					}
					if declNames[id.Value] {
						return &ast.Identifier{Position: id.Position, Value: modName + "__" + id.Value}
					}
					if target, ok := importedMembers[id.Value]; ok {
						return &ast.Identifier{Position: id.Position, Value: target}
					}
				}
				return expr
			})
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
				funcStmts = append(funcStmts, s)
			case *ast.VarStatement:
				varStmts = append(varStmts, s)
			case *ast.UnitStatement:
				unitStmts = append(unitStmts, s)
			}
		}
		return nil
	}

	// First pass on mainProg: inline modules
	for _, stmt := range mainProg.Statements {
		if s, ok := stmt.(*ast.FromUseStatement); ok {
			if len(s.Modules) > 0 {
				for _, m := range s.Modules {
					if err := inlineModule(s.Source, m, ""); err != nil {
						return nil, err
					}
				}
			} else if s.Module != "" {
				if err := inlineModule(s.Source, s.Module, s.Alias); err != nil {
					return nil, err
				}
				if len(s.Members) > 0 {
					for _, member := range s.Members {
						registerImportedMember(s.Module, member, "")
					}
				} else if s.Member != "" {
					registerImportedMember(s.Module, s.Member, s.Alias)
				}
			}
		}
	}

	// Second pass on mainProg: rewrite scope accesses and imported identifiers
	for i, stmt := range mainProg.Statements {
		if reassign, ok := stmt.(*ast.ReassignStatement); ok {
			if !knownArgs[reassign.Name] {
				if target, ok := importedMembers[reassign.Name]; ok {
					reassign.Name = target
				}
			}
		}
		mainProg.Statements[i] = transformStmt(stmt, func(expr ast.Expression) ast.Expression {
			if sa, ok := expr.(*ast.ScopeAccessExpression); ok && !sa.IsUnitSelf {
				if knownArgs[sa.Member] {
					return &ast.Identifier{Position: sa.Position, Value: sa.Member}
				}
				if targetMod, ok := moduleAliases[sa.Scope]; ok {
					return &ast.Identifier{Position: sa.Position, Value: targetMod + "__" + sa.Member}
				}
				return &ast.Identifier{Position: sa.Position, Value: sa.Scope + "__" + sa.Member}
			}
			if inst, ok := expr.(*ast.InstantiationExpression); ok {
				if target, ok := importedMembers[inst.UnitName]; ok {
					inst.UnitName = target
				}
				return inst
			}
			if id, ok := expr.(*ast.Identifier); ok {
				if knownArgs[id.Value] {
					return id
				}
				if target, ok := importedMembers[id.Value]; ok {
					return &ast.Identifier{Position: id.Position, Value: target}
				}
			}
			return expr
		})
	}

	for _, stmt := range mainProg.Statements {
		switch s := stmt.(type) {
		case *ast.FromUseStatement, *ast.ModStatement:
			// Stripped in bundled output
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
			funcStmts = append(funcStmts, s)
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
		default:
			mainStmts = append(mainStmts, s)
		}
	}

	var finalStmts []ast.Statement
	finalStmts = append(finalStmts, useStmts...)
	finalStmts = append(finalStmts, argStmts...)
	finalStmts = append(finalStmts, unitStmts...)
	finalStmts = append(finalStmts, varStmts...)
	finalStmts = append(finalStmts, funcStmts...)
	finalStmts = append(finalStmts, mainStmts...)

	mainProg.Statements = finalStmts

	// Semantic validation: abort build or packing if there are validation errors
	chk := checker.New()
	chk.SetCurrentFile(entryPath)
	chk.Check(mainProg)
	if len(chk.Errors()) > 0 {
		var errMsgs []string
		for _, e := range chk.Errors() {
			errMsgs = append(errMsgs, fmt.Sprintf("  • %s", e))
		}
		return nil, fmt.Errorf("validation error in %s:\n%s", filepath.Base(entryPath), strings.Join(errMsgs, "\n"))
	}

	return mainProg, nil
}

func transformExpr(expr ast.Expression, fn func(ast.Expression) ast.Expression) ast.Expression {
	if expr == nil {
		return nil
	}
	expr = fn(expr)
	switch e := expr.(type) {
	case *ast.PrefixExpression:
		e.Right = transformExpr(e.Right, fn)
	case *ast.InfixExpression:
		e.Left = transformExpr(e.Left, fn)
		e.Right = transformExpr(e.Right, fn)
	case *ast.CallExpression:
		e.Caller = transformExpr(e.Caller, fn)
		for i, a := range e.Arguments {
			e.Arguments[i] = transformExpr(a, fn)
		}
	case *ast.InstantiationExpression:
		for i, a := range e.Arguments {
			e.Arguments[i] = transformExpr(a, fn)
		}
	case *ast.MemberExpression:
		e.Object = transformExpr(e.Object, fn)
	case *ast.ListLiteral:
		for i, el := range e.Elements {
			e.Elements[i] = transformExpr(el, fn)
		}
	case *ast.MapLiteral:
		for i, p := range e.Pairs {
			e.Pairs[i].Key = transformExpr(p.Key, fn)
			e.Pairs[i].Value = transformExpr(p.Value, fn)
		}
	case *ast.LambdaExpression:
		e.Body = transformExpr(e.Body, fn)
	case *ast.BlockStatement:
		for i, st := range e.Statements {
			e.Statements[i] = transformStmt(st, fn)
		}
	case *ast.IfExpression:
		e.Condition = transformExpr(e.Condition, fn)
		e.Consequence = transformExpr(e.Consequence, fn)
		e.Alternative = transformExpr(e.Alternative, fn)
	case *ast.CatchExpression:
		e.Target = transformExpr(e.Target, fn)
		if fbExpr, ok := e.Fallback.(ast.Expression); ok {
			e.Fallback = transformExpr(fbExpr, fn)
		}
	case *ast.WithExpression:
		e.Target = transformExpr(e.Target, fn)
	}
	return expr
}

func transformStmt(stmt ast.Statement, fn func(ast.Expression) ast.Expression) ast.Statement {
	if stmt == nil {
		return nil
	}
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		s.Expression = transformExpr(s.Expression, fn)
	case *ast.VarStatement:
		s.Value = transformExpr(s.Value, fn)
	case *ast.ReassignStatement:
		s.Value = transformExpr(s.Value, fn)
	case *ast.ReturnStatement:
		s.Value = transformExpr(s.Value, fn)
	case *ast.IfStatement:
		s.Condition = transformExpr(s.Condition, fn)
		for i, st := range s.Consequence {
			s.Consequence[i] = transformStmt(st, fn)
		}
		for i, eb := range s.ElifBlocks {
			s.ElifBlocks[i].Condition = transformExpr(eb.Condition, fn)
			for j, st := range eb.Body {
				s.ElifBlocks[i].Body[j] = transformStmt(st, fn)
			}
		}
		for i, st := range s.Alternative {
			s.Alternative[i] = transformStmt(st, fn)
		}
	case *ast.ForStatement:
		s.Iterable = transformExpr(s.Iterable, fn)
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.WhileStatement:
		s.Condition = transformExpr(s.Condition, fn)
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.MatchStatement:
		s.Target = transformExpr(s.Target, fn)
		for i, c := range s.Cases {
			s.Cases[i].Pattern = transformExpr(c.Pattern, fn)
			for j, st := range c.Body {
				s.Cases[i].Body[j] = transformStmt(st, fn)
			}
		}
		for i, st := range s.DefaultBlock {
			s.DefaultBlock[i] = transformStmt(st, fn)
		}
	case *ast.FileStatement:
		if s.PathExpr != nil {
			s.PathExpr = transformExpr(s.PathExpr, fn)
		}
		if s.Handler != nil {
			s.Handler = transformExpr(s.Handler, fn)
		}
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.InStatement:
		s.Path = transformExpr(s.Path, fn)
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.FuncStatement:
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.WithStatement:
		if s.Target != nil {
			s.Target = transformExpr(s.Target, fn)
		}
		for i, st := range s.Body {
			s.Body[i] = transformStmt(st, fn)
		}
	case *ast.BlockStatement:
		for i, st := range s.Statements {
			s.Statements[i] = transformStmt(st, fn)
		}
	}
	return stmt
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

		// Validation sémantique stricte du descripteur dispatché
		subChk := checker.New()
		subChk.SetCurrentFile(path)
		subChk.SetupDescriptorScope()
		subChk.Check(subProg)
		if len(subChk.Errors()) > 0 {
			var errMsgs []string
			for _, e := range subChk.Errors() {
				errMsgs = append(errMsgs, fmt.Sprintf("  • %s", e))
			}
			return nil, fmt.Errorf("validation error in %s:\n%s", path, strings.Join(errMsgs, "\n"))
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
			st = transformStmt(st, func(expr ast.Expression) ast.Expression {
				if sa, ok := expr.(*ast.ScopeAccessExpression); ok {
					if sa.Scope == "main" || sa.Scope == "root" {
						return &ast.Identifier{Position: sa.Position, Value: sa.Member}
					}
				}
				return expr
			})
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
	counter       int
	nameMap       map[string]string
	reserved      map[string]bool
	argParams     map[string]bool
	mapVars       map[string]bool
	mapInterfaces map[string]bool
	localScopes   []map[string]string
}

func NewMangler(prog *ast.Program) *Mangler {
	m := &Mangler{
		nameMap:       make(map[string]string),
		reserved:      make(map[string]bool),
		argParams:     make(map[string]bool),
		mapVars:       make(map[string]bool),
		mapInterfaces: make(map[string]bool),
	}

	// Keywords & built-in primitives to never mangle
	builtins := []string{
		"keep", "Keep", "drop", "Drop", "UseVariant", "useVariant", "Local",
		"file", "in", "if", "else", "elif", "for", "match", "case", "default",
		"return", "with", "true", "false", "nil", "null", "void", "arg",
		"print", "println", "len", "append", "exit", "std", "fs", "Fs", "cmd", "Cmd",
		"crypto", "Crypto", "net", "path", "exists", "exec", "getRandomBytes",
		"replace", "replaceFuzzy", "removeLine", "removeLines", "removeBalanced", "removeBetween",
		"cleanGaps", "cleanGap",
		"setJson", "getJson", "content", "save", "as", "or", "and", "not", "is", "fn",
		"func", "let", "const", "use", "mod", "from", "to", "by", "unit", "interface",
		"type", "set", "get", "has", "delete", "clear", "merge", "keys", "values", "entries",
		"size", "while", "catch", "on", "header", "directive", "global", "pass", "includes",
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
	res := ""
	for {
		rem := n % 26
		res = string(rune('a'+rem)) + res
		n = n/26 - 1
		if n < 0 {
			break
		}
	}
	// Avoid collisions with reserved words
	if m.reserved[res] {
		return m.nextName()
	}
	return res
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
	// Pass 1: register map interfaces
	for _, stmt := range prog.Statements {
		if iface, ok := stmt.(*ast.InterfaceStatement); ok && iface.IsMap {
			m.mapInterfaces[iface.Name] = true
		}
	}

	// Pass 1.1: register all top-level user function, arg, variable, and unit names so calls and refs resolve properly
	for _, stmt := range prog.Statements {
		if fn, ok := stmt.(*ast.FuncStatement); ok {
			if !m.reserved[fn.Name] {
				_ = m.getMangledName(fn.Name)
			}
		}
		if u, ok := stmt.(*ast.UnitStatement); ok {
			if !m.reserved[u.Name] {
				_ = m.getMangledName(u.Name)
			}
		}
		if v, ok := stmt.(*ast.VarStatement); ok {
			varName := v.Name
			if !m.reserved[v.Name] {
				varName = m.getMangledName(v.Name)
			}
			if _, isMap := v.Value.(*ast.MapLiteral); isMap || v.Type == "map" || v.Type == "@map" || strings.HasPrefix(v.Type, "@map") || m.mapInterfaces[v.Type] {
				m.mapVars[v.Name] = true
				m.mapVars[varName] = true
			}
		}
		if arg, ok := stmt.(*ast.ArgStatement); ok {
			alias := m.nameMap[arg.Name]
			if arg.Type == "map" || arg.Type == "@map" || strings.HasPrefix(arg.Type, "@map") || m.mapInterfaces[arg.Type] {
				m.mapVars[arg.Name] = true
				if alias != "" {
					m.mapVars[alias] = true
				}
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

	case *ast.InterfaceStatement:
		mapKw := ""
		if s.IsMap {
			mapKw = " @map"
		}
		var fields []string
		for _, f := range s.Fields {
			opt := ""
			if f.Optional {
				opt = "?"
			}
			fields = append(fields, fmt.Sprintf("%s%s %s", f.Name, opt, f.Type))
		}
		return fmt.Sprintf("interface %s%s{%s}", s.Name, mapKw, strings.Join(fields, ";"))

	case *ast.UnitStatement:
		unitName := s.Name
		if !m.reserved[unitName] {
			unitName = m.getMangledName(unitName)
		}
		inc := ""
		if s.Includes != "" {
			incTarget := s.Includes
			if resolved := m.resolveName(incTarget); resolved != "" {
				incTarget = resolved
			}
			inc = " extends " + incTarget
		}
		var parts []string
		for _, p := range s.Properties {
			valStr := ""
			if p.Value != nil {
				valStr = " = " + m.minifyExpression(p.Value)
			}
			typeStr := ""
			if p.Type != "" {
				typeStr = ": " + p.Type
			}
			vis := ""
			if p.IsPrivate {
				vis = "private "
			}
			parts = append(parts, fmt.Sprintf("declare %s%s%s%s", vis, p.Name, typeStr, valStr))
		}
		for _, d := range s.Directives {
			var bodyParts []string
			for _, st := range d.Body {
				ms := m.minifyStatement(st)
				if ms != "" {
					bodyParts = append(bodyParts, ms)
				}
			}
			var params []string
			for _, param := range d.Parameters {
				params = append(params, param.Name)
			}
			parts = append(parts, fmt.Sprintf("directive %s(%s){%s}", d.Name, strings.Join(params, ","), strings.Join(bodyParts, ";")))
		}
		for _, mth := range s.Methods {
			m.pushScope()
			var params []string
			for _, p := range mth.Parameters {
				pName := m.defineLocal(p.Name)
				params = append(params, pName)
			}
			var bodyParts []string
			for _, st := range mth.Body {
				ms := m.minifyStatement(st)
				if ms != "" {
					bodyParts = append(bodyParts, ms)
				}
			}
			m.popScope()
			parts = append(parts, fmt.Sprintf("func %s(%s){%s}", mth.Name, strings.Join(params, ","), strings.Join(bodyParts, ";")))
		}
		return fmt.Sprintf("unit %s%s{%s}", unitName, inc, strings.Join(parts, ";"))

	case *ast.MatchStatement:
		targetStr := m.minifyExpression(s.Target)
		var cases []string
		for _, c := range s.Cases {
			patStr := m.minifyExpression(c.Pattern)
			var bodyParts []string
			for _, st := range c.Body {
				ms := m.minifyStatement(st)
				if ms != "" {
					bodyParts = append(bodyParts, ms)
				}
			}
			cases = append(cases, fmt.Sprintf("case %s{%s}", patStr, strings.Join(bodyParts, ";")))
		}
		if len(s.DefaultBlock) > 0 {
			var defParts []string
			for _, st := range s.DefaultBlock {
				ms := m.minifyStatement(st)
				if ms != "" {
					defParts = append(defParts, ms)
				}
			}
			cases = append(cases, fmt.Sprintf("default{%s}", strings.Join(defParts, ";")))
		}
		return fmt.Sprintf("match %s{%s}", targetStr, strings.Join(cases, ";"))

	case *ast.VarStatement:
		kw := "let"
		if s.IsConst {
			kw = "const"
		}
		varName := s.Name
		if !m.reserved[varName] {
			if len(m.localScopes) > 0 {
				varName = m.defineLocal(varName)
			} else {
				varName = m.getMangledName(varName)
			}
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
		var pathStr string
		if s.Path != nil {
			pathStr = " " + m.minifyExpression(s.Path)
		}
		var bodyParts []string
		for _, st := range s.Body {
			ms := m.minifyStatement(st)
			if ms != "" {
				bodyParts = append(bodyParts, ms)
			}
		}
		return fmt.Sprintf("in%s{%s}", pathStr, strings.Join(bodyParts, ";"))

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
		// If e.Object is a map, convert property access to .get("prop")
		if m.isMapExpr(e.Object) && !isBuiltinMapMethod(e.Property) {
			return fmt.Sprintf("%s.get(\"%s\")", obj, e.Property)
		}
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
		return fmt.Sprintf("{%s}", strings.Join(parts, ";"))

	case *ast.ScopeAccessExpression:
		if e.IsUnitSelf {
			return fmt.Sprintf("&%s::%s", e.Scope, e.Member)
		}
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
				if len(m.localScopes) > 0 {
					errVar = m.defineLocal(errVar)
				} else {
					errVar = m.getMangledName(errVar)
				}
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
		unitName := e.UnitName
		if resolved := m.resolveName(unitName); resolved != "" {
			unitName = resolved
		}
		var args []string
		for _, a := range e.Arguments {
			args = append(args, m.minifyExpression(a))
		}
		return fmt.Sprintf("@%s(%s)", unitName, strings.Join(args, ","))

	case *ast.MapLiteral:
		var pairs []string
		for _, p := range e.Pairs {
			kStr := m.minifyExpression(p.Key)
			if strLit, ok := p.Key.(*ast.StringLiteral); ok {
				if isIdentString(strLit.Value) {
					kStr = strLit.Value
				}
			}
			vStr := m.minifyExpression(p.Value)
			pairs = append(pairs, fmt.Sprintf("%s:%s", kStr, vStr))
		}
		return fmt.Sprintf("@map{%s}", strings.Join(pairs, ","))

	default:
		return e.String()
	}
}


func (m *Mangler) isMapExpr(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.Identifier:
		return m.mapVars[e.Value] || m.mapVars[m.resolveName(e.Value)]
	case *ast.MapLiteral:
		return true
	case *ast.CallExpression:
		if member, ok := e.Caller.(*ast.MemberExpression); ok {
			prop := strings.ToLower(member.Property)
			return prop == "set" || prop == "merge"
		}
	}
	return false
}

func isBuiltinMapMethod(prop string) bool {
	switch strings.ToLower(prop) {
	case "clear", "delete", "entries", "has", "keys", "len", "size", "values", "get", "set", "merge":
		return true
	}
	return false
}

func isIdentString(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}
