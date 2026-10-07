package parser_test

import (
	"testing"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func TestParserComplete(t *testing.T) {
	input := `
#USE net as Net
#ARG PORT: number or 3000
#ARG TIMEOUT: duration or 30s

let targetDir: string = "./my-project"
let isProd: bool = true
let manager: TemplateManager = @TemplateManager()

#UNIT TemplateManager
    #DECLARE baseDir: string = "./src"
    
    #DIRECTIVE OPTION(key: string, value: any)
        Utils.Log("Option set")
    #END

    #FUNC Clean(target: string) -> bool
        Fs.Remove("{&baseDir}/{target}")
        #RETURN true
    #END
#END

#WITH Net.Fetch("https://api.github.com/releases")
    #SET Method = "GET"
    #HEADER "Accept" -> "application/json"
#END

#IF {PORT} == 3000
    Utils.Log("Default port used")
#ELIF {PORT} > 8000
    Utils.Log("High port")
#ELSE
    Utils.Log("Custom port")
#END

#MATCH {PORT}
    #CASE 3000
        let pType = "standard"
    #DEFAULT
        let pType = "other"
#END

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

	// Vérification de la présence des déclarations clés
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
