package parser_test

import (
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func TestParserComplete(t *testing.T) {
	input := `
use net as Net
arg port: number = 3000
arg timeout: duration = 30s

let targetDir: string = "./my-project"
let isProd: bool = true
let manager: TemplateManager = @TemplateManager()

unit TemplateManager {
    declare baseDir: string = "./src"
    
    directive OPTION(key: string, value: any) {
        Utils.Log("Option set")
    }

    func Clean(target: string) -> bool {
        Fs.Remove("{&baseDir}/{target}")
        return true
    }
}

let downloadTask = with Net.Fetch("https://api.github.com/releases")
let res = downloadTask.wait()

if port == 3000 {
    Utils.Log("Default port used")
} elif port > 8000 {
    Utils.Log("High port")
} else {
    Utils.Log("Custom port")
}

match port {
    case 3000 {
        let pType = "standard"
    }
    default {
        let pType = "other"
    }
}

let headers: map[string, string] = [
    "Accept" -> "application/json",
    "User-Agent" -> "XRU"
]

let files: list[string] = ["index.ts", "server.ts"]
`

	l := lexer.New("sample.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("Parser error: %s", err)
		}
		t.Fatalf("Parser failed with %d errors", len(p.Errors()))
	}

	if len(prog.Statements) == 0 {
		t.Fatal("Expected statements in program, got 0")
	}

	hasUnit := false
	hasWith := false
	hasIf := false
	hasMatch := false

	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.UnitStatement:
			hasUnit = true
		case *ast.WithStatement:
			hasWith = true
		case *ast.VarStatement:
			if _, ok := s.Value.(*ast.WithExpression); ok {
				hasWith = true
			}
		case *ast.ExpressionStatement:
			if _, ok := s.Expression.(*ast.WithExpression); ok {
				hasWith = true
			}
		case *ast.IfStatement:
			hasIf = true
		case *ast.MatchStatement:
			hasMatch = true
		}
	}

	if !hasUnit {
		t.Error("Missing UnitStatement in parsed AST")
	}
	if !hasWith {
		t.Error("Missing WithStatement in parsed AST")
	}
	if !hasIf {
		t.Error("Missing IfStatement in parsed AST")
	}
	if !hasMatch {
		t.Error("Missing MatchStatement in parsed AST")
	}
}

func TestWithAsyncAndRejectBlock(t *testing.T) {
	// 1. Valid async with expression / statement
	validInput := `use net as Net
let downloadTask = with Net.Fetch("https://api.github.com/releases")
let content = downloadTask.wait()
with Net.Fetch("https://example.com")
`
	l := lexer.New("valid.xru", validInput)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Expected 0 errors for valid async with, got: %v", p.Errors())
	}
	if len(prog.Statements) != 4 {
		t.Fatalf("Expected 4 statements, got %d", len(prog.Statements))
	}

	// 2. Reject magic block on with
	invalidInput := `use net as Net
with Net.Fetch("https://api.github.com/releases") {
    on Complete -> (data, err) => { }
}
`
	l2 := lexer.New("invalid.xru", invalidInput)
	p2 := parser.New(l2)
	p2.ParseProgram()
	if len(p2.Errors()) == 0 {
		t.Fatal("Expected syntax error when passing block to 'with', got none")
	}
	foundExpected := false
	for _, err := range p2.Errors() {
		if strings.Contains(err, "'with' operator does not accept a block") {
			foundExpected = true
			break
		}
	}
	if !foundExpected {
		t.Fatalf("Expected error about 'with' not accepting a block, got: %v", p2.Errors())
	}
}

func TestUnclosedBlock(t *testing.T) {
	input := `
if port == 3000 {
    let a = "ok"
`
	l := lexer.New("unclosed.xru", input)
	p := parser.New(l)
	p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatal("Expected parser error for unclosed block, got none")
	}

	foundExpected := false
	for _, err := range p.Errors() {
		if strings.Contains(err, "expected token '}'") || strings.Contains(err, "}") {
			foundExpected = true
		}
	}
	if !foundExpected {
		t.Fatalf("Expected 'expected token '}' error, got: %v", p.Errors())
	}
}

func TestIfExpression(t *testing.T) {
	input := `
let host = if isProd -> "https://api.xypriss.dev" else "http://localhost:3000"
let status = if count > 10 -> "high" else if count > 0 -> "low" else "zero"
`
	l := lexer.New("inline_if.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", p.Errors())
	}

	if len(prog.Statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(prog.Statements))
	}

	varStmt1, ok := prog.Statements[0].(*ast.VarStatement)
	if !ok {
		t.Fatalf("Expected VarStatement, got %T", prog.Statements[0])
	}
	ifExpr1, ok := varStmt1.Value.(*ast.IfExpression)
	if !ok {
		t.Fatalf("Expected IfExpression, got %T", varStmt1.Value)
	}
	if ifExpr1.Condition.String() != "isProd" {
		t.Errorf("Expected condition 'isProd', got '%s'", ifExpr1.Condition.String())
	}

	varStmt2, ok := prog.Statements[1].(*ast.VarStatement)
	if !ok {
		t.Fatalf("Expected VarStatement, got %T", prog.Statements[1])
	}
	ifExpr2, ok := varStmt2.Value.(*ast.IfExpression)
	if !ok {
		t.Fatalf("Expected IfExpression, got %T", varStmt2.Value)
	}
	// Chained else if
	nestedIf, ok := ifExpr2.Alternative.(*ast.IfExpression)
	if !ok {
		t.Fatalf("Expected nested IfExpression in alternative, got %T", ifExpr2.Alternative)
	}
	if nestedIf.Consequence.String() != "\"low\"" {
		t.Errorf("Expected nested consequence '\"low\"', got '%s'", nestedIf.Consequence.String())
	}
}

func TestVarStatementWithoutValue(t *testing.T) {
	input := `const name = 
let x = 1`
	l := lexer.New("test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatalf("Expected parse error for 'const name = ', got 0 errors")
	}
	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err, "expected expression after '=' in variable declaration") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected error to contain 'expected expression after '=' in variable declaration', got %v", p.Errors())
	}

	// Ensure the second statement 'let x = 1' was still parsed successfully
	if len(prog.Statements) != 2 {
		t.Errorf("Expected 2 statements in AST, got %d", len(prog.Statements))
	}
}

func TestMultiModuleFromUseStatement(t *testing.T) {
	input := `from "dll" use (args, archPrune, hydratateMeta)`
	l := lexer.New("test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", p.Errors())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}
	fromUse, ok := prog.Statements[0].(*ast.FromUseStatement)
	if !ok {
		t.Fatalf("Expected FromUseStatement, got %T", prog.Statements[0])
	}
	if len(fromUse.Modules) != 3 {
		t.Fatalf("Expected 3 modules, got %d", len(fromUse.Modules))
	}
	expected := []string{"args", "archPrune", "hydratateMeta"}
	for i, exp := range expected {
		if fromUse.Modules[i] != exp {
			t.Errorf("Expected module %d to be '%s', got '%s'", i, exp, fromUse.Modules[i])
		}
	}
}

func TestGroupedMemberFromUseStatement(t *testing.T) {
	input := `from "dll" use gArgs::@map{author, description, version, name, plugin}`
	l := lexer.New("test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", p.Errors())
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(prog.Statements))
	}
	fromUse, ok := prog.Statements[0].(*ast.FromUseStatement)
	if !ok {
		t.Fatalf("Expected FromUseStatement, got %T", prog.Statements[0])
	}
	if fromUse.Module != "gArgs" {
		t.Errorf("Expected module 'gArgs', got '%s'", fromUse.Module)
	}
	if len(fromUse.Members) != 5 {
		t.Fatalf("Expected 5 members, got %d", len(fromUse.Members))
	}
	expected := []string{"author", "description", "version", "name", "plugin"}
	for i, exp := range expected {
		if fromUse.Members[i] != exp {
			t.Errorf("Expected member %d to be '%s', got '%s'", i, exp, fromUse.Members[i])
		}
	}
}

func TestSemicolonSeparatedStatements(t *testing.T) {
	input := `let a = 1; const b = 2; func add(x, y) { return x + y }; let c = add(a, b);`
	l := lexer.New("test_semi.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", p.Errors())
	}
	if len(prog.Statements) != 4 {
		t.Fatalf("Expected 4 statements, got %d", len(prog.Statements))
	}
}

func TestStdInStatementParsing(t *testing.T) {
	input := `
std::in {
	let a = ::input("your name : ")
	const sel = ::select("choose an image", ["img_01", "img_02"])
	let ok = ::confirm("are you sure?")
}
std.in {
	let b = ::input()
}
`
	l := lexer.New("test_stdin.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Unexpected parser errors: %v", p.Errors())
	}
	if len(prog.Statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(prog.Statements))
	}

	st1, ok1 := prog.Statements[0].(*ast.StdInStatement)
	if !ok1 {
		t.Fatalf("Expected first statement to be StdInStatement, got %T", prog.Statements[0])
	}
	if st1.Scope != "std" {
		t.Errorf("Expected Scope 'std', got '%s'", st1.Scope)
	}
	if len(st1.Body.Statements) != 3 {
		t.Errorf("Expected 3 statements in body, got %d", len(st1.Body.Statements))
	}

	st2, ok2 := prog.Statements[1].(*ast.StdInStatement)
	if !ok2 {
		t.Fatalf("Expected second statement to be StdInStatement, got %T", prog.Statements[1])
	}
	if len(st2.Body.Statements) != 1 {
		t.Errorf("Expected 1 statement in body, got %d", len(st2.Body.Statements))
	}
}


