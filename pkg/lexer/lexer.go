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

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	if l.ch == 0 {
		return Token{
			Type:    EOF,
			Literal: "",
			Pos:     Position{Filename: l.filename, Line: l.line, Column: l.column},
		}
	}

	startPos := Position{Filename: l.filename, Line: l.line, Column: l.column}

	// 1. Directives débutant par '#'
	if l.ch == '#' {
		directive := l.readDirective()
		tokType := LookupDirective(directive)
		return Token{Type: tokType, Literal: directive, Pos: startPos}
	}

	// 2. Chaines de caractères "..." ou """...""" ou '''...''' ou `...`
	if l.ch == '"' {
		if l.peekChar() == '"' && l.peekChar2() == '"' {
			str := l.readTripleString('"')
			return Token{Type: STRING, Literal: str, Pos: startPos}
		}
		str := l.readString()
		return Token{Type: STRING, Literal: str, Pos: startPos}
	}
	if l.ch == '\'' {
		if l.peekChar() == '\'' && l.peekChar2() == '\'' {
			str := l.readTripleString('\'')
			return Token{Type: STRING, Literal: str, Pos: startPos}
		}
	}
	if l.ch == '`' {
		str := l.readRawBacktickString()
		return Token{Type: STRING, Literal: str, Pos: startPos}
	}

	// 3. Opérateurs composites (->, =>, ==, !=, <=, >=)
	switch l.ch {
	case '-':
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return Token{Type: ARROW, Literal: "->", Pos: startPos}
		}
		// Négatif possible pour les nombres
		if unicode.IsDigit(l.peekChar()) {
			num := l.readNumberOrDuration()
			return Token{Type: NUMBER, Literal: "-" + num, Pos: startPos}
		}
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: EQ, Literal: "==", Pos: startPos}
		}
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return Token{Type: FAT_ARROW, Literal: "=>", Pos: startPos}
		}
		l.readChar()
		return Token{Type: ASSIGN, Literal: "=", Pos: startPos}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: NEQ, Literal: "!=", Pos: startPos}
		}
	case '<':
		if l.peekChar() == '<' && l.peekChar2() == '<' {
			str := l.readChevronHeredoc()
			return Token{Type: STRING, Literal: str, Pos: startPos}
		}
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: LTE, Literal: "<=", Pos: startPos}
		}
		l.readChar()
		return Token{Type: LT, Literal: "<", Pos: startPos}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return Token{Type: GTE, Literal: ">=", Pos: startPos}
		}
		l.readChar()
		return Token{Type: GT, Literal: ">", Pos: startPos}
	case '@':
		l.readChar()
		return Token{Type: AT, Literal: "@", Pos: startPos}
	case '&':
		l.readChar()
		return Token{Type: AMP, Literal: "&", Pos: startPos}
	case '(':
		l.readChar()
		return Token{Type: LPAREN, Literal: "(", Pos: startPos}
	case ')':
		l.readChar()
		return Token{Type: RPAREN, Literal: ")", Pos: startPos}
	case '[':
		l.readChar()
		return Token{Type: LBRACKET, Literal: "[", Pos: startPos}
	case ']':
		l.readChar()
		return Token{Type: RBRACKET, Literal: "]", Pos: startPos}
	case '{':
		l.readChar()
		return Token{Type: LBRACE, Literal: "{", Pos: startPos}
	case '}':
		l.readChar()
		return Token{Type: RBRACE, Literal: "}", Pos: startPos}
	case ',':
		l.readChar()
		return Token{Type: COMMA, Literal: ",", Pos: startPos}
	case ':':
		if l.peekChar() == ':' {
			l.readChar()
			l.readChar()
			return Token{Type: DOUBLE_COLON, Literal: "::", Pos: startPos}
		}
		l.readChar()
		return Token{Type: COLON, Literal: ":", Pos: startPos}
	case '.':
		l.readChar()
		return Token{Type: DOT, Literal: ".", Pos: startPos}
	}

	// 4. Nombres et Durées (ex: 3000, 30s, 500ms)
	if unicode.IsDigit(l.ch) {
		tokType, literal := l.readNumberAndSuffix()
		return Token{Type: tokType, Literal: literal, Pos: startPos}
	}

	// 5. Identifiants et Mots-Clés
	if isLetter(l.ch) {
		ident := l.readIdentifier()
		tokType := LookupIdent(ident)
		return Token{Type: tokType, Literal: ident, Pos: startPos}
	}

	illegalChar := string(l.ch)
	l.readChar()
	return Token{Type: ILLEGAL, Literal: illegalChar, Pos: startPos}
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

func (l *Lexer) readDirective() string {
	start := l.position
	for l.ch == '#' || isLetter(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readString() string {
	l.readChar() // saute le premier "
	var buf strings.Builder
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			next := l.peekChar()
			switch next {
			case '"':
				buf.WriteRune('"')
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
	if l.ch == '"' {
		l.readChar() // saute le dernier "
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
	for unicode.IsDigit(l.ch) || l.ch == '.' {
		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readNumberAndSuffix() (TokenType, string) {
	start := l.position
	for unicode.IsDigit(l.ch) || l.ch == '.' {
		l.readChar()
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
