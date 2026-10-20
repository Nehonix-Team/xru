package formatter

import (
	"strings"

	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// Format takes raw XRU source code and returns a beautifully formatted,
// indented, semicolon-complete version (prettier-like).
func Format(source string) (string, error) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "", nil
	}

	l := lexer.New("format.xru", source)
	l.KeepComments = true

	var tokens []lexer.Token
	for {
		tok := l.NextToken()
		if tok.Type == lexer.EOF {
			break
		}
		tokens = append(tokens, tok)
	}

	if len(tokens) == 0 {
		return "", nil
	}

	f := &fmtEngine{
		tokens: tokens,
		pos:    0,
	}

	return f.format(), nil
}

type fmtEngine struct {
	tokens []lexer.Token
	pos    int

	indent        int
	buf           strings.Builder
	lineEmpty     bool
	consecNL      int
	inParen       int
	inBracket     int
	braceStack    []string
	lastStarter   string
	lastClosedCtx string
	prevTokType   lexer.TokenType
	prevRaw       string

	inHeader    bool
	inForHeader bool
	inArrowExpr bool
}

func (f *fmtEngine) cur() lexer.Token {
	if f.pos < len(f.tokens) {
		return f.tokens[f.pos]
	}
	return lexer.Token{Type: lexer.EOF}
}

func (f *fmtEngine) peek() lexer.Token {
	if f.pos+1 < len(f.tokens) {
		return f.tokens[f.pos+1]
	}
	return lexer.Token{Type: lexer.EOF}
}

func (f *fmtEngine) prev() lexer.Token {
	if f.pos > 0 && f.pos-1 < len(f.tokens) {
		return f.tokens[f.pos-1]
	}
	return lexer.Token{Type: lexer.EOF}
}

func (f *fmtEngine) writeIndent() {
	if f.lineEmpty {
		f.buf.WriteString(strings.Repeat("    ", f.indent))
		f.lineEmpty = false
	}
}

func (f *fmtEngine) write(s string) {
	f.writeIndent()
	f.buf.WriteString(s)
	f.consecNL = 0
}

func (f *fmtEngine) writeSpace() {
	if !f.lineEmpty && !strings.HasSuffix(f.buf.String(), " ") {
		f.buf.WriteString(" ")
	}
}

func (f *fmtEngine) writeNewline() {
	f.buf.WriteString("\n")
	f.lineEmpty = true
	f.consecNL++
}

func (f *fmtEngine) ensureBlankLine() {
	if f.buf.Len() == 0 {
		return
	}
	if f.consecNL < 2 {
		f.writeNewline()
	}
}

func isStatementStarter(t lexer.TokenType) bool {
	switch t {
	case lexer.USE, lexer.MOD, lexer.FROM, lexer.GLOBAL, lexer.ARG,
		lexer.LET, lexer.CONST, lexer.UNIT, lexer.INTERFACE, lexer.TYPE,
		lexer.FUNC, lexer.DIRECTIVE, lexer.DECLARE, lexer.IF, lexer.FOR,
		lexer.WHILE, lexer.MATCH, lexer.RETURN, lexer.PASS, lexer.FILE,
		lexer.IN, lexer.WITH, lexer.DISPATCH, lexer.DISPATCH_TREE, lexer.TRAP:
		return true
	}
	return false
}

func isBlockBrace(context string) bool {
	switch context {
	case "unit", "func", "directive", "if", "elif", "else", "for", "while", "match", "case", "default", "file", "in", "with", "interface", "trap":
		return true
	}
	return false
}

func isBinaryOp(t lexer.TokenType) bool {
	switch t {
	case lexer.ASSIGN, lexer.MUT_ASSIGN, lexer.PLUS, lexer.MINUS, lexer.ASTERISK,
		lexer.SLASH, lexer.PERCENT, lexer.EQ, lexer.NEQ, lexer.LT, lexer.LTE,
		lexer.GT, lexer.GTE, lexer.AND, lexer.OR_LOGICAL, lexer.PIPE,
		lexer.ARROW, lexer.FAT_ARROW, lexer.OR:
		return true
	}
	return false
}

func (f *fmtEngine) format() string {
	f.lineEmpty = true

	for f.pos < len(f.tokens) {
		tok := f.cur()

		if tok.Type == lexer.FOR {
			f.inForHeader = true
			f.inHeader = true
		} else if tok.Type == lexer.IF || tok.Type == lexer.ELIF || tok.Type == lexer.WHILE ||
			tok.Type == lexer.MATCH || tok.Type == lexer.FILE || tok.Type == lexer.WITH || tok.Type == lexer.TRAP {
			f.inHeader = true
		} else if tok.Type == lexer.ARROW {
			f.inArrowExpr = true
		}

		if isStatementStarter(tok.Type) {
			if tok.Type == lexer.IN && f.inForHeader {
				// not a statement starter
			} else {
				f.lastStarter = string(tok.Type)
			}
		}

		if f.prevTokType == lexer.AT && tok.Type == lexer.MAP {
			if f.lastStarter != "interface" && f.lastStarter != "trap" && f.lastStarter != "from" && f.lastStarter != "use" {
				f.lastStarter = "map"
			}
		}

		switch tok.Type {
		case lexer.COMMENT:
			f.handleComment(tok)
			f.pos++
			continue

		case lexer.AS:
			f.writeSpace()
			f.write("as")
			f.writeSpace()

		case lexer.LPAREN:
			f.inParen++
			// No space before ( if preceded by function name or call
			f.write("(")

		case lexer.RPAREN:
			if f.inParen > 0 {
				f.inParen--
			}
			f.inArrowExpr = false
			f.write(")")

		case lexer.LBRACKET:
			f.inBracket++
			f.write("[")

		case lexer.RBRACKET:
			if f.inBracket > 0 {
				f.inBracket--
			}
			f.inArrowExpr = false
			f.write("]")

		case lexer.LBRACE:
			if f.isInlineMapBrace() {
				f.write("{")
				f.pos++
				isFirst := true
				for f.pos < len(f.tokens) && f.cur().Type != lexer.RBRACE {
					if f.cur().Type == lexer.COMMA {
						f.write(",")
						f.writeSpace()
					} else {
						if !isFirst && f.prevTokType != lexer.COMMA && f.needsSpaceBefore(f.cur()) {
							f.writeSpace()
						}
						f.write(f.cur().Literal)
					}
					isFirst = false
					f.prevTokType = f.cur().Type
					f.pos++
				}
				if f.pos < len(f.tokens) && f.cur().Type == lexer.RBRACE {
					f.write("}")
					f.prevTokType = lexer.RBRACE
					if f.peek().Type == lexer.AS || f.peek().Type == lexer.LBRACE {
						f.writeSpace()
					}
				}
				f.pos++
				continue
			}

			f.inHeader = false
			f.inForHeader = false
			f.inArrowExpr = false

			ctx := "block"
			if f.lastStarter != "" {
				ctx = f.lastStarter
				f.lastStarter = ""
			}
			f.braceStack = append(f.braceStack, ctx)

			f.writeSpace()
			f.write("{")
			f.writeNewline()
			f.indent++

		case lexer.RBRACE:
			f.inHeader = false
			f.inForHeader = false
			f.inArrowExpr = false

			if f.indent > 0 {
				f.indent--
			}
			ctx := "block"
			if len(f.braceStack) > 0 {
				ctx = f.braceStack[len(f.braceStack)-1]
				f.braceStack = f.braceStack[:len(f.braceStack)-1]
			}
			f.lastClosedCtx = ctx

			// If previous line wasn't empty, ensure we are on a new line
			if !f.lineEmpty {
				f.writeNewline()
			}
			f.write("}")

			peekTok := f.peek()
			if peekTok.Type == lexer.ELSE || peekTok.Type == lexer.ELIF {
				f.writeSpace()
			} else if peekTok.Type == lexer.SEMICOLON {
				if isBlockBrace(ctx) {
					// Drop redundant semicolon after block statement }
					f.pos++
					f.writeNewline()
					if f.indent == 0 && (ctx == "unit" || ctx == "func" || ctx == "interface") {
						f.ensureBlankLine()
					}
				}
				// If not a block brace (e.g. map), do not write newline now;
				// the following SEMICOLON will write on the same line: };
			} else {
				f.writeNewline()
				// Top-level units and functions get an empty line after
				if f.indent == 0 && (ctx == "unit" || ctx == "func" || ctx == "interface") {
					f.ensureBlankLine()
				}
			}

		case lexer.SEMICOLON:
			f.inArrowExpr = false
			f.inHeader = false
			f.inForHeader = false
			if f.prevTokType == lexer.RBRACE && isBlockBrace(f.lastClosedCtx) {
				// Redundant semicolon after block brace: ignore
				f.prevTokType = tok.Type
				f.pos++
				continue
			}
			if f.lineEmpty {
				// Isolated semicolon on an empty line: ignore
				f.prevTokType = tok.Type
				f.pos++
				continue
			}
			f.write(";")
			f.writeNewline()

		case lexer.COMMA:
			f.inArrowExpr = false
			f.write(",")
			f.writeSpace()

		case lexer.COLON:
			f.write(":")
			f.writeSpace()

		case lexer.DOUBLE_COLON:
			// No space around ::
			f.write("::")

		case lexer.DOT:
			f.write(".")

		case lexer.RANGE, lexer.ELLIPSIS:
			f.write(tok.Literal)

		case lexer.AT:
			// Space before @ if preceded by identifier, etc. (e.g. interface EnvObj @map)
			if f.needsSpaceBefore(tok) {
				f.writeSpace()
			}
			f.write("@")

		case lexer.AMP:
			// Internal prop &prop or &self::
			f.write("&")

		case lexer.BANG:
			if f.needsSpaceBefore(tok) {
				f.writeSpace()
			}
			f.write("!")

		case lexer.UNIT, lexer.FUNC, lexer.INTERFACE:
			if f.indent == 0 || (f.indent == 1 && len(f.braceStack) > 0 && f.braceStack[len(f.braceStack)-1] == "unit") {
				f.ensureBlankLine()
			}
			f.write(tok.Literal)
			f.writeSpace()

		case lexer.ELIF, lexer.ELSE:
			if f.inArrowExpr {
				f.writeSpace()
				f.write(tok.Literal)
				f.writeSpace()
			} else {
				if f.prevTokType == lexer.RBRACE {
					f.writeSpace()
				} else if !f.lineEmpty {
					f.writeNewline()
				}
				f.write(tok.Literal)
				f.writeSpace()
			}

		default:
			// Operator spacing
			if isBinaryOp(tok.Type) {
				// Distinguish unary minus from binary minus
				if tok.Type == lexer.MINUS && (f.prevTokType == lexer.ASSIGN || f.prevTokType == lexer.MUT_ASSIGN ||
					f.prevTokType == lexer.LPAREN || f.prevTokType == lexer.LBRACKET || f.prevTokType == lexer.COMMA ||
					f.prevTokType == lexer.COLON || f.prevTokType == lexer.RETURN || f.prevTokType == lexer.ARROW) {
					f.write("-")
				} else {
					f.writeSpace()
					f.write(tok.Literal)
					f.writeSpace()
				}
			} else {
				// Normal tokens / keywords / identifiers / literals
				if !f.lineEmpty {
					// Add space between tokens if needed
					if f.needsSpaceBefore(tok) {
						f.writeSpace()
					}
				}

				// Emit literal text (preserving raw quotes / heredoc for strings)
				text := tok.Raw
				if text == "" {
					text = tok.Literal
				}
				// If string starts with raw <<<, write verbatim
				f.write(text)
			}
		}

		// Semicolon insertion check:
		// If at statement level (not in parens/brackets) and we just finished an expression or statement
		// that should have a semicolon, but didn't:
		if f.inParen == 0 && f.inBracket == 0 && tok.Type != lexer.SEMICOLON && tok.Type != lexer.LBRACE && tok.Type != lexer.RBRACE {
			nextTok := f.peek()
			if f.shouldInsertSemicolon(tok, nextTok) {
				f.write(";")
				f.writeNewline()
			}
		}

		f.prevTokType = tok.Type
		f.prevRaw = tok.Raw
		f.pos++
	}

	res := f.buf.String()
	// Clean trailing empty lines to single newline
	res = strings.TrimRight(res, " \t\r\n") + "\n"
	return res
}

func (f *fmtEngine) handleComment(tok lexer.Token) {
	if strings.HasPrefix(tok.Raw, "//") {
		if !f.lineEmpty {
			f.buf.WriteString("  ")
		} else {
			f.writeIndent()
		}
		f.buf.WriteString(strings.TrimRight(tok.Raw, "\r\n"))
		f.writeNewline()
	} else {
		// Multi-line comment /* ... */
		if !f.lineEmpty {
			f.writeNewline()
		}
		f.writeIndent()
		f.buf.WriteString(tok.Raw)
		f.writeNewline()
	}
}

func (f *fmtEngine) needsSpaceBefore(tok lexer.Token) bool {
	switch f.prevTokType {
	case lexer.LPAREN, lexer.LBRACKET, lexer.AT, lexer.AMP, lexer.DOT, lexer.DOUBLE_COLON, lexer.BANG:
		return false
	}
	switch tok.Type {
	case lexer.RPAREN, lexer.RBRACKET, lexer.DOT, lexer.DOUBLE_COLON, lexer.QUESTION:
		return false
	case lexer.BANG:
		if f.prevTokType == lexer.IF || f.prevTokType == lexer.ELIF || f.prevTokType == lexer.WHILE ||
			isBinaryOp(f.prevTokType) || f.prevTokType == lexer.RETURN {
			return true
		}
		return false
	}
	return true
}

func (f *fmtEngine) shouldInsertSemicolon(cur, next lexer.Token) bool {
	// If in header (for, if, while, etc.) or in an arrow expression, never insert semicolon
	if f.inHeader || f.inForHeader || f.inArrowExpr {
		return false
	}
	// Inside a map literal, never insert semicolons
	if len(f.braceStack) > 0 && f.braceStack[len(f.braceStack)-1] == "map" {
		return false
	}

	// Never insert semicolon right before 'in' keyword or around 'as'
	if next.Type == lexer.IN || next.Type == lexer.AS || cur.Type == lexer.AS {
		return false
	}

	// If next is semicolon, EOF, or closing brace: no need to insert before them
	if next.Type == lexer.SEMICOLON || next.Type == lexer.EOF || next.Type == lexer.RBRACE {
		// If next is RBRACE or EOF, a statement should terminate with ';' if it didn't!
		if next.Type == lexer.RBRACE || next.Type == lexer.EOF {
			return isTerminatingToken(cur.Type)
		}
		return false
	}

	// Inside block brace contexts, if the next token starts a new statement
	if isStatementStarter(next.Type) && isTerminatingToken(cur.Type) {
		return true
	}

	// If next token is on a new line and previous token can terminate a statement
	if cur.Pos.Line != next.Pos.Line && isTerminatingToken(cur.Type) {
		return true
	}

	return false
}

func isTerminatingToken(t lexer.TokenType) bool {
	switch t {
	case lexer.IDENT, lexer.STRING, lexer.NUMBER, lexer.BOOL, lexer.DURATION,
		lexer.RPAREN, lexer.RBRACKET, lexer.PASS, lexer.RETURN, lexer.QUESTION:
		return true
	}
	return false
}

func (f *fmtEngine) isInlineMapBrace() bool {
	if f.prevTokType != lexer.MAP {
		return false
	}
	if f.pos >= 3 && f.tokens[f.pos-2].Type == lexer.MAP && f.tokens[f.pos-3].Type == lexer.AT {
		if f.pos >= 4 && f.tokens[f.pos-4].Type == lexer.DOUBLE_COLON {
			return true
		}
	}
	if f.inHeader || f.lastStarter == "trap" || f.lastStarter == "from" || f.lastStarter == "use" {
		return true
	}
	return false
}
