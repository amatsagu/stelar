package main

type TokenType uint8

const (
	ILLEGAL_TOKEN TokenType = iota // uninitialized or illegal state
	EOF_TOKEN                      // End of File

	// Identifiers and Literals
	IDENT_TOKEN   // main, res, myStack, i
	INTEGER_TOKEN // 10, 20
	FLOAT_TOKEN   // 3.14
	STRING_TOKEN  // "Hello World"

	// Operators
	PLUS_TOKEN     // +
	MINUS_TOKEN    // -
	ASTERISK_TOKEN // *
	SLASH_TOKEN    // /
	GT_TOKEN       // >
	LT_TOKEN       // <
	EQ_TOKEN       // ==

	// Punctuation & Access Modifiers
	SEMICOLON_TOKEN  // ; (Greedy terminator, stop sign)
	UNDERSCORE_TOKEN // _ (Explicit pop / placeholder)
	DOT_TOKEN        // . (Used for stack.method)
	PEEK_TOKEN       // .0, .1, .4 (Treat index peeking as its own token)

	// Keywords
	PROC_TOKEN
	END_TOKEN
	STACK_TOKEN
	PUT_TOKEN
	POP_TOKEN
	CLONE_TOKEN
	RESET_TOKEN
	FOR_TOKEN
	SWITCH_TOKEN
	CASE_TOKEN
	DEFAULT_TOKEN
	IF_TOKEN
	ELSE_TOKEN
	BIND_TOKEN
	REQUIRE_TOKEN
	WITH_TOKEN
	SIZE_TOKEN
	PRINT_TOKEN
	EXIT_TOKEN
	SWAP_TOKEN
	UNSAFE_PROC_TOKEN // ~proc
	RETURN_TOKEN
)

var keywords = map[string]TokenType{
	"proc":    PROC_TOKEN,
	"~proc":   UNSAFE_PROC_TOKEN,
	"return":  RETURN_TOKEN,
	"end":     END_TOKEN,
	"stack":   STACK_TOKEN,
	"put":     PUT_TOKEN,
	"pop":     POP_TOKEN,
	"clone":   CLONE_TOKEN,
	"reset":   RESET_TOKEN,
	"for":     FOR_TOKEN,
	"switch":  SWITCH_TOKEN,
	"case":    CASE_TOKEN,
	"default": DEFAULT_TOKEN,
	"if":      IF_TOKEN,
	"else":    ELSE_TOKEN,
	"bind":    BIND_TOKEN,
	"require": REQUIRE_TOKEN,
	"with":    WITH_TOKEN,
	"size":    SIZE_TOKEN,
	"print":   PRINT_TOKEN,
	"exit":    EXIT_TOKEN,
	"swap":    SWAP_TOKEN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT_TOKEN
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int
	column       int
}

func NewLexer(input string) *Lexer {
	l := &Lexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Line = l.line
	tok.Column = l.column

	switch l.ch {
	case ';':
		tok = l.newToken(SEMICOLON_TOKEN, string(l.ch))
	case '_':
		tok = l.newToken(UNDERSCORE_TOKEN, string(l.ch))
	case '+':
		tok = l.newToken(PLUS_TOKEN, string(l.ch))
	case '-':
		tok = l.newToken(MINUS_TOKEN, string(l.ch))
	case '*':
		tok = l.newToken(ASTERISK_TOKEN, string(l.ch))
	case '/':
		if l.peekChar() == '/' {
			l.skipComment()
			return l.NextToken()
		} else if l.peekChar() == '*' {
			l.skipBlockComment()
			return l.NextToken()
		} else {
			tok = l.newToken(SLASH_TOKEN, string(l.ch))
		}
	case '<':
		tok = l.newToken(LT_TOKEN, string(l.ch))
	case '>':
		tok = l.newToken(GT_TOKEN, string(l.ch))
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = l.newToken(EQ_TOKEN, literal)
		} else {
			tok = l.newToken(ILLEGAL_TOKEN, string(l.ch))
		}
	case '~':
		if l.peekChar() == 'p' {
			l.readChar() // consume ~
			ident := l.readIdentifier()
			literal := "~" + ident
			if tokType, ok := keywords[literal]; ok {
				tok.Type = tokType
				tok.Literal = literal
				tok.Line = l.line
				tok.Column = l.column - len(literal) + 1 // Adjust column back to start of ~
				return tok
			}
			tok = l.newToken(ILLEGAL_TOKEN, "~"+ident)
		} else {
			tok = l.newToken(ILLEGAL_TOKEN, string(l.ch))
		}
	case '"':
		tok.Type = STRING_TOKEN
		tok.Literal = l.readString()
	case '.':
		if isDigit(l.peekChar()) {
			tok.Type = PEEK_TOKEN
			tok.Literal = l.readPeekIndex()
			return tok
		} else if isLetter(l.peekChar()) || isOperator(l.peekChar()) {
			tok = l.newToken(DOT_TOKEN, string(l.ch))
		} else {
			tok = l.newToken(ILLEGAL_TOKEN, string(l.ch))
		}
	case 0:
		tok.Literal = ""
		tok.Type = EOF_TOKEN
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Literal = l.readNumber()
			if l.ch == '.' && isDigit(l.peekChar()) {
				l.readChar() // consume .
				tok.Literal += "." + l.readNumber()
				tok.Type = FLOAT_TOKEN
			} else {
				tok.Type = INTEGER_TOKEN
			}
			return tok
		} else {
			tok = l.newToken(ILLEGAL_TOKEN, string(l.ch))
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}

	l.position = l.readPosition
	l.readPosition += 1

	if l.ch == '\n' {
		l.line += 1
		l.column = 0
	} else {
		l.column += 1
	}
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() {
	l.readChar() // consume /
	l.readChar() // consume *
	for {
		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume *
			l.readChar() // consume /
			break
		}
		if l.ch == 0 {
			break
		}
		l.readChar()
	}
}

func (l *Lexer) readPeekIndex() string {
	position := l.position
	l.readChar() // consume the '.'

	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isOperator(ch byte) bool {
	return ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '>' || ch == '<' || ch == '='
}

func (l *Lexer) newToken(tokenType TokenType, literal string) Token {
	return Token{Type: tokenType, Literal: literal, Line: l.line, Column: l.column}
}
