package checker_test

import (
	"strings"
	"testing"

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
let s = "{&baseDir}/target"
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
	badCode2 := `
use net as Net
with Net.Fetch("https://example.com") {
    unknown_dir "arg" -> "val"
}
`
	l2 := lexer.New("bad2.xru", badCode2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()

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
	code := `
use std
std.println("Starting")

from "services" use auth as a
from "services" use auth::b
from "services" use (cdn, db)
`
	l := lexer.New("test_mod.xru", code)
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
	tests := []struct {
		name        string
		code        string
		expectedErr string
	}{
		{
			name: "unmapped namespace",
			code: `from "unknownNamespace" use auth`,
			expectedErr: "ERR_UNKNOWN_NAMESPACE",
		},
		{
			name: "nonexistent module in valid namespace",
			code: `from "services" use nonExistentService`,
			expectedErr: "ERR_MODULE_NOT_FOUND",
		},
		{
			name: "nonexistent member on valid module",
			code: `from "services" use auth::nonExistentMember`,
			expectedErr: "ERR_MEMBER_NOT_FOUND",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New("test_err.xru", tc.code)
			p := parser.New(l)
			prog := p.ParseProgram()
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
let isProd = true
let fallbackHost = "http://localhost:3000"
let host: string = if isProd -> "https://api.xypriss.dev" else fallbackHost

use std
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
use net as Net
use std
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





