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
	IDENT    TokenType = "IDENT"    // ex: myVar, Clean, Net
	STRING   TokenType = "STRING"   // ex: "hello", "https://..."
	NUMBER   TokenType = "NUMBER"   // ex: 3000, 42, 3.14
	BOOL     TokenType = "BOOL"     // true, false
	DURATION TokenType = "DURATION" // ex: 30s, 500ms, 2m, 1h

	// Directives starting with '#'
	USE_DIR       TokenType = "#USE"
	ARG_DIR       TokenType = "#ARG"
	UNIT_DIR      TokenType = "#UNIT"
	DECLARE_DIR   TokenType = "#DECLARE"
	DIRECTIVE_DIR TokenType = "#DIRECTIVE"
	FUNC_DIR      TokenType = "#FUNC"
	RETURN_DIR    TokenType = "#RETURN"
	WITH_DIR      TokenType = "#WITH"
	END_DIR       TokenType = "#END"
	IF_DIR        TokenType = "#IF"
	ELIF_DIR      TokenType = "#ELIF"
	ELSE_DIR      TokenType = "#ELSE"
	MATCH_DIR     TokenType = "#MATCH"
	CASE_DIR      TokenType = "#CASE"
	DEFAULT_DIR   TokenType = "#DEFAULT"
	DISPATCH_TREE TokenType = "#DISPATCH_TREE"
	FILE_DIR      TokenType = "#FILE"
	SCOPE_DIR     TokenType = "#SCOPE"
	GLOBAL_DIR    TokenType = "#GLOBAL"
	PASS_DIR      TokenType = "#PASS"

	// Keywords
	LET   TokenType = "let"
	CONST TokenType = "const"
	AS    TokenType = "as"
	OR    TokenType = "or"

	// Operators & Symbols
	ASSIGN TokenType = "="
	ARROW  TokenType = "->"
	FAT_ARROW TokenType = "=>"
	AT     TokenType = "@" // Instanciation @Unit()
	AMP    TokenType = "&" // Propriété interne &prop

	// Comparaisons
	EQ  TokenType = "=="
	NEQ TokenType = "!="
	LT  TokenType = "<"
	LTE TokenType = "<="
	GT  TokenType = ">"
	GTE TokenType = ">="

	// Delimiters
	LPAREN   TokenType = "("
	RPAREN   TokenType = ")"
	LBRACKET TokenType = "["
	RBRACKET TokenType = "]"
	LBRACE   TokenType = "{"
	RBRACE   TokenType = "}"
	COMMA    TokenType = ","
	COLON    TokenType = ":"
	DOT      TokenType = "."
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
	"let":   LET,
	"const": CONST,
	"as":    AS,
	"or":    OR,
	"true":  BOOL,
	"false": BOOL,
}

var directives = map[string]TokenType{
	"#USE":           USE_DIR,
	"#ARG":           ARG_DIR,
	"#UNIT":          UNIT_DIR,
	"#DECLARE":       DECLARE_DIR,
	"#DIRECTIVE":     DIRECTIVE_DIR,
	"#FUNC":          FUNC_DIR,
	"#RETURN":        RETURN_DIR,
	"#WITH":          WITH_DIR,
	"#END":           END_DIR,
	"#IF":            IF_DIR,
	"#ELIF":          ELIF_DIR,
	"#ELSE":          ELSE_DIR,
	"#MATCH":         MATCH_DIR,
	"#CASE":          CASE_DIR,
	"#DEFAULT":       DEFAULT_DIR,
	"#DISPATCH_TREE": DISPATCH_TREE,
	"#FILE":          FILE_DIR,
	"#SCOPE":         SCOPE_DIR,
	"#GLOBAL":        GLOBAL_DIR,
	"#global":        GLOBAL_DIR,
	"#PASS":          PASS_DIR,
	"#pass":          PASS_DIR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

const (
	CUSTOM_DIR TokenType = "CUSTOM_DIR" // ex: #HEADER, #SET, #OPTION
)

func LookupDirective(dir string) TokenType {
	if tok, ok := directives[dir]; ok {
		return tok
	}
	if strings.HasPrefix(dir, "#") {
		return CUSTOM_DIR
	}
	return ILLEGAL
}
