package lexer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	filename     string
	input        string
	position     int  // Position courante dans input (pointe sur ch)
	readPosition int  // Prochaine position de lecture (après ch)
	ch           rune // Caractère courant sous examen
	line         int  // Ligne courante (1-indexed)
	column       int  // Colonne courante (1-indexed)
	KeepComments bool
}

func New(filename string, input string) *Lexer {
	l := &Lexer{
		filename: filename,
		input:    input,
		line:     1,
		column:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = len(l.input)
	} else {
		r, width := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += width
		l.column++
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

func (l *Lexer) peekChar2() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	_, width := utf8.DecodeRuneInString(l.input[l.readPosition:])
	nextPos := l.readPosition + width
	if nextPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[nextPos:])
	return r
}

func (l *Lexer) makeToken(tokType TokenType, lit string, startPos Position, startOffset int) Token {
	raw := l.input[startOffset:l.position]
	return Token{Type: tokType, Literal: lit, Pos: startPos, Raw: raw}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		l.readChar()
	}
}

func (l *Lexer) NextToken() Token {
	if l.KeepComments {
		l.skipWhitespace()
	} else {
		l.skipWhitespaceAndComments()
	}

	if l.ch == 0 {
		return Token{
			Type:    EOF,
			Literal: "",
			Raw:     "",
			Pos:     Position{Filename: l.filename, Line: l.line, Column: l.column},
		}
	}

	startPos := Position{Filename: l.filename, Line: l.line, Column: l.column}
	startOffset := l.position

	// Comment check when KeepComments is enabled
	if l.KeepComments {
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			return l.makeToken(COMMENT, l.input[startOffset:l.position], startPos, startOffset)
		}
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar()
			l.readChar()
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '\n' {
					l.line++
					l.column = 0
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar()
					l.readChar()
					break
				}
				l.readChar()
			}
			return l.makeToken(COMMENT, l.input[startOffset:l.position], startPos, startOffset)
		}
	}

	// 1. Chaines de caractères "..." ou """...""" ou '''...''' ou `...`
	if l.ch == '"' {
		if l.peekChar() == '"' && l.peekChar2() == '"' {
			str := l.readTripleString('"')
			return l.makeToken(STRING, str, startPos, startOffset)
		}
		str := l.readString()
		return l.makeToken(STRING, str, startPos, startOffset)
	}
	if l.ch == '\'' {
		if l.peekChar() == '\'' && l.peekChar2() == '\'' {
			str := l.readTripleString('\'')
			return l.makeToken(STRING, str, startPos, startOffset)
		}
		str := l.readQuotedString('\'')
		return l.makeToken(STRING, str, startPos, startOffset)
	}
	if l.ch == '`' {
		str := l.readRawBacktickString()
		return l.makeToken(STRING, str, startPos, startOffset)
	}

	// 3. Opérateurs composites (->, =>, ==, !=, <=, >=)
	switch l.ch {
	case '+':
		l.readChar()
		return l.makeToken(PLUS, "+", startPos, startOffset)
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return l.makeToken(ARROW, "->", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(MINUS, "-", startPos, startOffset)
	case '*':
		l.readChar()
		return l.makeToken(ASTERISK, "*", startPos, startOffset)
	case '/':
		l.readChar()
		return l.makeToken(SLASH, "/", startPos, startOffset)
	case '%':
		l.readChar()
		return l.makeToken(PERCENT, "%", startPos, startOffset)
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return l.makeToken(EQ, "==", startPos, startOffset)
		}
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return l.makeToken(FAT_ARROW, "=>", startPos, startOffset)
		}
		if l.peekChar() == ':' {
			l.readChar()
			l.readChar()
			return l.makeToken(MUT_ASSIGN, "=:", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(ASSIGN, "=", startPos, startOffset)
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return l.makeToken(NEQ, "!=", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(BANG, "!", startPos, startOffset)
	case '<':
		if l.peekChar() == '<' && l.peekChar2() == '<' {
			str := l.readChevronHeredoc()
			return l.makeToken(STRING, str, startPos, startOffset)
		}
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return l.makeToken(LTE, "<=", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(LT, "<", startPos, startOffset)
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return l.makeToken(GTE, ">=", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(GT, ">", startPos, startOffset)
	case '@':
		l.readChar()
		return l.makeToken(AT, "@", startPos, startOffset)
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			l.readChar()
			return l.makeToken(AND, "&&", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(AMP, "&", startPos, startOffset)
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			l.readChar()
			return l.makeToken(OR_LOGICAL, "||", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(PIPE, "|", startPos, startOffset)
	case '(':
		l.readChar()
		return l.makeToken(LPAREN, "(", startPos, startOffset)
	case ')':
		l.readChar()
		return l.makeToken(RPAREN, ")", startPos, startOffset)
	case '[':
		l.readChar()
		return l.makeToken(LBRACKET, "[", startPos, startOffset)
	case ']':
		l.readChar()
		return l.makeToken(RBRACKET, "]", startPos, startOffset)
	case '{':
		l.readChar()
		return l.makeToken(LBRACE, "{", startPos, startOffset)
	case '}':
		l.readChar()
		return l.makeToken(RBRACE, "}", startPos, startOffset)
	case ',':
		l.readChar()
		return l.makeToken(COMMA, ",", startPos, startOffset)
	case ':':
		if l.peekChar() == ':' {
			l.readChar()
			l.readChar()
			return l.makeToken(DOUBLE_COLON, "::", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(COLON, ":", startPos, startOffset)
	case '.':
		if l.peekChar() == '.' {
			if l.peekChar2() == '.' {
				l.readChar()
				l.readChar()
				l.readChar()
				return l.makeToken(ELLIPSIS, "...", startPos, startOffset)
			}
			l.readChar()
			l.readChar()
			return l.makeToken(RANGE, "..", startPos, startOffset)
		}
		l.readChar()
		return l.makeToken(DOT, ".", startPos, startOffset)
	case ';':
		l.readChar()
		return l.makeToken(SEMICOLON, ";", startPos, startOffset)
	case '?':
		l.readChar()
		return l.makeToken(QUESTION, "?", startPos, startOffset)
	}

	// 4. Nombres et Durées (ex: 3000, 30s, 500ms)
	if unicode.IsDigit(l.ch) {
		tokType, literal := l.readNumberAndSuffix()
		return l.makeToken(tokType, literal, startPos, startOffset)
	}

	// 5. Identifiants et Mots-Clés
	if isLetter(l.ch) {
		ident := l.readIdentifier()
		tokType := LookupIdent(ident)
		return l.makeToken(tokType, ident, startPos, startOffset)
	}

	illegalChar := string(l.ch)
	l.readChar()
	return l.makeToken(ILLEGAL, illegalChar, startPos, startOffset)
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		if l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
			if l.ch == '\n' {
				l.line++
				l.column = 0
			}
			l.readChar()
			continue
		}

		// Commentaire //
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}

		// Commentaire /* ... */
		if l.ch == '/' && l.peekChar() == '*' {
			l.readChar() // saute /
			l.readChar() // saute *
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '\n' {
					l.line++
					l.column = 0
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar()
					l.readChar()
					break
				}
				l.readChar()
			}
			continue
		}

		break
	}
}


func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readString() string {
	return l.readQuotedString('"')
}

func (l *Lexer) readQuotedString(delim rune) string {
	l.readChar() // saute le premier delim
	var buf strings.Builder
	for l.ch != delim && l.ch != 0 {
		if l.ch == '\\' {
			next := l.peekChar()
			switch next {
			case delim:
				buf.WriteRune(delim)
				l.readChar()
				l.readChar()
				continue
			case '\\':
				buf.WriteRune('\\')
				l.readChar()
				l.readChar()
				continue
			case 'n':
				buf.WriteRune('\n')
				l.readChar()
				l.readChar()
				continue
			case 't':
				buf.WriteRune('\t')
				l.readChar()
				l.readChar()
				continue
			case 'r':
				buf.WriteRune('\r')
				l.readChar()
				l.readChar()
				continue
			}
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		buf.WriteRune(l.ch)
		l.readChar()
	}
	if l.ch == delim {
		l.readChar() // saute le dernier delim
	}
	return buf.String()
}


func (l *Lexer) readTripleString(quote rune) string {
	l.readChar() // saute 1er quote
	l.readChar() // saute 2ème quote
	l.readChar() // saute 3ème quote

	var buf strings.Builder
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == quote && l.peekChar() == quote && l.peekChar2() == quote {
			l.readChar() // saute 1er fermant
			l.readChar() // saute 2ème fermant
			l.readChar() // saute 3ème fermant
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		buf.WriteRune(l.ch)
		l.readChar()
	}
	return buf.String()
}

func (l *Lexer) readRawBacktickString() string {
	l.readChar() // saute le premier `
	var buf strings.Builder
	for l.ch != '`' && l.ch != 0 {
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		buf.WriteRune(l.ch)
		l.readChar()
	}
	if l.ch == '`' {
		l.readChar() // saute le dernier `
	}
	return buf.String()
}

func (l *Lexer) readChevronHeredoc() string {
	l.readChar() // saute 1er <
	l.readChar() // saute 2ème <
	l.readChar() // saute 3ème <

	var buf strings.Builder
	for {
		if l.ch == 0 {
			break
		}
		if l.ch == '>' && l.peekChar() == '>' && l.peekChar2() == '>' {
			l.readChar() // saute 1er >
			l.readChar() // saute 2ème >
			l.readChar() // saute 3ème >
			break
		}
		if l.ch == '\n' {
			l.line++
			l.column = 0
		}
		buf.WriteRune(l.ch)
		l.readChar()
	}
	return buf.String()
}

func (l *Lexer) readNumberOrDuration() string {
	start := l.position
	for {
		if unicode.IsDigit(l.ch) {
			l.readChar()
		} else if l.ch == '.' {
			if l.peekChar() == '.' || !unicode.IsDigit(l.peekChar()) {
				break
			}
			l.readChar()
		} else {
			break
		}
	}
	return l.input[start:l.position]
}

func (l *Lexer) readNumberAndSuffix() (TokenType, string) {
	start := l.position
	for {
		if unicode.IsDigit(l.ch) {
			l.readChar()
		} else if l.ch == '.' {
			if l.peekChar() == '.' || !unicode.IsDigit(l.peekChar()) {
				break
			}
			l.readChar()
		} else {
			break
		}
	}
	// Vérifier si suivi d'un suffixe d'unité (s, ms, m, h)
	if isLetter(l.ch) {
		suffixStart := l.position
		for isLetter(l.ch) {
			l.readChar()
		}
		suffix := l.input[suffixStart:l.position]
		switch strings.ToLower(suffix) {
		case "ms", "s", "m", "h":
			return DURATION, l.input[start:l.position]
		}
	}
	return NUMBER, l.input[start:l.position]
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// Clone returns an exact copy of the Lexer state for non-destructive lookahead.
func (l *Lexer) Clone() *Lexer {
	clone := *l
	return &clone
}
