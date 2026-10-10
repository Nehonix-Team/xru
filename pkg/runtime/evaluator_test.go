package runtime_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func TestEvaluatorBasic(t *testing.T) {
	code := `
use std
use fs as Fs

arg app_name: string = "demo-app"
let version = "1.0.0"

std.println("<green>[TEST]</> Orchestration {app_name} v{version}")
`
	l := lexer.New("runtime_test.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	_, err := ev.Eval(prog, env)
	if err != nil {
		t.Fatalf("Runtime evaluation error: %v", err)
	}

	val, ok := env.Get("version")
	if !ok || val.String() != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got '%v'", val)
	}

	appVal, ok := env.Get("app_name")
	if !ok || appVal.String() != "demo-app" {
		t.Errorf("Expected app_name to be 'demo-app', got '%v'", appVal)
	}
}

func TestFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	testFilePath := tmpDir + "/test.txt"
	initialContent := `line 1
// BEGIN BLOCK
block line A
block line B
// END BLOCK
line 2
import cdn from "cdn";
line 3
`
	if err := os.WriteFile(testFilePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	code := `
file "` + testFilePath + `" {
    Replace("line 1", "first line")
    RemoveBlock("// BEGIN BLOCK", "// END BLOCK")
    RemoveLines("cdn")
}
`
	l := lexer.New("test_file_ops.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	updatedBytes, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}
	updated := string(updatedBytes)

	if !strings.Contains(updated, "first line") {
		t.Errorf("Expected 'first line' in updated content, got: %s", updated)
	}
	if strings.Contains(updated, "block line A") || strings.Contains(updated, "BEGIN BLOCK") {
		t.Errorf("Expected block to be removed, but found in content: %s", updated)
	}
	if strings.Contains(updated, "cdn") {
		t.Errorf("Expected 'cdn' lines to be removed, but found in content: %s", updated)
	}
	if !strings.Contains(updated, "line 2") || !strings.Contains(updated, "line 3") {
		t.Errorf("Expected line 2 and line 3 preserved, got: %s", updated)
	}
}

func TestGlobalFunctionAndDispatchTreeBlock(t *testing.T) {
	code := `
global func Helper(name: string) {
    let msg = "Hello " + name
    return msg
}

dispatch_tree "./src", ".rules.xru" {
    pass MODE = "minimal"
    pass CDN = false
}
`
	l := lexer.New("test_global.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	var dispatchedPath, dispatchedPattern string
	var passedEnv *runtime.Environment

	ev.SetTreeDispatcher(func(dt *ast.DispatchTreeStatement, eval *runtime.Evaluator, treeEnv *runtime.Environment) error {
		dispatchedPath = dt.Path
		dispatchedPattern = dt.Pattern
		passedEnv = treeEnv
		return nil
	})

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	if dispatchedPath != "./src" || dispatchedPattern != ".rules.xru" {
		t.Errorf("Unexpected dispatch path/pattern: %s, %s", dispatchedPath, dispatchedPattern)
	}

	modeVal, ok := passedEnv.Get("MODE")
	if !ok || modeVal.String() != "minimal" {
		t.Errorf("Expected MODE 'minimal' passed to tree, got %v", modeVal)
	}

	cdnVal, ok := passedEnv.Get("CDN")
	if !ok || cdnVal.String() != "false" {
		t.Errorf("Expected CDN 'false' passed to tree, got %v", cdnVal)
	}

	// Verify global function is accessible from child env
	fnVal, ok := passedEnv.Get("Helper")
	if !ok {
		t.Fatalf("Expected global function 'Helper' accessible in passedEnv")
	}
	userFn, ok := fnVal.(runtime.UserFunction)
	if !ok {
		t.Fatalf("Expected 'Helper' to be a UserFunction")
	}
	if userFn.Stmt.Name != "Helper" {
		t.Errorf("Expected Helper function, got %s", userFn.Stmt.Name)
	}
}

func TestGlobalFunctionInvocationSyntax(t *testing.T) {
	code := `
global func Greet(name: string) {
    let msg = "Hello {name}"
    return msg
}

func LocalGreet(name: string) {
    let msg = "Hi {name}"
    return msg
}

let r1 = ::Greet("Alice")
let r2 = global::Greet("Bob")
let r3 = LocalGreet("Charlie")
`
	l := lexer.New("test_syntax.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	r1, _ := env.Get("r1")
	if r1.String() != "Hello Alice" {
		t.Errorf("Expected 'Hello Alice', got '%s'", r1.String())
	}

	r2, _ := env.Get("r2")
	if r2.String() != "Hello Bob" {
		t.Errorf("Expected 'Hello Bob', got '%s'", r2.String())
	}

	r3, _ := env.Get("r3")
	if r3.String() != "Hi Charlie" {
		t.Errorf("Expected 'Hi Charlie', got '%s'", r3.String())
	}

	// Test: calling global function as local must fail
	failCode := `
let fail = Greet("Eve")
`
	lFail := lexer.New("test_fail.xru", failCode)
	pFail := parser.New(lFail)
	progFail := pFail.ParseProgram()
	_, errFail := ev.Eval(progFail, env)
	if errFail == nil || !strings.Contains(errFail.Error(), "cannot call global function 'Greet' as local") {
		t.Errorf("Expected error calling global function as local, got: %v", errFail)
	}

	// Test: calling local function as global must fail
	failGlobalCode := `
let fail2 = ::LocalGreet("Dave")
`
	lFail2 := lexer.New("test_fail2.xru", failGlobalCode)
	pFail2 := parser.New(lFail2)
	progFail2 := pFail2.ParseProgram()
	_, errFail2 := ev.Eval(progFail2, env)
	if errFail2 == nil || !strings.Contains(errFail2.Error(), "undefined global function 'LocalGreet'") {
		t.Errorf("Expected error calling local function as global, got: %v", errFail2)
	}
}

func TestCryptoUnitGetRandomBytes(t *testing.T) {
	code := `
use crypto as Crypto
use std

let secret1 = Crypto.GetRandomBytes(32)
let secret2 = Crypto.getRandomBytes(16)
let secret3 = Crypto.randomHex(32)
`
	l := lexer.New("test_crypto.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	s1, ok := env.Get("secret1")
	if !ok || len(s1.String()) != 64 {
		t.Errorf("Expected 64 hex characters for 32 bytes, got %v", s1)
	}

	s2, ok := env.Get("secret2")
	if !ok || len(s2.String()) != 32 {
		t.Errorf("Expected 32 hex characters for 16 bytes, got %v", s2)
	}

	s3, ok := env.Get("secret3")
	if !ok || len(s3.String()) != 64 {
		t.Errorf("Expected 64 hex characters from Utils, got %v", s3)
	}

	if s1.String() == s3.String() {
		t.Errorf("Expected two different random byte sequences")
	}
}

func TestRuntimeCollisions(t *testing.T) {
	// 1. Two imports with same alias
	code1 := `
use net as client
use fs as client
`
	l1 := lexer.New("t1.xru", code1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	ev := runtime.NewEvaluator()
	env1 := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog1, env1); err == nil {
		t.Fatalf("Expected runtime error on duplicate import collision, got nil")
	}

	// 2. Variable collides with imported module
	code2 := `
use net
let net = "string"
`
	l2 := lexer.New("t2.xru", code2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected runtime error on variable colliding with module, got nil")
	}
}

func TestArithmeticAndLogicalExpressions(t *testing.T) {
	code := `
use std

let a = 10
let b = 5

let sum = a + 1
let diff = a - b
let prod = a * 2
let div = a / 2
let rem = a % 3

let isLess = b < 10
let isLeq = b <= 5
let isGreater = a > 5
let isGeq = a >= 10
let isEqual = a == 10
let isNeq = a != b

let andTrue = (a > 5) && (b < 10)
let andFalse = (a > 5) && (b > 10)
let orTrue = (a < 5) || (b == 5)
let orWord = (a < 5) or (b == 5)

let notVal = !isLess
let negVal = -a
`
	l := lexer.New("test_math.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	checkNum := func(name string, expected float64) {
		v, ok := env.Get(name)
		if !ok {
			t.Fatalf("Variable %s not found", name)
		}
		if nv, ok := v.(runtime.NumberValue); !ok || nv.Val != expected {
			t.Errorf("Expected %s = %f, got %v", name, expected, v)
		}
	}

	checkBool := func(name string, expected bool) {
		v, ok := env.Get(name)
		if !ok {
			t.Fatalf("Variable %s not found", name)
		}
		if bv, ok := v.(runtime.BoolValue); !ok || bv.Val != expected {
			t.Errorf("Expected %s = %t, got %v", name, expected, v)
		}
	}

	checkNum("sum", 11)
	checkNum("diff", 5)
	checkNum("prod", 20)
	checkNum("div", 5)
	checkNum("rem", 1)
	checkNum("negVal", -10)

	checkBool("isLess", true)
	checkBool("isLeq", true)
	checkBool("isGreater", true)
	checkBool("isGeq", true)
	checkBool("isEqual", true)
	checkBool("isNeq", true)

	checkBool("andTrue", true)
	checkBool("andFalse", false)
	checkBool("orTrue", true)
	checkBool("orWord", true)
	checkBool("notVal", false)
}

func TestMutationAndConstProtection(t *testing.T) {
	codeLet := `
let x = 10
x =: 42
global let g = "initial"
g =: "updated"
`
	l := lexer.New("test_mut.xru", codeLet)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	xVal, ok := env.Get("x")
	if !ok || xVal.(runtime.NumberValue).Val != 42 {
		t.Fatalf("Expected x = 42, got %v", xVal)
	}
	gVal, ok := env.Get("g")
	if !ok || gVal.(runtime.StringValue).Val != "updated" {
		t.Fatalf("Expected g = updated, got %v", gVal)
	}

	// Const re-assignment must fail
	codeConst := `
const pi = 3.14
pi =: 3.14159
`
	l2 := lexer.New("test_const.xru", codeConst)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	if len(p2.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p2.Errors())
	}
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected error when mutating const, got nil")
	}
}

func TestEarlyReturnInIfBlock(t *testing.T) {
	code := `
let executedAfterReturn = false
func checkLocked(locked: bool) -> bool {
	if locked == true {
		return false
	}
	executedAfterReturn =: true
	return true
}

let res = checkLocked(true)
`
	l := lexer.New("test_ret.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	resVal, ok := env.Get("res")
	if !ok || resVal.(runtime.BoolValue).Val != false {
		t.Fatalf("Expected res = false, got %v", resVal)
	}
	afterVal, ok := env.Get("executedAfterReturn")
	if !ok || afterVal.(runtime.BoolValue).Val != false {
		t.Fatalf("Code after return was executed! Expected executedAfterReturn = false, got %v", afterVal)
	}
}

func TestFunctionSignatureValidation(t *testing.T) {
	// 1. Missing argument
	codeMissingArg := `
func greet(name: string) -> string {
	return "Hello " + name
}
greet()
`
	l := lexer.New("test_sig1.xru", codeMissingArg)
	p := parser.New(l)
	prog := p.ParseProgram()
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err == nil {
		t.Fatalf("Expected error when calling function with 0 args instead of 1, got nil")
	}

	// 2. Argument type mismatch
	codeTypeMismatch := `
func square(n: number) -> number {
	return n * n
}
square("not_a_number")
`
	l2 := lexer.New("test_sig2.xru", codeTypeMismatch)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected error when passing string to number param, got nil")
	}

	// 3. Return type mismatch
	codeRetMismatch := `
func getFlag() -> bool {
	return "invalid_string"
}
getFlag()
`
	l3 := lexer.New("test_sig3.xru", codeRetMismatch)
	p3 := parser.New(l3)
	prog3 := p3.ParseProgram()
	ev3 := runtime.NewEvaluator()
	env3 := runtime.NewEnvironment(nil)
	if _, err := ev3.Eval(prog3, env3); err == nil {
		t.Fatalf("Expected error when returning string for bool return type, got nil")
	}
}

func TestFsOperationsAndErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xru_fs_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "sample.txt")
	destPath := filepath.Join(tempDir, "sample_copy.txt")

	code := fmt.Sprintf(`
use fs as Fs

Fs.write("%s", "hello world")
let exists = Fs.exists("%s")
let isF = Fs.isFile("%s")
let isD = Fs.isDir("%s")
let content = Fs.read("%s")
let sz = Fs.size("%s")
Fs.copy("%s", "%s")
Fs.remove("%s")
Fs.remove("%s")
`, filePath, filePath, filePath, tempDir, filePath, filePath, filePath, destPath, filePath, destPath)

	l := lexer.New("test_fs.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error on valid Fs operations: %v", err)
	}

	cVal, _ := env.Get("content")
	if cVal.String() != "hello world" {
		t.Fatalf("Expected content 'hello world', got '%s'", cVal.String())
	}

	// Test that Fs.read on non-existent file returns OS error
	codeReadNonExistent := `
use fs as Fs
Fs.read("/path/does/not/exist/ever_12345.txt")
`
	l2 := lexer.New("test_fs_err.xru", codeReadNonExistent)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected OS error on Fs.read non-existent file, got nil")
	}

	// Test that Fs.remove on non-existent file returns OS error
	codeRemoveNonExistent := `
use fs as Fs
Fs.remove("/path/does/not/exist/ever_67890.txt")
`
	l3 := lexer.New("test_fs_err2.xru", codeRemoveNonExistent)
	p3 := parser.New(l3)
	prog3 := p3.ParseProgram()
	ev3 := runtime.NewEvaluator()
	env3 := runtime.NewEnvironment(nil)
	if _, err := ev3.Eval(prog3, env3); err == nil {
		t.Fatalf("Expected OS error on Fs.remove non-existent file, got nil")
	}
}

func TestUnitPropertyRestriction(t *testing.T) {
	// 1. &prop outside unit should fail
	code1 := `let a = &baseDir`
	l1 := lexer.New("test1.xru", code1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	ev1 := runtime.NewEvaluator()
	env1 := runtime.NewEnvironment(nil)
	if _, err := ev1.Eval(prog1, env1); err == nil {
		t.Fatalf("Expected error evaluating &baseDir outside unit, got nil")
	}

	// 2. {&prop} in string interpolation outside unit should fail
	code2 := `let a = "{&baseDir}/target"`
	l2 := lexer.New("test2.xru", code2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected error evaluating '{&baseDir}/target' outside unit, got nil")
	}
}

func TestCircularModuleImport(t *testing.T) {
	tempDir := t.TempDir()
	modCfg := `map {
	ns -> "./*"
}`
	_ = os.WriteFile(filepath.Join(tempDir, "_mod.xru"), []byte(modCfg), 0644)
	m1Code := `from "ns" use m2`
	m2Code := `from "ns" use m1`
	_ = os.WriteFile(filepath.Join(tempDir, "m1.xru"), []byte(m1Code), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "m2.xru"), []byte(m2Code), 0644)

	mainCode := `from "ns" use m1`
	l := lexer.New("main.xru", mainCode)
	p := parser.New(l)
	prog := p.ParseProgram()

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(tempDir)

	_, err := ev.Eval(prog, env)
	if err == nil {
		t.Fatalf("Expected error on circular module import, got nil")
	}
	if !strings.Contains(err.Error(), "circular module dependency") && !strings.Contains(err.Error(), "ERR_CIRCULAR_IMPORT") && !strings.Contains(err.Error(), "Circular module import") {
		t.Fatalf("Expected error message to contain circular import error, got: %v", err)
	}
}

func TestIfExpressionEvaluation(t *testing.T) {
	input := `
let isProd = true
let host = if isProd -> "https://api.xypriss.dev" else "http://localhost:3000"

let isDev = false
let port = if isDev -> 8080 else 3000

let score = 75
let grade = if score >= 90 -> "A" else if score >= 70 -> "B" else "C"
`
	l := lexer.New("inline_if_test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	_, err := ev.Eval(prog, env)
	if err != nil {
		t.Fatalf("Evaluation error: %v", err)
	}

	hostVal, ok := env.Get("host")
	if !ok || hostVal.String() != "https://api.xypriss.dev" {
		t.Errorf("Expected host to be 'https://api.xypriss.dev', got %v", hostVal)
	}

	portVal, ok := env.Get("port")
	if !ok || portVal.String() != "3000" {
		t.Errorf("Expected port to be 3000, got %v", portVal)
	}

	gradeVal, ok := env.Get("grade")
	if !ok || gradeVal.String() != "B" {
		t.Errorf("Expected grade to be 'B', got %v", gradeVal)
	}
}

func TestAsyncTaskWithExecution(t *testing.T) {
	code := `
func compute(n: number) -> number {
    return n * 2
}

let task = with compute(21)
let doneBefore = task.isDone()
let res = task.wait()
let doneAfter = task.isDone()
`
	l := lexer.New("async_test.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	resVal, ok := env.Get("res")
	if !ok || resVal.String() != "42" {
		t.Errorf("Expected task.wait() to return 42, got %v", resVal)
	}

	doneAfterVal, ok := env.Get("doneAfter")
	if !ok || doneAfterVal.String() != "true" {
		t.Errorf("Expected task.isDone() after wait to be true, got %v", doneAfterVal)
	}
}

func TestUnitInstanceMethodsAndSignatures(t *testing.T) {
	code := `
unit TemplateManager {
    declare baseDir: string = "./src"
    declare isLocked: bool = false

    func clean(target: string) -> bool {
        if &isLocked == true {
            return false
        }
        return true
    }

    func lock() -> void {
        &isLocked =: true
    }
}

let mgr = @TemplateManager()
let ok1 = mgr.clean("cache")
mgr.lock()
let ok2 = mgr.clean("cache")
`
	l := lexer.New("unit_methods_test.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	ok1Val, _ := env.Get("ok1")
	if ok1Val.String() != "true" {
		t.Errorf("Expected ok1 to be true, got %v", ok1Val)
	}

	ok2Val, _ := env.Get("ok2")
	if ok2Val.String() != "false" {
		t.Errorf("Expected ok2 to be false after lock(), got %v", ok2Val)
	}

	// Test: runtime mismatch on arguments
	badArgsCode := `
let mgr2 = @TemplateManager()
mgr2.clean()
`
	lBad := lexer.New("bad_args.xru", badArgsCode)
	pBad := parser.New(lBad)
	progBad := pBad.ParseProgram()
	_, errBad := ev.Eval(progBad, env)
	if errBad == nil || !strings.Contains(errBad.Error(), "expects 1 arguments, but got 0") {
		t.Errorf("Expected runtime error for wrong argument count, got: %v", errBad)
	}

	// Test: runtime mismatch on return type
	badRetCode := `
unit BadUnit {
    func broken() -> bool {
        return "invalid string"
    }
}
let b = @BadUnit()
b.broken()
`
	lRet := lexer.New("bad_ret.xru", badRetCode)
	pRet := parser.New(lRet)
	progRet := pRet.ParseProgram()
	_, errRet := ev.Eval(progRet, env)
	if errRet == nil || !strings.Contains(errRet.Error(), "returned value of type 'string', expected 'bool'") {
		t.Errorf("Expected runtime error for return type mismatch, got: %v", errRet)
	}
}




