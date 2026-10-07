package runtime_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func TestEvaluatorBasic(t *testing.T) {
	code := `
#USE utils as Utils
#USE fs as Fs

#ARG APP_NAME or "demo-app"
let version = "1.0.0"

Utils.Log("<green>[TEST]</> Orchestration {APP_NAME} v{version}")
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

	appVal, ok := env.Get("APP_NAME")
	if !ok || appVal.String() != "demo-app" {
		t.Errorf("Expected APP_NAME to be 'demo-app', got '%v'", appVal)
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
#FILE "` + testFilePath + `"
    Replace("line 1", "first line")
    RemoveBlock("// BEGIN BLOCK", "// END BLOCK")
    RemoveLines("cdn")
#END
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

