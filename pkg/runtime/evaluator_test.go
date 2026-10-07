package runtime_test

import (
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
