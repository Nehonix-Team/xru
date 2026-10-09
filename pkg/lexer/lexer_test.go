package lexer_test

import (
	"testing"

	"github.com/Nehonix-Team/xru/pkg/lexer"
)

func TestLexerBasic(t *testing.T) {
	input := `
use net as Net
arg port: number = 3000
arg timeout: duration = 30s

let target: string = "./dist"
let manager = @TemplateManager()

unit TemplateManager {
    declare baseDir: string = "./src"
    func Clean(target: string) -> bool {
        Fs.Remove("{&baseDir}/{target}")
        return true
    }
}

with Net.Fetch("https://example.com") {
    header "Accept" -> "application/json"
    set Timeout = 30s
}
`
	l := lexer.New("test.xru", input)

	expectedTokens := []lexer.TokenType{
		lexer.USE, lexer.IDENT, lexer.AS, lexer.IDENT,
		lexer.ARG, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.NUMBER,
		lexer.ARG, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.DURATION,

		lexer.LET, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.STRING,
		lexer.LET, lexer.IDENT, lexer.ASSIGN, lexer.AT, lexer.IDENT, lexer.LPAREN, lexer.RPAREN,

		lexer.UNIT, lexer.IDENT, lexer.LBRACE,
		lexer.DECLARE, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.STRING,
		lexer.FUNC, lexer.IDENT, lexer.LPAREN, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.RPAREN, lexer.ARROW, lexer.IDENT, lexer.LBRACE,
		lexer.IDENT, lexer.DOT, lexer.IDENT, lexer.LPAREN, lexer.STRING, lexer.RPAREN,
		lexer.RETURN, lexer.BOOL,
		lexer.RBRACE,
		lexer.RBRACE,

		lexer.WITH, lexer.IDENT, lexer.DOT, lexer.IDENT, lexer.LPAREN, lexer.STRING, lexer.RPAREN, lexer.LBRACE,
		lexer.HEADER, lexer.STRING, lexer.ARROW, lexer.STRING,
		lexer.SET, lexer.IDENT, lexer.ASSIGN, lexer.DURATION,
		lexer.RBRACE,

		lexer.EOF,
	}

	for i, expected := range expectedTokens {
		tok := l.NextToken()
		if tok.Type != expected {
			t.Fatalf("[%d] Token mismatch at %s: expected %s, got %s ('%s')",
				i, tok.Pos, expected, tok.Type, tok.Literal)
		}
	}
}

func TestLexerMultiLineStrings(t *testing.T) {
	input := "let a = \"\"\"import { createServer } from \"xypriss\";\"\"\"\nlet b = `multi\nline`"
	l := lexer.New("test_multiline.xru", input)

	_ = l.NextToken() // let
	_ = l.NextToken() // a
	_ = l.NextToken() // =
	tok4 := l.NextToken() // """...""" string
	if tok4.Type != lexer.STRING || tok4.Literal != "import { createServer } from \"xypriss\";" {
		t.Errorf("Expected raw content with inner quotes, got: %s", tok4.Literal)
	}

	_ = l.NextToken() // let
	_ = l.NextToken() // b
	_ = l.NextToken() // =
	tok8 := l.NextToken() // `multi\nline` string
	if tok8.Type != lexer.STRING || tok8.Literal != "multi\nline" {
		t.Errorf("Expected raw backtick string, got: %s", tok8.Literal)
	}

	inputHeredoc := "let c = <<<import {\n  permanentCdnPath\n} from \"./cdn\";>>>"
	lH := lexer.New("test_heredoc.xru", inputHeredoc)
	_ = lH.NextToken() // let
	_ = lH.NextToken() // c
	_ = lH.NextToken() // =
	tokC := lH.NextToken()
	expectedC := "import {\n  permanentCdnPath\n} from \"./cdn\";"
	if tokC.Type != lexer.STRING || tokC.Literal != expectedC {
		t.Errorf("Expected <<< heredoc raw content, got: %s", tokC.Literal)
	}
}
