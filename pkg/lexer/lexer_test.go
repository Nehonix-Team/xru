package lexer_test

import (
	"testing"

	"github.com/Nehonix-Team/xru/pkg/lexer"
)

func TestLexerBasic(t *testing.T) {
	input := `
#USE net as Net
#ARG PORT: number or 3000
#ARG TIMEOUT: duration or 30s

let target: string = "./dist"
let manager = @TemplateManager()

#UNIT TemplateManager
    #DECLARE baseDir: string = "./src"
    #FUNC Clean(target: string) -> bool
        Fs.Remove("{&baseDir}/{target}")
        #RETURN true
    #END
#END

#WITH Net.Fetch("https://example.com")
    #HEADER "Accept" -> "application/json"
    #SET Timeout = 30s
#END
`
	l := lexer.New("test.xru", input)

	expectedTokens := []lexer.TokenType{
		lexer.USE_DIR, lexer.IDENT, lexer.AS, lexer.IDENT,
		lexer.ARG_DIR, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.OR, lexer.NUMBER,
		lexer.ARG_DIR, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.OR, lexer.DURATION,

		lexer.LET, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.STRING,
		lexer.LET, lexer.IDENT, lexer.ASSIGN, lexer.AT, lexer.IDENT, lexer.LPAREN, lexer.RPAREN,

		lexer.UNIT_DIR, lexer.IDENT,
		lexer.DECLARE_DIR, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.ASSIGN, lexer.STRING,
		lexer.FUNC_DIR, lexer.IDENT, lexer.LPAREN, lexer.IDENT, lexer.COLON, lexer.IDENT, lexer.RPAREN, lexer.ARROW, lexer.IDENT,
		lexer.IDENT, lexer.DOT, lexer.IDENT, lexer.LPAREN, lexer.STRING, lexer.RPAREN,
		lexer.RETURN_DIR, lexer.BOOL,
		lexer.END_DIR,
		lexer.END_DIR,

		lexer.WITH_DIR, lexer.IDENT, lexer.DOT, lexer.IDENT, lexer.LPAREN, lexer.STRING, lexer.RPAREN,
		lexer.CUSTOM_DIR, lexer.STRING, lexer.ARROW, lexer.STRING,
		lexer.CUSTOM_DIR, lexer.IDENT, lexer.ASSIGN, lexer.DURATION,
		lexer.END_DIR,

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
