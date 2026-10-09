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

with Net.Fetch("https://api.github.com/releases") {
    set Method = "GET"
    header "Accept" -> "application/json"
}

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
		switch stmt.(type) {
		case *ast.UnitStatement:
			hasUnit = true
		case *ast.WithStatement:
			hasWith = true
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

func TestWithLambdaAndBlockClosure(t *testing.T) {
	input := `use utils as Utils
use net as Net

let a: string = "testing with block"
Utils.Log(a)

with Net.Fetch("https://api.github.com/repos/Nehonix-Team/xypriss-templates/releases/latest") {
    set Method = "GET"
    set Timeout = 30s
    set ExtractTo = "./src"

    header "Accept" -> "application/vnd.github.v3+json"
    header "User-Agent" -> "XRU-Engine"

    on Progress -> (percent, speed) => Utils.Log("<cyan>[PROGRESS]</> {percent}% à {speed}")
    on Complete -> (data, err) => {
        if err != nil {
            Utils.Log("Failed")
        } else {
            Utils.Log("<green>[DONE]</> Téléchargement terminé.")
        }
    }
}
`
	l := lexer.New("test.xru", input)
	p := parser.New(l)
	prog := p.ParseProgram()
	for _, err := range p.Errors() {
		t.Errorf("Error: %s", err)
	}
	if len(prog.Statements) == 0 {
		t.Fatal("No statements")
	}
}

func TestUnclosedBlock(t *testing.T) {
	input := `
with Net.Fetch("https://example.com") {
    set Method = "GET"
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
		t.Errorf("Expected 'expected token '}' error, got: %v", p.Errors())
	}
}
