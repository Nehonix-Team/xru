package lexer

import (
	"fmt"
	"strings"
)

type TokenType string

const (
	// Special tokens
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"
	COMMENT TokenType = "COMMENT"

	// Identifiers & Literals
	IDENT    TokenType = "IDENT"    // ex: myVar, clean, Net
	STRING   TokenType = "STRING"   // ex: "hello", "https://..."
	NUMBER   TokenType = "NUMBER"   // ex: 3000, 42, 3.14
	BOOL     TokenType = "BOOL"     // true, false
	DURATION TokenType = "DURATION" // ex: 30s, 500ms, 2m, 1h

	// Modern clean language keywords (all lowercase)
	USE           TokenType = "use"
	ARG           TokenType = "arg"
	LET           TokenType = "let"
	CONST         TokenType = "const"
	UNIT          TokenType = "unit"
	DECLARE       TokenType = "declare"
	DIRECTIVE     TokenType = "directive"
	FUNC          TokenType = "func"
	RETURN        TokenType = "return"
	WITH          TokenType = "with"
	SET           TokenType = "set"
	HEADER        TokenType = "header"
	ON            TokenType = "on"
	IF            TokenType = "if"
	ELIF          TokenType = "elif"
	ELSE          TokenType = "else"
	MATCH         TokenType = "match"
	CASE          TokenType = "case"
	DEFAULT       TokenType = "default"
	DISPATCH      TokenType = "dispatch"
	DISPATCH_TREE TokenType = "dispatch_tree"
	FILE          TokenType = "file"
	GLOBAL        TokenType = "global"
	PASS          TokenType = "pass"
	AS            TokenType = "as"
	OR            TokenType = "or"
	MAP           TokenType = "map"
	FROM          TokenType = "from"
	MOD           TokenType = "mod"
	FOR           TokenType = "for"
	IN            TokenType = "in"
	WHILE         TokenType = "while"
	CATCH         TokenType = "catch"
	INTERFACE     TokenType = "interface"
	TYPE          TokenType = "type"
	INCLUDES      TokenType = "includes"
	PUBLIC        TokenType = "public"
	PRIVATE       TokenType = "private"

	// Operators & Symbols
	ASSIGN     TokenType = "="
	MUT_ASSIGN TokenType = "=:"
	ARROW      TokenType = "->"
	FAT_ARROW TokenType = "=>"
	AT        TokenType = "@" // Instanciation @Unit()
	AMP       TokenType = "&" // Propriété interne &prop
	PLUS      TokenType = "+"
	MINUS     TokenType = "-"
	ASTERISK  TokenType = "*"
	SLASH     TokenType = "/"
	PERCENT   TokenType = "%"
	RANGE     TokenType = ".."
	ELLIPSIS  TokenType = "..."

	// Comparaisons
	EQ         TokenType = "=="
	NEQ        TokenType = "!="
	LT         TokenType = "<"
	LTE        TokenType = "<="
	GT         TokenType = ">"
	GTE        TokenType = ">="
	BANG       TokenType = "!"
	AND        TokenType = "&&"
	OR_LOGICAL TokenType = "||"
	PIPE       TokenType = "|"

	// Delimiters
	LPAREN       TokenType = "("
	RPAREN       TokenType = ")"
	LBRACKET     TokenType = "["
	RBRACKET     TokenType = "]"
	LBRACE       TokenType = "{"
	RBRACE       TokenType = "}"
	COMMA        TokenType = ","
	COLON        TokenType = ":"
	DOUBLE_COLON TokenType = "::"
	DOT          TokenType = "."
	SEMICOLON    TokenType = ";"
	QUESTION     TokenType = "?"
)

type Position struct {
	Filename string
	Line     int
	Column   int
}

func (p Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Filename, p.Line, p.Column)
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

func (t Token) String() string {
	return fmt.Sprintf("[%s '%s' at %s]", t.Type, t.Literal, t.Pos)
}

var keywords = map[string]TokenType{
	"let":           LET,
	"const":         CONST,
	"as":            AS,
	"or":            OR,
	"true":          BOOL,
	"false":         BOOL,
	"use":           USE,
	"arg":           ARG,
	"unit":          UNIT,
	"declare":       DECLARE,
	"directive":     DIRECTIVE,
	"func":          FUNC,
	"return":        RETURN,
	"with":          WITH,
	"set":           SET,
	"header":        HEADER,
	"on":            ON,
	"if":            IF,
	"elif":          ELIF,
	"else":          ELSE,
	"match":         MATCH,
	"case":          CASE,
	"default":       DEFAULT,
	"dispatch":      DISPATCH,
	"dispatch_tree": DISPATCH_TREE,
	"file":          FILE,
	"global":        GLOBAL,
	"export":        GLOBAL,
	"pass":          PASS,
	"map":           MAP,
	"from":          FROM,
	"mod":           MOD,
	"for":           FOR,
	"in":            IN,
	"while":         WHILE,
	"catch":         CATCH,
	"interface":     INTERFACE,
	"type":          TYPE,
	"includes":      INCLUDES,
	"public":        PUBLIC,
	"private":       PRIVATE,
}

func LookupIdent(ident string) TokenType {
	lower := strings.ToLower(ident)
	if tok, ok := keywords[lower]; ok {
		return tok
	}
	return IDENT
}


func IsKeyword(tok TokenType) bool {
	for _, kw := range keywords {
		if kw == tok {
			return true
		}
	}
	return false
}
