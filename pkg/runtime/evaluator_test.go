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

std.println("<green>[TEST]</> Orchestration $f{app_name} v$f{version}")
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

let MODE = "minimal"
let CDN = false

dispatch "./src", ".rules.xru"
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
    let msg = "Hello $f{name}"
    return msg
}

func LocalGreet(name: string) {
    let msg = "Hi $f{name}"
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

	// 2. $f{&prop} in string interpolation outside unit should fail
	code2 := `let a = "$f{&baseDir}/target"`
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

func TestForLoopAndRange(t *testing.T) {
	code := `
let sum = 0
for i in 1..5 {
    sum =: sum + i
}

let items = ["a", "b", "c"]
let collected = ""
for item, idx in items {
    collected =: collected + item + idx
}
`
	l := lexer.New("for_test.xru", code)
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

	sumVal, _ := env.Get("sum")
	if sumVal.String() != "15" {
		t.Errorf("Expected sum to be 15, got %v", sumVal)
	}

	collVal, _ := env.Get("collected")
	if collVal.String() != "a0b1c2" {
		t.Errorf("Expected collected to be 'a0b1c2', got %v", collVal)
	}
}

func TestWithExpressionAndTask(t *testing.T) {
	code := `
unit Worker {
    func compute() -> number {
        let sum = 0
        for i in 1..10 {
            sum =: sum + i
        }
        return sum
    }
}

let w = @Worker()
let task = with w.compute()
let doneBefore = task.isDone()
let res = task.wait()
let doneAfter = task.isDone()
`
	l := lexer.New("with_test.xru", code)
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

	doneAfterVal, _ := env.Get("doneAfter")
	if doneAfterVal.String() != "true" {
		t.Errorf("Expected doneAfter to be 'true', got %v", doneAfterVal)
	}

	resVal, _ := env.Get("res")
	if resVal.String() != "55" {
		t.Errorf("Expected res to be '55', got %v", resVal)
	}
}

func TestCatchExpression(t *testing.T) {
	code := `
use fs as Fs

// 1. Division by zero trapped with fallback
let safeVal = (10 / 0) catch err or 42
let hadErr1 = (err != nil)

// 2. Successful call: err is nil
let okVal = (10 / 2) catch err2 or 99
let hadErr2 = (err2 != nil)

// 3. Direct action catch (void): removing nonexistent file fails but is trapped
Fs.remove("/nonexistent_path_test_1234") catch err3
let hadErr3 = (err3 != nil)

// 4. Guard / early exit inside function: catch err or return <val>
func testGuard(shouldFail: bool) -> string {
    if shouldFail {
        (10 / 0) catch e or return "failed_cleanly"
    }
    return "completed"
}

let guard1 = testGuard(true)
let guard2 = testGuard(false)
`
	l := lexer.New("catch_test.xru", code)
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

	safeVal, _ := env.Get("safeVal")
	if safeVal.String() != "42" {
		t.Errorf("Expected safeVal to be '42', got %v", safeVal)
	}

	hadErr1, _ := env.Get("hadErr1")
	if hadErr1.String() != "true" {
		t.Errorf("Expected hadErr1 to be true, got %v", hadErr1)
	}

	okVal, _ := env.Get("okVal")
	if okVal.String() != "5" {
		t.Errorf("Expected okVal to be '5', got %v", okVal)
	}

	hadErr2, _ := env.Get("hadErr2")
	if hadErr2.String() != "false" {
		t.Errorf("Expected hadErr2 to be false, got %v", hadErr2)
	}

	hadErr3, _ := env.Get("hadErr3")
	if hadErr3.String() != "true" {
		t.Errorf("Expected hadErr3 to be true, got %v", hadErr3)
	}

	guard1, _ := env.Get("guard1")
	if guard1.String() != "failed_cleanly" {
		t.Errorf("Expected guard1 to be 'failed_cleanly', got %v", guard1)
	}

	guard2, _ := env.Get("guard2")
	if guard2.String() != "completed" {
		t.Errorf("Expected guard2 to be 'completed', got %v", guard2)
	}
}

func TestDynamicStringInterpolation(t *testing.T) {
	code := `
use crypto as Crypto

unit UserHelper {
    declare prefix: string = "USR"
    func tag(id: number) -> string {
        return "$f{&prefix}_$f{id * 2}"
    }
}

let h = @UserHelper()
let res1 = "Result: $f{10 + 25}"
let res2 = "ID: $f{h.tag(5)}"
let res3 = "Hex: $f{Crypto.GetRandomBytes(4)}"
let codeLiteral = "import { UserService } from '${apiHost}/users'"
`
	l := lexer.New("interp_test.xru", code)
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

	res1Val, _ := env.Get("res1")
	if res1Val.String() != "Result: 35" {
		t.Errorf("Expected 'Result: 35', got %v", res1Val)
	}

	res2Val, _ := env.Get("res2")
	if res2Val.String() != "ID: USR_10" {
		t.Errorf("Expected 'ID: USR_10', got %v", res2Val)
	}

	res3Val, _ := env.Get("res3")
	if !strings.HasPrefix(res3Val.String(), "Hex: ") || len(res3Val.String()) != 13 {
		t.Errorf("Expected 'Hex: <8 chars hex>', got %v", res3Val)
	}

	litVal, _ := env.Get("codeLiteral")
	if litVal.String() != "import { UserService } from '${apiHost}/users'" {
		t.Errorf("Expected code literal to remain unmolested, got %v", litVal)
	}
}

func TestUnitInstantiationRules(t *testing.T) {
	// 1. Built-in module cannot be instantiated with @
	code1 := `
use crypto as Crypto
let c = @Crypto()
`
	l1 := lexer.New("test_builtin_inst.xru", code1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	ev1 := runtime.NewEvaluator()
	env1 := runtime.NewEnvironment(nil)
	if _, err := ev1.Eval(prog1, env1); err == nil {
		t.Fatalf("Expected error when instantiating built-in Crypto with @, got nil")
	} else if !strings.Contains(err.Error(), "ERR_CANNOT_INSTANTIATE_BUILTIN") && !strings.Contains(err.Error(), "cannot be instantiated") {
		t.Fatalf("Expected ERR_CANNOT_INSTANTIATE_BUILTIN error message, got: %v", err)
	}

	// 2. User unit cannot call instance methods without @
	code2 := `
unit Service {
    func action() -> string {
        return "done"
    }
}
let res = Service.action()
`
	l2 := lexer.New("test_static_call.xru", code2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err == nil {
		t.Fatalf("Expected error when calling method on uninstantiated unit, got nil")
	} else if !strings.Contains(err.Error(), "cannot call method") {
		t.Fatalf("Expected cannot call method error message, got: %v", err)
	}
}

func TestMatchStatementEvaluation(t *testing.T) {
	code := `
let target = "beta"
let res = "none"
match target {
    case "alpha" {
        res =: "first"
    }
    case "beta" {
        res =: "second"
    }
    default {
        res =: "other"
    }
}

let fallback = "default"
match "unknown" {
    case "alpha" {
        fallback =: "a"
    }
    default {
        fallback =: "def"
    }
}
`
	l := lexer.New("test_match.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	resVal, _ := env.Get("res")
	if resVal.String() != "second" {
		t.Errorf("Expected res == 'second', got %v", resVal)
	}

	fbVal, _ := env.Get("fallback")
	if fbVal.String() != "def" {
		t.Errorf("Expected fallback == 'def', got %v", fbVal)
	}
}

func TestStringNativeMethods(t *testing.T) {
	code := `
let str = "  Hello World  "
let trimmed = str.trim()
let upper = trimmed.toUpper()
let lower = trimmed.toLower()
let l = str.len()
let has = trimmed.contains("World")
let starts = trimmed.startsWith("Hello")
let ends = trimmed.endsWith("World")
let replaced = trimmed.replace("World", "XRU")
let sliced = trimmed.slice(0, 5)
let parts = trimmed.split(" ")
let countParts = parts.len()
`
	l := lexer.New("test_str_methods.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	trimmed, _ := env.Get("trimmed")
	if trimmed.String() != "Hello World" {
		t.Errorf("Expected trimmed to be 'Hello World', got %v", trimmed)
	}
	upper, _ := env.Get("upper")
	if upper.String() != "HELLO WORLD" {
		t.Errorf("Expected upper to be 'HELLO WORLD', got %v", upper)
	}
	lower, _ := env.Get("lower")
	if lower.String() != "hello world" {
		t.Errorf("Expected lower to be 'hello world', got %v", lower)
	}
	lVal, _ := env.Get("l")
	if lVal.String() != "15" {
		t.Errorf("Expected len == 15, got %v", lVal)
	}
	hasVal, _ := env.Get("has")
	if hasVal.String() != "true" {
		t.Errorf("Expected contains == true, got %v", hasVal)
	}
	startsVal, _ := env.Get("starts")
	if startsVal.String() != "true" {
		t.Errorf("Expected startsWith == true, got %v", startsVal)
	}
	endsVal, _ := env.Get("ends")
	if endsVal.String() != "true" {
		t.Errorf("Expected endsWith == true, got %v", endsVal)
	}
	repVal, _ := env.Get("replaced")
	if repVal.String() != "Hello XRU" {
		t.Errorf("Expected replaced == 'Hello XRU', got %v", repVal)
	}
	sliceVal, _ := env.Get("sliced")
	if sliceVal.String() != "Hello" {
		t.Errorf("Expected sliced == 'Hello', got %v", sliceVal)
	}
	countVal, _ := env.Get("countParts")
	if countVal.String() != "2" {
		t.Errorf("Expected countParts == 2, got %v", countVal)
	}
}

func TestListNativeMethods(t *testing.T) {
	code := `
let items = ["alpha", "beta", "gamma"]
let joined = items.join("-")
let hasBeta = items.contains("beta")
let count = items.len()
items.push("delta")
let countAfterPush = items.len()
let popped = items.pop()
let reversed = items.reverse()
`
	l := lexer.New("test_list_methods.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	joined, _ := env.Get("joined")
	if joined.String() != "alpha-beta-gamma" {
		t.Errorf("Expected joined == 'alpha-beta-gamma', got %v", joined)
	}
	hasBeta, _ := env.Get("hasBeta")
	if hasBeta.String() != "true" {
		t.Errorf("Expected hasBeta == true, got %v", hasBeta)
	}
	count, _ := env.Get("count")
	if count.String() != "3" {
		t.Errorf("Expected count == 3, got %v", count)
	}
	capVal, _ := env.Get("countAfterPush")
	if capVal.String() != "4" {
		t.Errorf("Expected countAfterPush == 4, got %v", capVal)
	}
	popVal, _ := env.Get("popped")
	if popVal.String() != "delta" {
		t.Errorf("Expected popped == 'delta', got %v", popVal)
	}
	revVal, _ := env.Get("reversed")
	if revVal.String() != "[gamma, beta, alpha]" {
		t.Errorf("Expected reversed == [gamma, beta, alpha], got %v", revVal)
	}
}

func TestBuiltinSingletons(t *testing.T) {
	code := `
let p = Path.Join("src", "services", "auth.xru")
let d = Path.Dir(p)
let b = Path.Base(p)
let e = Path.Ext(p)

let r = Math.Round(3.7)
let f = Math.Floor(3.7)
let c = Math.Ceil(3.2)
let a = Math.Abs(-42)
let m = Math.Min(10, 5)

let data = ["name" -> "xypriss", "version" -> 2]
let stringified = Json.Stringify(data)
let parsed = Json.Parse(stringified)
let backToStr = Json.Stringify(parsed)
`
	l := lexer.New("test_singletons.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	dVal, _ := env.Get("d")
	if dVal.String() != "src/services" {
		t.Errorf("Expected dir == 'src/services', got %v", dVal)
	}
	bVal, _ := env.Get("b")
	if bVal.String() != "auth.xru" {
		t.Errorf("Expected base == 'auth.xru', got %v", bVal)
	}
	eVal, _ := env.Get("e")
	if eVal.String() != ".xru" {
		t.Errorf("Expected ext == '.xru', got %v", eVal)
	}
	rVal, _ := env.Get("r")
	if rVal.String() != "4" {
		t.Errorf("Expected round == 4, got %v", rVal)
	}
	fVal, _ := env.Get("f")
	if fVal.String() != "3" {
		t.Errorf("Expected floor == 3, got %v", fVal)
	}
	cVal, _ := env.Get("c")
	if cVal.String() != "4" {
		t.Errorf("Expected ceil == 4, got %v", cVal)
	}
	aVal, _ := env.Get("a")
	if aVal.String() != "42" {
		t.Errorf("Expected abs == 42, got %v", aVal)
	}
	mVal, _ := env.Get("m")
	if mVal.String() != "5" {
		t.Errorf("Expected min == 5, got %v", mVal)
	}
	strVal, _ := env.Get("stringified")
	if !strings.Contains(strVal.String(), "xypriss") {
		t.Errorf("Expected json stringified to contain 'xypriss', got %v", strVal)
	}
}

func TestModernDispatchSyntax(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	_ = os.MkdirAll(subDir, 0755)

	ruleContent := `
let executed = true
`
	_ = os.WriteFile(filepath.Join(subDir, ".rules.xru"), []byte(ruleContent), 0644)

	code := fmt.Sprintf(`dispatch "%s"`, subDir)
	l := lexer.New("test_dispatch.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Dispatch eval error: %v", err)
	}
}

func TestDispatchDynamicPath(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	_ = os.MkdirAll(subDir, 0755)

	ruleContent := `let done = true`
	_ = os.WriteFile(filepath.Join(subDir, "patch.xru"), []byte(ruleContent), 0644)

	// 1. With variable: dispatch targetPath, "patch.xru"
	code1 := fmt.Sprintf(`
let targetPath = "%s"
dispatch targetPath, "patch.xru"
`, subDir)

	l1 := lexer.New("test_disp_var.xru", code1)
	p1 := parser.New(l1)
	prog1 := p1.ParseProgram()
	if len(p1.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p1.Errors())
	}
	ev1 := runtime.NewEvaluator()
	env1 := runtime.NewEnvironment(nil)
	if _, err := ev1.Eval(prog1, env1); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	// 2. With string interpolation: dispatch "{baseDir}/sub", "patch.xru"
	code2 := fmt.Sprintf(`
let baseDir = "%s"
dispatch "{baseDir}/sub", "patch.xru"
`, tempDir)

	l2 := lexer.New("test_disp_interp.xru", code2)
	p2 := parser.New(l2)
	prog2 := p2.ParseProgram()
	if len(p2.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p2.Errors())
	}
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	if _, err := ev2.Eval(prog2, env2); err != nil {
		t.Fatalf("Eval error: %v", err)
	}
}

func TestModernMultiViewFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	samplePath := filepath.Join(tmpDir, "sample.txt")
	sampleContent := `import { PasswordManager } from "xypriss-security";

export const pwd = new PasswordManager({
  algorithm: "argon2id",
  memoryCost: 65536,
  parallelism: 4,
  nested: {
    enabled: true
  }
});

let remaining = "keep this";`

	if err := os.WriteFile(samplePath, []byte(sampleContent), 0644); err != nil {
		t.Fatalf("Failed to write sample: %v", err)
	}

	code := fmt.Sprintf(`
file "%s", (doc, err) -> {
    // 1. Suppression du bloc équilibré d'accolades complet
    doc.removeBalanced("export const pwd = new PasswordManager", "{", "}")

    // 2. Suppression de la ligne d'import via removeBetween
    doc.removeBetween("import", "from \"xypriss-security\";", true)

    // 3. Ajout d'une ligne
    doc.append("\n// ADDED")
}
`, samplePath)

	l := lexer.New("modern_file.xru", code)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("Failed to read sample: %v", err)
	}
	res := string(data)

	if strings.Contains(res, "PasswordManager") {
		t.Errorf("Expected PasswordManager to be completely removed, got: %s", res)
	}
	if strings.Contains(res, "memoryCost") {
		t.Errorf("Expected balanced block (memoryCost) to be removed, got: %s", res)
	}
	if !strings.Contains(res, "keep this") {
		t.Errorf("Expected 'keep this' to remain, got: %s", res)
	}
	if !strings.Contains(res, "// ADDED") {
		t.Errorf("Expected '// ADDED' to be appended, got: %s", res)
	}
}

func TestInStatementEvaluation(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	keepFile := filepath.Join(subDir, "keep.ts")
	dropFile := filepath.Join(subDir, "drop.ts")
	_ = os.WriteFile(keepFile, []byte("const hello = 'foo';"), 0644)
	_ = os.WriteFile(dropFile, []byte("should be deleted"), 0644)

	code := fmt.Sprintf(`
in "%s" {
    keep(["keep.ts"])
    file "keep.ts", (doc) -> {
        doc.replace("foo", "bar")
    }
}
`, subDir)

	l := lexer.New("test_in.xru", code)
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

	// Verify keep.ts content was modified
	content, err := os.ReadFile(keepFile)
	if err != nil {
		t.Fatalf("Failed to read keep.ts: %v", err)
	}
	if !strings.Contains(string(content), "bar") {
		t.Errorf("Expected keep.ts to contain 'bar', got: %s", string(content))
	}

	// Verify drop.ts was removed by keep([...])
	if _, err := os.Stat(dropFile); !os.IsNotExist(err) {
		t.Errorf("Expected drop.ts to be deleted by keep")
	}
}

func TestCmdExecWorkingDir(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "marker.txt")
	if err := os.WriteFile(testFile, []byte("hello-xru"), 0644); err != nil {
		t.Fatalf("Failed to write marker file: %v", err)
	}

	input := `
use cmd as Cmd
let out = Cmd.exec("cat marker.txt")
`
	l := lexer.New("test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(tempDir)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	val, ok := env.Get("out")
	if !ok {
		t.Fatalf("Expected variable 'out' to be set")
	}
	if !strings.Contains(val.String(), "hello-xru") {
		t.Errorf("Expected out to contain 'hello-xru', got: %q", val.String())
	}
}

func TestImportExportArg(t *testing.T) {
	tempDir := t.TempDir()
	modFile := filepath.Join(tempDir, "_mod.xru")
	modContent := `map {
    root -> "./"
}`
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		t.Fatalf("Failed to write _mod.xru: %v", err)
	}

	mainFile := filepath.Join(tempDir, "main.xru")
	mainContent := `mod main
global arg name: string = "cool-app"
export arg port: number = 3000
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.xru: %v", err)
	}

	subFile := filepath.Join(tempDir, "sub.xru")
	subContent := `
from "root" use main::name
from "root" use main::port as myPort
from "root" use main as m
let checkName = name
let checkPort = myPort
let checkMember = m.name
`
	if err := os.WriteFile(subFile, []byte(subContent), 0644); err != nil {
		t.Fatalf("Failed to write sub.xru: %v", err)
	}

	l := lexer.New("sub.xru", subContent)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}

	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(tempDir)

	if _, err := ev.Eval(prog, env); err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	checkName, ok := env.Get("checkName")
	if !ok || checkName.String() != "cool-app" {
		t.Errorf("Expected checkName to be 'cool-app', got: %v", checkName)
	}

	checkPort, ok := env.Get("checkPort")
	if !ok || checkPort.String() != "3000" {
		t.Errorf("Expected checkPort to be '3000', got: %v", checkPort)
	}

	checkMember, ok := env.Get("checkMember")
	if !ok || checkMember.String() != "cool-app" {
		t.Errorf("Expected checkMember to be 'cool-app', got: %v", checkMember)
	}
}

func TestStrictRuntimeTypeValidationAndOptional(t *testing.T) {
	// 1. Missing required field in map passed to function
	codeMissingField := `
type TExample = @map{
    a: string,
    c: number,
}
func run(params: TExample) {
    let x = params.a
}
run(@map{ a: "only_a" })
`
	p1 := parser.New(lexer.New("t1.xru", codeMissingField))
	prog1 := p1.ParseProgram()
	ev1 := runtime.NewEvaluator()
	env1 := runtime.NewEnvironment(nil)
	_, err1 := ev1.Eval(prog1, env1)
	if err1 == nil {
		t.Fatal("Expected runtime error for missing field 'c' in TExample map argument, got nil")
	}

	// 2. Non-existent property access on typed parameter
	codeNonExistentProp := `
type TExample = @map{
    a: string,
    c: number,
}
func run(params: TExample) {
    let x = params.notExist
}
run(@map{ a: "hello", c: 42 })
`
	p2 := parser.New(lexer.New("t2.xru", codeNonExistentProp))
	prog2 := p2.ParseProgram()
	ev2 := runtime.NewEvaluator()
	env2 := runtime.NewEnvironment(nil)
	_, err2 := ev2.Eval(prog2, env2)
	if err2 == nil {
		t.Fatal("Expected runtime error accessing 'params.notExist' on TExample, got nil")
	}
	if !strings.Contains(err2.Error(), "does not exist on type") {
		t.Errorf("Expected 'does not exist on type' error message, got: %v", err2)
	}

	// 3. Return type validation
	codeBadReturn := `
type TExample = @map{
    c: number,
}
func getC(): TExample["c"] {
    return "string instead of number"
}
getC()
`
	p3 := parser.New(lexer.New("t3.xru", codeBadReturn))
	prog3 := p3.ParseProgram()
	ev3 := runtime.NewEvaluator()
	env3 := runtime.NewEnvironment(nil)
	_, err3 := ev3.Eval(prog3, env3)
	if err3 == nil {
		t.Fatal("Expected runtime error for invalid return type, got nil")
	}

	// 4. Optional parameters
	codeOpt := `
let res1 = ""
let res2 = ""
let res3 = ""
func greet(name: string, other: string?, opt2?: number) {
    res1 =: name
    if other != nil {
        res2 =: other
    }
}
greet("Alice")
greet("Bob", "info")
`
	p4 := parser.New(lexer.New("t4.xru", codeOpt))
	prog4 := p4.ParseProgram()
	if len(p4.Errors()) > 0 {
		t.Fatalf("Parser errors in t4: %v", p4.Errors())
	}
	ev4 := runtime.NewEvaluator()
	env4 := runtime.NewEnvironment(nil)
	_, err4 := ev4.Eval(prog4, env4)
	if err4 != nil {
		t.Fatalf("Unexpected runtime error for optional params: %v", err4)
	}
	v1, _ := env4.Get("res1")
	v2, _ := env4.Get("res2")
	if v1.String() != "Bob" {
		t.Errorf("Expected res1 to be 'Bob', got: %v", v1)
	}
	if v2.String() != "info" {
		t.Errorf("Expected res2 to be 'info', got: %v", v2)
	}
}

func TestDateBuiltinRuntime(t *testing.T) {
	code := `
use date
let now = date.now()
let year = now.year
let isoStr = date.iso(now)
let formatted = date.format(now, "YYYY-MM-DD")
let parsed = date.parse("2026-10-15", "YYYY-MM-DD")
let parsedYear = parsed.year
let added = date.add(parsed, 10, "days")
let addedDay = added.day
let diffDays = date.diff(added, parsed, "days")
`
	p := parser.New(lexer.New("date_rt.xru", code))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	_, err := ev.Eval(prog, env)
	if err != nil {
		t.Fatalf("Runtime error evaluating date: %v", err)
	}

	py, _ := env.Get("parsedYear")
	if py.String() != "2026" {
		t.Errorf("Expected parsedYear to be 2026, got: %v", py)
	}

	ad, _ := env.Get("addedDay")
	if ad.String() != "25" {
		t.Errorf("Expected addedDay to be 25, got: %v", ad)
	}

	diff, _ := env.Get("diffDays")
	if diff.String() != "10" {
		t.Errorf("Expected diffDays to be 10, got: %v", diff)
	}
}

func TestVariadicParametersAndNativeHooks(t *testing.T) {
	code := `
func joinLines(...lines: string) -> string {
	return lines.join(" - ")
}

unit CustomHelper {
	@native("crypto_sha256")
	func hash(s: string) -> string

	func multiHash(...items: string) -> string {
		let combined = items.join(":")
		return &this.hash(combined)
	}
}

let res1 = joinLines("alpha", "beta", "gamma")
let resEmpty = joinLines()

@native("crypto_sha256")
func rawSha(val: string) -> string

let directHash = rawSha("hello")
`
	p := parser.New(lexer.New("variadic_test.xru", code))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	_, err := ev.Eval(prog, env)
	if err != nil {
		t.Fatalf("Runtime error: %v", err)
	}

	r1, _ := env.Get("res1")
	if r1.String() != "alpha - beta - gamma" {
		t.Errorf("Expected 'alpha - beta - gamma', got: %v", r1)
	}

	re, _ := env.Get("resEmpty")
	if re.String() != "" {
		t.Errorf("Expected empty string, got: %v", re)
	}

	dh, _ := env.Get("directHash")
	if dh.String() != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("Expected SHA256 of hello, got: %v", dh)
	}
}

func TestEvalUnitSelfScopeAccess(t *testing.T) {
	code := `
unit Calc {
    declare base = 10

    func add(x: number) -> number {
        return &base + x
    }

    func compute(a: number, b: number) -> number {
        let r1 = &Calc::add(a)
        let r2 = &self::add(b)
        return r1 + r2
    }
}

let c = @Calc()
let total = c.compute(5, 7)
`
	p := parser.New(lexer.New("self_test.xru", code))
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser error: %v", p.Errors())
	}
	ev := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)
	_, err := ev.Eval(prog, env)
	if err != nil {
		t.Fatalf("Runtime error: %v", err)
	}

	totalVal, ok := env.Get("total")
	if !ok {
		t.Fatalf("Variable 'total' not found")
	}
	// r1 = 10 + 5 = 15; r2 = 10 + 7 = 17; total = 32
	if totalVal.String() != "32" {
		t.Errorf("Expected 32, got %v", totalVal)
	}
}



