package checker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func TestCheckerStrictRules(t *testing.T) {
	// 1. Test violation : &prop hors d'une unit
	badCode1 := `
let x = &baseDir
`
	l1 := lexer.New("bad1.xru", badCode1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()

	c1 := checker.New()
	c1.Check(prog1)

	if len(c1.Errors()) == 0 {
		t.Fatal("Expected ERR_INVALID_PROP_ACCESS error, got 0 errors")
	}

	foundPropErr := false
	for _, err := range c1.Errors() {
		if strings.Contains(err, "ERR_INVALID_PROP_ACCESS") {
			foundPropErr = true
		}
	}
	if !foundPropErr {
		t.Errorf("Expected ERR_INVALID_PROP_ACCESS, got: %v", c1.Errors())
	}

	// 1b. Test violation : {&prop} dans une string interpolation hors d'une unit
	badCode1b := `
let s = "$f{&baseDir}/target"
`
	l1b := lexer.New("bad1b.xru", badCode1b)
	p1b := parser.New(l1b)
	prog1b := p1b.ParseProgram()

	c1b := checker.New()
	c1b.Check(prog1b)

	foundInterpPropErr := false
	for _, err := range c1b.Errors() {
		if strings.Contains(err, "ERR_INVALID_PROP_ACCESS") {
			foundInterpPropErr = true
		}
	}
	if !foundInterpPropErr {
		t.Errorf("Expected ERR_INVALID_PROP_ACCESS for interpolation, got: %v", c1b.Errors())
	}

	// 2. Test violation : Directive orpheline dans with Net
	prog2 := &ast.Program{
		Statements: []ast.Statement{
			&ast.UseStatement{Module: "net", Alias: "Net"},
			&ast.WithStatement{
				Target: &ast.CallExpression{
					Caller: &ast.MemberExpression{
						Object:   &ast.Identifier{Value: "Net"},
						Property: "Fetch",
					},
				},
				Body: []ast.Statement{
					&ast.CustomDirectiveInvocation{
						Directive: "unknown_dir",
					},
				},
			},
		},
	}

	c2 := checker.New()
	c2.Check(prog2)

	foundDirErr := false
	for _, err := range c2.Errors() {
		if strings.Contains(err, "ERR_ORPHAN_DIRECTIVE") {
			foundDirErr = true
		}
	}
	if !foundDirErr {
		t.Errorf("Expected ERR_ORPHAN_DIRECTIVE, got: %v", c2.Errors())
	}
}

func TestTypeMismatch(t *testing.T) {
	code := `
let a: string = 3
`
	l := lexer.New("type_err.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) == 0 {
		t.Fatal("Expected ERR_TYPE_MISMATCH, got 0 errors")
	}

	foundTypeErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") {
			foundTypeErr = true
		}
	}
	if !foundTypeErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH, got: %v", c.Errors())
	}
}

func TestCollisionsAndDuplicates(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		expectedErr string
	}{
		{
			name: "duplicate module import",
			code: `
use net
use net
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
		{
			name: "duplicate alias collision",
			code: `
use net as client
use fs as client
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
		{
			name: "variable collides with imported module",
			code: `
use net
let net: string = "hello"
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
		{
			name: "import collides with earlier variable",
			code: `
let net: string = "hello"
use net
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
		{
			name: "two local variables with same name",
			code: `
let a = 1
let a = 2
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
		{
			name: "function local variable collides with imported module",
			code: `
use net
func test() {
    let net = "inner"
}
`,
			expectedErr: "ERR_DUPLICATE_SYMBOL",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New("collision_test.xru", tc.code)
			p := parser.New(l)
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("Parser error in %s: %v", tc.name, p.Errors())
			}

			c := checker.New()
			c.Check(prog)

			found := false
			for _, err := range c.Errors() {
				if strings.Contains(err, tc.expectedErr) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected %s in %s, but got errors: %v", tc.expectedErr, tc.name, c.Errors())
			}
		})
	}
}

func TestCheckerUnknownMethod(t *testing.T) {
	code := `
use std
let a = "test"
std.invalidMethod(a)
`
	l := lexer.New("test_err.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) == 0 {
		t.Fatalf("Expected ERR_UNKNOWN_METHOD, got 0 errors")
	}

	found := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_UNKNOWN_METHOD") && strings.Contains(err, "invalidMethod") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ERR_UNKNOWN_METHOD for invalidMethod, got: %v", c.Errors())
	}

	// Verify legacy 'Utils' module is strictly rejected
	codeLegacy := `
use Utils
`
	lLegacy := lexer.New("test_legacy.xru", codeLegacy)
	pLegacy := parser.New(lLegacy)
	progLegacy := pLegacy.ParseProgram()
	cLegacy := checker.New()
	cLegacy.Check(progLegacy)
	foundLegacyErr := false
	for _, err := range cLegacy.Errors() {
		if strings.Contains(err, "ERR_UNKNOWN_MODULE") {
			foundLegacyErr = true
			break
		}
	}
	if !foundLegacyErr {
		t.Errorf("Expected ERR_UNKNOWN_MODULE for legacy 'Utils', got: %v", cLegacy.Errors())
	}
}

func TestCheckerModulesAndStd(t *testing.T) {
	tmpDir := t.TempDir()
	servicesDir := filepath.Join(tmpDir, "src", "services")
	_ = os.MkdirAll(servicesDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "_mod.xru"), []byte("map {\n    services -> \"./src/services/*\"\n}\n"), 0644)
	_ = os.WriteFile(filepath.Join(servicesDir, "auth.xru"), []byte("global let b = 1\n"), 0644)
	_ = os.WriteFile(filepath.Join(servicesDir, "cdn.xru"), []byte("global let c = 2\n"), 0644)
	_ = os.WriteFile(filepath.Join(servicesDir, "db.xru"), []byte("global let d = 3\n"), 0644)

	code := `
use std
from "services" use auth as a
from "services" use auth::b
from "services" use (cdn, db)

std.println("Starting")
`
	l := lexer.New("test_mod.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	c := checker.New()
	c.SetCurrentFile(filepath.Join(tmpDir, "test_mod.xru"))
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Errorf("Expected 0 errors, got: %v", c.Errors())
	}
}

func TestModFileStrictRules(t *testing.T) {
	// 1. _mod.xru must not contain code or variables
	invalidModCode := `
map {
    services -> "./src/services/*"
}
let a = 1
func test() {}
`
	l := lexer.New("_mod.xru", invalidModCode)
	p := parser.New(l)
	prog := p.ParseProgram()

	c := checker.New()
	c.SetCurrentFile("_mod.xru")
	c.Check(prog)

	foundErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_INVALID_MOD_DECLARATION") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_INVALID_MOD_DECLARATION on _mod.xru with code, got errors: %v", c.Errors())
	}

	// 2. Cannot name a module '_mod'
	reservedModCode := `
mod _mod
`
	lRes := lexer.New("sample.xru", reservedModCode)
	pRes := parser.New(lRes)
	progRes := pRes.ParseProgram()

	cRes := checker.New()
	cRes.Check(progRes)
	foundResErr := false
	for _, err := range cRes.Errors() {
		if strings.Contains(err, "ERR_RESERVED_MOD_NAME") {
			foundResErr = true
			break
		}
	}
	if !foundResErr {
		t.Errorf("Expected ERR_RESERVED_MOD_NAME for mod _mod, got: %v", cRes.Errors())
	}
}

func TestModuleResolutionErrors(t *testing.T) {
	tmpDir := t.TempDir()
	servicesDir := filepath.Join(tmpDir, "src", "services")
	_ = os.MkdirAll(servicesDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "_mod.xru"), []byte("map {\n    services -> \"./src/services/*\"\n}\n"), 0644)
	_ = os.WriteFile(filepath.Join(servicesDir, "auth.xru"), []byte("global let validMember = 1\n"), 0644)

	tests := []struct {
		name        string
		code        string
		expectedErr string
	}{
		{
			name:        "unmapped namespace",
			code:        `from "unknownNamespace" use auth`,
			expectedErr: "ERR_UNKNOWN_NAMESPACE",
		},
		{
			name:        "nonexistent module in valid namespace",
			code:        `from "services" use nonExistentService`,
			expectedErr: "ERR_MODULE_NOT_FOUND",
		},
		{
			name:        "nonexistent member on valid module",
			code:        `from "services" use auth::nonExistentMember`,
			expectedErr: "ERR_MEMBER_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New("test_err.xru", tc.code)
			p := parser.New(l)
			prog := p.ParseProgram()
			c := checker.New()
			c.SetCurrentFile(filepath.Join(tmpDir, "test_err.xru"))
			c.Check(prog)

			found := false
			for _, err := range c.Errors() {
				if strings.Contains(err, tc.expectedErr) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s in %s, got errors: %v", tc.expectedErr, tc.name, c.Errors())
			}
		})
	}
}

func TestCheckerCalculationsAndLogicals(t *testing.T) {
	code := `
let a = 1
let b = 2
let c: bool = (a < b) && !(b == 0)
let d: bool = (a == 1) || (b < 1)
let e: number = -(a + b * 2)
`
	l := lexer.New("test_calc.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Errorf("Expected 0 errors, got: %v", c.Errors())
	}
}

func TestCheckerMutationAndSignatures(t *testing.T) {
	// 1. Valid let mutation with =:
	validCode := `
let x = 10
x =: 20
`
	l := lexer.New("test_mut_valid.xru", validCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)
	if len(c.Errors()) != 0 {
		t.Errorf("Expected 0 errors for valid =:, got: %v", c.Errors())
	}

	// 2. Const mutation error
	constCode := `
const secret = "abc"
secret =: "xyz"
`
	l2 := lexer.New("test_const_mut.xru", constCode)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)
	foundConstErr := false
	for _, e := range c2.Errors() {
		if strings.Contains(e, "ERR_CONST_REASSIGNMENT") {
			foundConstErr = true
			break
		}
	}
	if !foundConstErr {
		t.Errorf("Expected ERR_CONST_REASSIGNMENT for mutating const, got: %v", c2.Errors())
	}

	// 3. Argument count mismatch
	callArgCode := `
func clean(target: string) -> bool {
	return true
}
clean()
`
	l3 := lexer.New("test_arg_count.xru", callArgCode)
	p3 := parser.New(l3)
	prog3 := p3.ParseProgram()
	c3 := checker.New()
	c3.Check(prog3)
	foundArgErr := false
	for _, e := range c3.Errors() {
		if strings.Contains(e, "ERR_ARG_COUNT") {
			foundArgErr = true
			break
		}
	}
	if !foundArgErr {
		t.Errorf("Expected ERR_ARG_COUNT for clean(), got: %v", c3.Errors())
	}
}

func TestCheckerUnusedVariables(t *testing.T) {
	code := `
use std
let usedVar = 1
const unusedConst = 42
let unusedVar = "hello"
std.println(usedVar)
`
	l := lexer.New("test_unused.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	unused := c.UnusedSymbols()
	if len(unused) != 2 {
		t.Fatalf("Expected 2 unused symbols, got %d: %v", len(unused), unused)
	}

	names := make(map[string]bool)
	for _, sym := range unused {
		names[sym.Name] = true
	}

	if !names["unusedConst"] {
		t.Errorf("Expected 'unusedConst' to be marked unused")
	}
	if !names["unusedVar"] {
		t.Errorf("Expected 'unusedVar' to be marked unused")
	}
	if names["usedVar"] {
		t.Errorf("Expected 'usedVar' to NOT be marked unused")
	}
}

func TestIfExpressionChecker(t *testing.T) {
	code := `
use std

let isProd = true
let fallbackHost = "http://localhost:3000"
let host: string = if isProd -> "https://api.xypriss.dev" else fallbackHost
std.println(host)
`
	l := lexer.New("test_if_expr.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Errorf("Expected 0 checker errors, got: %v", c.Errors())
	}

	// Toutes les variables sont utilisées (isProd, fallbackHost, host, std)
	unused := c.UnusedSymbols()
	if len(unused) > 0 {
		t.Errorf("Expected 0 unused symbols, got: %v", unused)
	}
}

func TestAsyncTaskChecker(t *testing.T) {
	validCode := `
use std
use net as Net
let task = with Net.fetch("https://api.xypriss.dev/templates")
let done = task.isDone()
let res = task.wait()
task.cancel()
std.println(done)
std.println(res)
`
	l := lexer.New("test_async_valid.xru", validCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)
	if len(c.Errors()) > 0 {
		t.Errorf("Expected 0 checker errors for valid task methods, got: %v", c.Errors())
	}

	invalidCode := `
use net as Net
let task = with Net.fetch("https://api.xypriss.dev")
task.unknownMethod()
`
	lBad := lexer.New("test_async_bad.xru", invalidCode)
	pBad := parser.New(lBad)
	progBad := pBad.ParseProgram()
	cBad := checker.New()
	cBad.Check(progBad)
	foundErr := false
	for _, e := range cBad.Errors() {
		if strings.Contains(e, "ERR_UNKNOWN_METHOD") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_UNKNOWN_METHOD for unknown task method, got: %v", cBad.Errors())
	}
}

func TestUnitInstanceMethodChecker(t *testing.T) {
	// 1. Calling unit instance method with wrong arg count
	badArgsCode := `
unit TemplateManager {
    func clean(target: string) -> bool {
        return true
    }
}
let mgr = @TemplateManager()
mgr.clean()
`
	l1 := lexer.New("unit_chk1.xru", badArgsCode)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	c1 := checker.New()
	c1.Check(prog1)
	foundArgErr := false
	for _, e := range c1.Errors() {
		if strings.Contains(e, "ERR_ARG_COUNT") {
			foundArgErr = true
			break
		}
	}
	if !foundArgErr {
		t.Errorf("Expected ERR_ARG_COUNT for mgr.clean(), got: %v", c1.Errors())
	}

	// 2. Calling unit instance method with wrong arg type
	badTypeCode := `
unit TemplateManager {
    func clean(target: string) -> bool {
        return true
    }
}
let mgr = @TemplateManager()
mgr.clean(123)
`
	l2 := lexer.New("unit_chk2.xru", badTypeCode)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)
	foundTypeErr := false
	for _, e := range c2.Errors() {
		if strings.Contains(e, "ERR_TYPE_MISMATCH") {
			foundTypeErr = true
			break
		}
	}
	if !foundTypeErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH for mgr.clean(123), got: %v", c2.Errors())
	}

	// 3. Incompatible return type across multiple branches
	badReturnCode := `
unit TemplateManager {
    declare isLocked: bool = false
    func clean(target: string) -> bool {
        if &isLocked == true {
            return false
        }
        return "invalid string return"
    }
}
`
	l3 := lexer.New("unit_chk3.xru", badReturnCode)
	p3 := parser.New(l3)
	prog3 := p3.ParseProgram()
	c3 := checker.New()
	c3.Check(prog3)
	foundRetErr := false
	for _, e := range c3.Errors() {
		if strings.Contains(e, "ERR_TYPE_MISMATCH") && strings.Contains(e, "Cannot return value of type 'string' from function 'clean' expecting 'bool'") {
			foundRetErr = true
			break
		}
	}
	if !foundRetErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH for clean returning 'string' while expecting 'bool', got: %v", c3.Errors())
	}
}

func TestMatchStatementChecker(t *testing.T) {
	// Undefined variable in match target
	code := `
match undefinedVar {
    case "test" {
        let x = 1
    }
}
`
	l := lexer.New("test_match_chk.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)

	foundUndefined := false
	for _, e := range c.Errors() {
		if strings.Contains(e, "ERR_UNDEFINED_VARIABLE") && strings.Contains(e, "undefinedVar") {
			foundUndefined = true
			break
		}
	}
	if !foundUndefined {
		t.Errorf("Expected ERR_UNDEFINED_VARIABLE for match undefinedVar, got: %v", c.Errors())
	}
}

func TestStringAndListMethodChecker(t *testing.T) {
	// Valid calls
	goodCode := `
let s = "hello"
let trimmed = s.trim()
let upper = s.toUpper()
let has = s.contains("ll")

let list = ["a", "b"]
let l = list.len()
let joined = list.join("-")
`
	l1 := lexer.New("good.xru", goodCode)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	c1 := checker.New()
	c1.Check(prog1)
	if len(c1.Errors()) > 0 {
		t.Fatalf("Expected 0 checker errors for valid string/list methods, got: %v", c1.Errors())
	}

	// Invalid method on string
	badCode := `
let s = "hello"
s.nonExistentMethod()
`
	l2 := lexer.New("bad_str_m.xru", badCode)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)

	foundUnknown := false
	for _, e := range c2.Errors() {
		if strings.Contains(e, "ERR_UNKNOWN_METHOD") {
			foundUnknown = true
			break
		}
	}
	if !foundUnknown {
		t.Errorf("Expected ERR_UNKNOWN_METHOD for nonExistentMethod, got: %v", c2.Errors())
	}
}

func TestModernDispatchChecker(t *testing.T) {
	code := `
dispatch "./src"
dispatch "./src", ".rules.xru"
`
	l := lexer.New("dispatch.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	c := checker.New()
	c.Check(prog)
	if len(c.Errors()) > 0 {
		t.Fatalf("Expected 0 checker errors for dispatch, got: %v", c.Errors())
	}
}

func TestNativeMethodSignatures(t *testing.T) {
	// 1. String replace with missing args -> ERR_ARG_COUNT
	code1 := `let res = "hello".replace()`
	l1 := lexer.New("sig1.xru", code1)
	p1 := parser.New(l1)
	c1 := checker.New()
	c1.Check(p1.ParseProgram())
	foundArgErr := false
	for _, e := range c1.Errors() {
		if strings.Contains(e, "ERR_ARG_COUNT") {
			foundArgErr = true
			break
		}
	}
	if !foundArgErr {
		t.Errorf("Expected ERR_ARG_COUNT for .replace(), got: %v", c1.Errors())
	}

	// 2. String replace with invalid argument type -> ERR_TYPE_MISMATCH
	code2 := `let res = "hello".replace(123, "world")`
	l2 := lexer.New("sig2.xru", code2)
	p2 := parser.New(l2)
	c2 := checker.New()
	c2.Check(p2.ParseProgram())
	foundTypeErr := false
	for _, e := range c2.Errors() {
		if strings.Contains(e, "ERR_TYPE_MISMATCH") {
			foundTypeErr = true
			break
		}
	}
	if !foundTypeErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH for .replace(123, ...), got: %v", c2.Errors())
	}

	// 3. Valid native methods call -> 0 errors
	code3 := `
let res = "hello".replace("l", "p")
let upper = "hello".toUpper()
let sub = "hello".contains("ell")
let part = "hello".slice(1, 3)
`
	l3 := lexer.New("sig3.xru", code3)
	p3 := parser.New(l3)
	c3 := checker.New()
	c3.Check(p3.ParseProgram())
	if len(c3.Errors()) > 0 {
		t.Errorf("Expected 0 errors for valid native calls, got: %v", c3.Errors())
	}
}

func TestFromUseUnusedImports(t *testing.T) {
	code := `
from "dll" use XpackD::downloadRelease
from "dll" use gArgs::name
let myName = name
`
	l := lexer.New("test_unused.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	c := checker.New()
	c.Check(prog)

	unused := c.UnusedImports()
	if len(unused) != 1 {
		t.Fatalf("Expected 1 unused import (downloadRelease), got %d: %v", len(unused), unused)
	}
	if unused[0].Name != "downloadRelease" {
		t.Errorf("Expected unused import 'downloadRelease', got '%s'", unused[0].Name)
	}
}

func TestStrictTypeValidationAndOptional(t *testing.T) {
	// 1. Bare object syntax is rejected by parser
	bareCode := `let obj = { a: 1 }`
	pBare := parser.New(lexer.New("bare.xru", bareCode))
	pBare.ParseProgram()
	if len(pBare.Errors()) == 0 {
		t.Fatal("Expected parser error for bare object syntax '{ ... }', got none")
	}

	// 2. Strict property access validation (ERR_NO_SUCH_PROPERTY)
	codeProp := `
type TExample = @map{
    a: string,
    c: number,
}
func test(params: TExample) {
    let x = params.notExist
}
`
	pProp := parser.New(lexer.New("prop.xru", codeProp))
	cProp := checker.New()
	cProp.Check(pProp.ParseProgram())
	foundNoProp := false
	for _, err := range cProp.Errors() {
		if strings.Contains(err, "ERR_NO_SUCH_PROPERTY") && strings.Contains(err, "notExist") {
			foundNoProp = true
			break
		}
	}
	if !foundNoProp {
		t.Errorf("Expected ERR_NO_SUCH_PROPERTY for params.notExist, got errors: %v", cProp.Errors())
	}

	// 3. Strict call argument validation against interface (ERR_MISSING_REQUIRED_FIELD)
	codeArg := `
type TExample = @map{
    a: string,
    c: number,
}
func run(params: TExample) {}
run(@map{ a: "hello" })
`
	pArg := parser.New(lexer.New("arg.xru", codeArg))
	cArg := checker.New()
	cArg.Check(pArg.ParseProgram())
	foundMissing := false
	for _, err := range cArg.Errors() {
		if strings.Contains(err, "missing required property 'c'") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Errorf("Expected missing required property 'c', got errors: %v", cArg.Errors())
	}

	// 4. Return type indexed type resolution & validation
	codeRetOk := `
type TExample = @map{
    c: number,
}
func getC(): TExample["c"] {
    return 42
}
`
	pRetOk := parser.New(lexer.New("ret_ok.xru", codeRetOk))
	cRetOk := checker.New()
	cRetOk.Check(pRetOk.ParseProgram())
	if len(cRetOk.Errors()) > 0 {
		t.Errorf("Expected 0 errors for indexed return type matching number, got: %v", cRetOk.Errors())
	}

	codeRetBad := `
type TExample = @map{
    c: number,
}
func getC(): TExample["c"] {
    return "not a number"
}
`
	pRetBad := parser.New(lexer.New("ret_bad.xru", codeRetBad))
	cRetBad := checker.New()
	cRetBad.Check(pRetBad.ParseProgram())
	foundRetErr := false
	for _, err := range cRetBad.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") {
			foundRetErr = true
			break
		}
	}
	if !foundRetErr {
		t.Errorf("Expected ERR_TYPE_MISMATCH for invalid return type, got errors: %v", cRetBad.Errors())
	}

	// 5. Optional parameters support
	codeOpt := `
use std
func greet(name: string, other: string?, opt2?: number) {
    std.println(name)
}
greet("John")
greet("John", "info")
greet("John", "info", 123)
`
	pOpt := parser.New(lexer.New("opt.xru", codeOpt))
	cOpt := checker.New()
	cOpt.Check(pOpt.ParseProgram())
	if len(cOpt.Errors()) > 0 {
		t.Errorf("Expected 0 errors for optional parameters, got: %v", cOpt.Errors())
	}
}

func TestIndexedTypePropertyNotFound(t *testing.T) {
	code := `
type TExample = @map{
    a: string,
    b: number,
}
func test() -> TExample["notFound"] {
    return "ok"
}
`
	p := parser.New(lexer.New("bad_idx.xru", code))
	c := checker.New()
	c.Check(p.ParseProgram())
	found := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_NO_SUCH_PROPERTY") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ERR_NO_SUCH_PROPERTY for non-existent indexed property, got: %v", c.Errors())
	}
}

func TestContradictoryComparison(t *testing.T) {
	code := `
type TOpts = @map{
    silent?: "bool",
}
let opts: TOpts = @map{ silent: "bool" }
if opts.silent == false {
    let x = 1
}
`
	p := parser.New(lexer.New("comp.xru", code))
	c := checker.New()
	c.Check(p.ParseProgram())
	found := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ERR_TYPE_MISMATCH for contradictory comparison, got: %v", c.Errors())
	}
}

func TestNestedMapAndLambdaReturnValidation(t *testing.T) {
	code := `
type TConfig = @map{
    timeout: duration,
    action: func() -> number,
}
let conf: TConfig = @map{
    timeout: 12,
    action: () -> "42",
}
`
	p := parser.New(lexer.New("nested_map.xru", code))
	c := checker.New()
	c.Check(p.ParseProgram())
	foundTimeoutErr := false
	foundActionErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "timeout") && strings.Contains(err, "expects type 'duration', got 'number'") {
			foundTimeoutErr = true
		}
		if strings.Contains(err, "action") && (strings.Contains(err, "returns type 'string', expected 'number'") || strings.Contains(err, "ERR_TYPE_MISMATCH")) {
			foundActionErr = true
		}
	}
	if !foundTimeoutErr {
		t.Errorf("Expected error for field timeout expects duration, got: %v", c.Errors())
	}
	if !foundActionErr {
		t.Errorf("Expected error for field action lambda return type mismatch, got: %v", c.Errors())
	}
}

func TestDateBuiltinChecker(t *testing.T) {
	code := `
use date
let now = date.now()
let iso = date.iso()
let yr = date.year()
let ts = date.timestamp()
let added = date.add(now, 5, "days")
let diffSec = date.diff(now, added, "seconds")
`
	p := parser.New(lexer.New("date_test.xru", code))
	c := checker.New()
	c.Check(p.ParseProgram())
	if len(c.Errors()) > 0 {
		t.Errorf("Expected 0 errors for valid date unit usage, got: %v", c.Errors())
	}
}

func TestMissingReturnStatement(t *testing.T) {
	code := `
func compute() -> number {
    let x = 10
}
`
	p := parser.New(lexer.New("missing_ret.xru", code))
	c := checker.New()
	c.Check(p.ParseProgram())
	found := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_MISSING_RETURN") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ERR_MISSING_RETURN for function with return type but no return, got: %v", c.Errors())
	}
}

func TestImportAndModOrderStrictRules(t *testing.T) {
	// 1. Mod after use -> ERR_MOD_ORDER
	c1 := `
use std
mod myModule
`
	p1 := parser.New(lexer.New("t1.xru", c1))
	chk1 := checker.New()
	chk1.Check(p1.ParseProgram())
	if !strings.Contains(strings.Join(chk1.Errors(), " "), "ERR_MOD_ORDER") {
		t.Errorf("Expected ERR_MOD_ORDER when mod is placed after use, got: %v", chk1.Errors())
	}

	// 2. Native import after aliased import -> ERR_IMPORT_ORDER
	c2 := `
use net as Net
use std
`
	p2 := parser.New(lexer.New("t2.xru", c2))
	chk2 := checker.New()
	chk2.Check(p2.ParseProgram())
	if !strings.Contains(strings.Join(chk2.Errors(), " "), "ERR_IMPORT_ORDER") {
		t.Errorf("Expected ERR_IMPORT_ORDER when native import is placed after aliased import, got: %v", chk2.Errors())
	}

	// 3. Import after code declaration -> ERR_IMPORT_ORDER
	c3 := `
let a = 1
use std
`
	p3 := parser.New(lexer.New("t3.xru", c3))
	chk3 := checker.New()
	chk3.Check(p3.ParseProgram())
	if !strings.Contains(strings.Join(chk3.Errors(), " "), "ERR_IMPORT_ORDER") {
		t.Errorf("Expected ERR_IMPORT_ORDER when import is placed after code, got: %v", chk3.Errors())
	}

	// 4. Duplicate mod -> ERR_DUPLICATE_MOD
	c4 := `
mod first
mod second
`
	p4 := parser.New(lexer.New("t4.xru", c4))
	chk4 := checker.New()
	chk4.Check(p4.ParseProgram())
	if !strings.Contains(strings.Join(chk4.Errors(), " "), "ERR_DUPLICATE_MOD") {
		t.Errorf("Expected ERR_DUPLICATE_MOD, got: %v", chk4.Errors())
	}

	// 5. Valid strict order: mod -> simple native imports -> aliased imports -> code -> 0 errors
	c5 := `
mod myModule
use std
use date
use net as Net
let x = 10
`
	p5 := parser.New(lexer.New("t5.xru", c5))
	chk5 := checker.New()
	chk5.Check(p5.ParseProgram())
	if len(chk5.Errors()) > 0 {
		t.Errorf("Expected 0 errors for valid import order, got: %v", chk5.Errors())
	}
}

func TestCheckStdFiles(t *testing.T) {
	stdDir := "../../std"
	entries, err := os.ReadDir(stdDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xru") {
			path := filepath.Join(stdDir, e.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(path, string(content)))
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Errorf("[%s] Parser errors:\n%s", e.Name(), strings.Join(p.Errors(), "\n"))
			}
			c := checker.New()
			c.SetCurrentFile(path)
			c.Check(prog)
			if len(c.Errors()) > 0 {
				t.Errorf("[%s] Checker errors:\n%s", e.Name(), strings.Join(c.Errors(), "\n"))
			}
		}
	}
}

func TestUnitSelfScopeAccess(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name: "valid self method call via unit name",
			input: `
			unit Worker {
				func ping() -> string { return "pong" }
				func execute() -> string {
					return &Worker::ping()
				}
			}
			`,
			wantErr: "",
		},
		{
			name: "valid self method call via self keyword",
			input: `
			unit Worker {
				func ping() -> string { return "pong" }
				func execute() -> string {
					return &self::ping()
				}
			}
			`,
			wantErr: "",
		},
		{
			name: "invalid self method call outside unit",
			input: `
			func outside() -> string {
				return &Worker::ping()
			}
			`,
			wantErr: "ERR_INVALID_PROP_ACCESS",
		},
		{
			name: "mismatched unit self ref",
			input: `
			unit Other {}
			unit Worker {
				func execute() -> string {
					return &Other::ping()
				}
			}
			`,
			wantErr: "ERR_INVALID_UNIT_SELF_REF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New("test.xru", tt.input)
			p := parser.New(l)
			prog := p.ParseProgram()
			if len(p.Errors()) > 0 {
				t.Fatalf("Parser errors: %v", p.Errors())
			}
			c := checker.New()
			c.Check(prog)
			if tt.wantErr == "" {
				if len(c.Errors()) > 0 {
					t.Errorf("Expected 0 errors, got: %v", c.Errors())
				}
			} else {
				found := false
				for _, err := range c.Errors() {
					if strings.Contains(err, tt.wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErr, c.Errors())
				}
			}
		})
	}
}

func TestInterfaceSyntaxAndCollisions(t *testing.T) {
	// 1. Valid interface with space separation and optional semicolons
	codeValid := `
	interface Config @map {
		host string;
		port? number;
		retries number
	}
	interface EndpointConfig @map {
		endpoint string
		timeout duration
	}
	interface Service {
		func run(cfg: Config) -> bool
	}
	`
	lVal := lexer.New("valid.xru", codeValid)
	pVal := parser.New(lVal)
	progVal := pVal.ParseProgram()
	if len(pVal.Errors()) > 0 {
		t.Fatalf("Expected 0 parser errors, got: %v", pVal.Errors())
	}
	cVal := checker.New()
	cVal.Check(progVal)
	if len(cVal.Errors()) > 0 {
		t.Fatalf("Expected 0 checker errors, got: %v", cVal.Errors())
	}

	// 2. Colons in interface fields are forbidden and report a parser error
	codeColon := `
	interface InvalidColon {
		host: string
	}
	`
	lColon := lexer.New("colon.xru", codeColon)
	pColon := parser.New(lColon)
	pColon.ParseProgram()
	foundColonErr := false
	for _, err := range pColon.Errors() {
		if strings.Contains(err, "colons ':' are not allowed in interface fields") {
			foundColonErr = true
			break
		}
	}
	if !foundColonErr {
		t.Errorf("Expected colon error in interface fields, got: %v", pColon.Errors())
	}

	// 3. Case-insensitive field collision (a string vs A number) triggers ERR_DUPLICATE_FIELD
	codeCollision := `
	interface DuplicateKeys {
		a string;
		A number;
	}
	`
	lCol := lexer.New("col.xru", codeCollision)
	pCol := parser.New(lCol)
	progCol := pCol.ParseProgram()
	if len(pCol.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", pCol.Errors())
	}
	cCol := checker.New()
	cCol.Check(progCol)
	foundDuplicate := false
	for _, err := range cCol.Errors() {
		if strings.Contains(err, "ERR_DUPLICATE_FIELD") && strings.Contains(err, "conflicts with 'a'") {
			foundDuplicate = true
			break
		}
	}
	if !foundDuplicate {
		t.Errorf("Expected ERR_DUPLICATE_FIELD for 'A' conflicting with 'a', got: %v", cCol.Errors())
	}

	// 4. Case-insensitive method collision in interface
	codeMethodCollision := `
	interface DuplicateMethods {
		func execute() -> void
		func Execute() -> void
	}
	`
	lMCol := lexer.New("mcol.xru", codeMethodCollision)
	pMCol := parser.New(lMCol)
	progMCol := pMCol.ParseProgram()
	if len(pMCol.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", pMCol.Errors())
	}
	cMCol := checker.New()
	cMCol.Check(progMCol)
	foundMethodDuplicate := false
	for _, err := range cMCol.Errors() {
		if strings.Contains(err, "ERR_DUPLICATE_FIELD") && strings.Contains(err, "conflicts with 'execute'") {
			foundMethodDuplicate = true
			break
		}
	}
	if !foundMethodDuplicate {
		t.Errorf("Expected ERR_DUPLICATE_FIELD for 'Execute' conflicting with 'execute', got: %v", cMCol.Errors())
	}
}

func TestTypeAnnotationValidation(t *testing.T) {
	// 1. Interface with undefined parameter type
	codeBadParam := `
	interface MyInterface {
		func run(opts: notFound) -> bool
	}
	`
	l := lexer.New("test.xru", codeBadParam)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)
	foundErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_UNDEFINED_TYPE") && strings.Contains(err, "notFound") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_UNDEFINED_TYPE for 'notFound', got: %v", c.Errors())
	}

	// 2. Valid interface type
	codeGood := `
	interface ROpt @map {
		onMismatch? "fatal" or "warn" or "auto-update"
		mode? "log" or "silent"
	}
	interface SysNative {
		func require(constraint: string, options: ROpt = nil) -> bool
	}
	`
	l2 := lexer.New("test2.xru", codeGood)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)
	if len(c2.Errors()) > 0 {
		t.Errorf("Expected 0 errors for valid interface type ROpt, got: %v", c2.Errors())
	}
}

func TestSysRequireOptionValidation(t *testing.T) {
	// 1. Valid Sys.require with mode: "log"
	codeValid := `
	use sys as Sys
	Sys.require("<= 2.0.0", @map{
		mode: "log"
	})
	`
	l1 := lexer.New("test1.xru", codeValid)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	c1 := checker.New()
	c1.Check(prog1)
	if len(c1.Errors()) > 0 {
		t.Errorf("Expected 0 errors for valid Sys.require with mode 'log', got: %v", c1.Errors())
	}

	// 2. Invalid mode value: "potato"
	codeInvalidVal := `
	use sys as Sys
	Sys.require("<= 2.0.0", @map{
		mode: "potato"
	})
	`
	l2 := lexer.New("test2.xru", codeInvalidVal)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)
	foundTypeMismatch := false
	for _, err := range c2.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") && strings.Contains(err, "mode") {
			foundTypeMismatch = true
			break
		}
	}
	if !foundTypeMismatch {
		t.Errorf("Expected ERR_TYPE_MISMATCH for mode: 'potato', got: %v", c2.Errors())
	}

	// 3. Unknown property: badOption: 123
	codeUnknownProp := `
	use sys as Sys
	Sys.require("<= 2.0.0", @map{
		badOption: 123
	})
	`
	l3 := lexer.New("test3.xru", codeUnknownProp)
	p3 := parser.New(l3)
	prog3 := p3.ParseProgram()
	c3 := checker.New()
	c3.Check(prog3)
	foundUnknownProp := false
	for _, err := range c3.Errors() {
		if strings.Contains(err, "ERR_UNKNOWN_PROPERTY") && strings.Contains(err, "badOption") {
			foundUnknownProp = true
			break
		}
	}
	if !foundUnknownProp {
		t.Errorf("Expected ERR_UNKNOWN_PROPERTY for badOption, got: %v", c3.Errors())
	}
}

func TestIndexedArgTypeAndUnionValidation(t *testing.T) {
	code := `
interface ServerConfig @map {
    host string
    port number
    mode "development" or "production"
}

interface S {
    mode "dev" | "prod"
}

arg env: ServerConfig["mode"] = "production"
arg port: number = 8080
arg appName: string

const config: ServerConfig = @map {
    host: "127.0.0.1",
    port: port,
    mode: env
}
`
	l := lexer.New("test_indexed_args.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	c := checker.New()
	c.Check(prog)
	if len(c.Errors()) > 0 {
		t.Fatalf("Checker errors: %v", c.Errors())
	}
}

func TestLocalInterfaceOverridesBuiltinDocument(t *testing.T) {
	code := `
interface Document {
    test string
}

file "tests.html" (e, d: Document) -> {
    d.test
    d.replace("a", "b")
}
`
	l := lexer.New("test_local_doc.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)

	foundErr := false
	for _, err := range c.Errors() {
		if strings.Contains(err, "ERR_NO_SUCH_PROPERTY") && strings.Contains(err, "replace") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_NO_SUCH_PROPERTY for d.replace with local Document interface, got: %v", c.Errors())
	}
}

func TestIndexedTypeAssignmentCompatibility(t *testing.T) {
	// Case 1: Compatible indexed type ServerConfig["mode"] -> ServerConfig.mode
	validCode := `
interface ServerConfig @map {
    host string
    port number
    mode "development" or "production"
}

arg env: ServerConfig["mode"] = "production"
arg port: number = 8080

const config: ServerConfig = @map {
    host: "127.0.0.1",
    port: port,
    mode: env
}
`
	l := lexer.New("test_valid_index.xru", validCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Fatalf("Expected 0 errors for valid indexed type assignment, got: %v", c.Errors())
	}

	// Case 2: Incompatible indexed type S["mode"] ("dev" or "prod") -> ServerConfig.mode ("development" or "production")
	invalidCode := `
interface ServerConfig @map {
    host string
    port number
    mode "development" or "production"
}

interface S {
    mode "dev" or "prod"
}

arg env: S["mode"] = "prod"
arg port: number = 8080

const config: ServerConfig = @map {
    host: "127.0.0.1",
    port: port,
    mode: env
}
`
	l2 := lexer.New("test_invalid_index.xru", invalidCode)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)

	if len(c2.Errors()) == 0 {
		t.Fatalf("Expected ERR_TYPE_MISMATCH for incompatible union values, got 0 errors")
	}

	foundErr := false
	for _, err := range c2.Errors() {
		if strings.Contains(err, "ERR_TYPE_MISMATCH") && strings.Contains(err, "Property 'mode' of 'config'") && strings.Contains(err, `S["mode"]`) {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected explicit error message 'Property \\'mode\\' of \\'config\\' expects type ... got \\'S[\"mode\"]\\'', got: %v", c2.Errors())
	}
}


func TestMapMethodForbiddenRule(t *testing.T) {
	// 1. Function in map literal
	code1 := `
let m = @map {
    name: "test",
    run: (x) -> { return x + 1 }
}
`
	l1 := lexer.New("test_map_fn.xru", code1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	c1 := checker.New()
	c1.Check(prog1)

	if len(c1.Errors()) == 0 {
		t.Fatalf("Expected ERR_MAP_METHOD_FORBIDDEN for function in map literal, got 0 errors")
	}
	found := false
	for _, err := range c1.Errors() {
		if strings.Contains(err, "ERR_MAP_METHOD_FORBIDDEN") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected ERR_MAP_METHOD_FORBIDDEN error, got: %v", c1.Errors())
	}

	// 2. Method in @map interface
	code2 := `
interface IMap @map {
    name string
    func doSomething() -> void
}
`
	l2 := lexer.New("test_map_iface_fn.xru", code2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	c2 := checker.New()
	c2.Check(prog2)

	if len(c2.Errors()) == 0 {
		t.Fatalf("Expected ERR_MAP_METHOD_FORBIDDEN for method in @map interface, got 0 errors")
	}
	found2 := false
	for _, err := range c2.Errors() {
		if strings.Contains(err, "ERR_MAP_METHOD_FORBIDDEN") {
			found2 = true
			break
		}
	}
	if !found2 {
		t.Errorf("Expected ERR_MAP_METHOD_FORBIDDEN error, got: %v", c2.Errors())
	}
}

func TestMapBuiltinMethodsTypeChecking(t *testing.T) {
	code := `
let m = @map {
    host: "127.0.0.1",
    port: 8080
}

m.set("mode", "production")
let v = m.get("host")
let hasH = m.has("host")
let k = m.keys()
let l = m.len()
`
	l := lexer.New("test_map_methods.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := checker.New()
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Fatalf("Expected 0 errors for valid map builtin calls, got: %v", c.Errors())
	}
}

func TestDispatchTreeScanningAndValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_dispatch_check_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "src", "controllers")
	_ = os.MkdirAll(subDir, 0755)

	// Subdescriptor with an invalid undefined variable
	badSubCode := `use std
if unknownVar == true {
    std.println("Bad")
}
`
	_ = os.WriteFile(filepath.Join(subDir, ".rules.xru"), []byte(badSubCode), 0644)

	mainCode := `mod main
dispatch "./src"
`
	mainPath := filepath.Join(tempDir, "main.xru")
	_ = os.WriteFile(mainPath, []byte(mainCode), 0644)

	l := lexer.New(mainPath, mainCode)
	p := parser.New(l)
	prog := p.ParseProgram()

	c := checker.New()
	c.SetCurrentFile(mainPath)
	c.Check(prog)

	if len(c.Errors()) == 0 {
		t.Fatalf("Expected validation error from dispatched subdescriptor, got 0 errors")
	}

	foundErr := false
	for _, e := range c.Errors() {
		if strings.Contains(e, "ERR_DISPATCH_VALIDATION") || strings.Contains(e, "unknownVar") {
			foundErr = true
			break
		}
	}
	if !foundErr {
		t.Errorf("Expected ERR_DISPATCH_VALIDATION error, got: %v", c.Errors())
	}
}

func TestDispatchTreeWithSubdescriptorImports(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_dispatch_sub_import_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "src", "servers")
	_ = os.MkdirAll(subDir, 0755)

	subRule := `use std
use fs
std.info("Hello from subdescriptor")
`
	_ = os.WriteFile(filepath.Join(subDir, ".rules.xru"), []byte(subRule), 0644)

	mainCode := `use std
use fs
dispatch "./src"
`
	mainPath := filepath.Join(tempDir, "main.xru")
	_ = os.WriteFile(mainPath, []byte(mainCode), 0644)

	l := lexer.New(mainPath, mainCode)
	p := parser.New(l)
	prog := p.ParseProgram()

	c := checker.New()
	c.SetCurrentFile(mainPath)
	c.Check(prog)

	if len(c.Errors()) > 0 {
		t.Fatalf("Expected 0 errors when subdescriptor imports std/fs, got: %v", c.Errors())
	}
}

