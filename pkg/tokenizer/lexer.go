package tokenizer

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"yur4uwe/pac/internal/helpers"
)

var (
	ErrInvalidDot          = errors.New("invalid dot in number")
	ErrInvalidTrailingChar = errors.New("invalid trailing character")
	ErrInvalidScientific   = errors.New("invalid scientific notation")
	ErrInvalidEncDigit     = errors.New("invalid digit for base")
	ErrInvalidBase         = errors.New("invalid number base")
	ErrMissingDigits       = errors.New("encoded number missing digits")
)

type Lexer struct {
	input    []rune
	position int
	readPos  int
	ch       rune
	line     uint
	col      uint
}

type lexerState struct{}

func (l *Lexer) getPos() Pos {
	return Pos{
		Line:   l.line,
		Col:    l.col,
		Offset: uint(l.position),
	}
}

func (l *Lexer) String() string {
	return fmt.Sprintf("\nposition: %s\nreadPos: %d\nch: %q", l.getPos().String(), l.readPos, l.ch)
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		input: []rune(input),
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() (ch rune) {
	ch = l.ch
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}

	if l.ch == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}

	l.position = l.readPos
	l.readPos++
	return
}

func (l *Lexer) isEOF() bool {
	return l.ch == 0
}

func (l *Lexer) consumeChar(tokenType TokenType, literal rune) Token {
	tok := Token{
		Type:    tokenType,
		Literal: string(literal),
		Span:    SourceSpan{Start: l.getPos()},
	}
	l.readChar()
	tok.Span.End = l.getPos()
	return tok
}

func (l *Lexer) peekChar() rune {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) Emit() Token {
	l.findNextTokenStart()

	if tok, resolved := ResolveUnambiguousToken(l); resolved {
		return tok
	} else {
		return ResolveAmbiguousToken(l)
	}
}

func (l *Lexer) findNextTokenStart() {
	for l.ch != '\n' && l.ch != '\r' && unicode.IsSpace(l.ch) {
		l.readChar()
	}
}

// func (l *Lexer) isPackageSeparator() bool {
// 	if len(l.packageSeparator) == 0 {
// 		return false
// 	}
// 	for i, r := range l.packageSeparator {
// 		if l.lookAheadChar(i) != r {
// 			return false
// 		}
// 	}
// 	return true
// }

func (l *Lexer) lookAheadChar(n int) rune {
	pos := l.position + n
	if pos >= len(l.input) {
		return 0
	}
	return l.input[pos]
}

// func (l *Lexer) consumePackageSeparator() string {
// 	start := l.position
// 	for range l.packageSeparator {
// 		l.readChar()
// 	}
// 	return string(l.input[start:l.position])
// }

func (l *Lexer) readUntilWhitespaceOrNewline() string {
	start := l.position
	for !l.isEOF() && !unicode.IsSpace(l.ch) {
		l.readChar()
	}
	return string(l.input[start:l.position])
}

func (l *Lexer) isIdentifierRune() bool {
	return helpers.IsIdentifierRune(l.ch) || l.ch == '\\' || l.ch == '_'
}

func (l *Lexer) readIdentifier() string {
	var result []rune

	for !l.isEOF() {
		if l.ch == '\\' && l.peekChar() != 0 {
			// Skip the backslash and take the next character literally
			l.readChar()
			if l.ch != 0 {
				result = append(result, l.ch)
				l.readChar()
			}
		} else if helpers.IsIdentifierRune(l.ch) || unicode.IsDigit(l.ch) {
			result = append(result, l.ch)
			l.readChar()
		} else {
			break
		}
	}
	return string(result)
}

func (l *Lexer) readEncodedNumber() (string, error) {
	start := l.position
	l.readChar() // consume '0'

	encoding := l.ch
	l.readChar() // consume encoding char (x, b, o)

	var base int
	switch encoding {
	case 'x', 'X':
		base = 16
	case 'b', 'B':
		base = 2
	case 'o', 'O':
		base = 8
	default:
		return string(l.input[start:l.position]), fmt.Errorf("%w: %c", ErrInvalidBase, encoding)
	}

	hasDigits := false
	for !l.isEOF() {
		if !unicode.IsDigit(l.ch) && !helpers.IsIdentifierRune(l.ch) {
			break
		}

		val, ok := getDigitValue(l.ch)
		if !ok || val >= base {
			return string(l.input[start:l.position]), fmt.Errorf("%w: base: '%d', digit: %c", ErrInvalidEncDigit, base, l.ch)
		}
		hasDigits = true
		l.readChar()
	}

	if !hasDigits {
		return string(l.input[start:l.position]), ErrMissingDigits
	}

	return string(l.input[start:l.position]), nil
}

func getDigitValue(ch rune) (int, bool) {
	if ch >= '0' && ch <= '9' {
		return int(ch - '0'), true
	}
	if ch >= 'a' && ch <= 'f' {
		return int(ch - 'a' + 10), true
	}
	if ch >= 'A' && ch <= 'F' {
		return int(ch - 'A' + 10), true
	}
	return 0, false
}

func (l *Lexer) readNumber() (string, error) {
	start := l.position

	// Check for encoded numbers (0x, 0b, 0o)
	if l.ch == '0' {
		peek := l.peekChar()
		switch peek {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			return l.readEncodedNumber()
		}
	}

state_integer:
	if unicode.IsDigit(l.ch) {
		l.readChar()
		goto state_integer
	}
	if l.ch == '.' {
		l.readChar()
		goto state_fraction
	}
	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		goto state_exponent_start
	}
	goto state_validate

state_fraction:
	if unicode.IsDigit(l.ch) {
		l.readChar()
		goto state_fraction
	}
	if l.ch == 'e' || l.ch == 'E' {
		l.readChar()
		goto state_exponent_start
	}
	if l.ch == '.' {
		return string(l.input[start:l.position]), ErrInvalidDot
	}
	goto state_validate

state_exponent_start:
	if l.ch == '+' || l.ch == '-' {
		l.readChar()
	}
	if !unicode.IsDigit(l.ch) {
		return string(l.input[start:l.position]), ErrInvalidScientific
	}
	goto state_exponent

state_exponent:
	if unicode.IsDigit(l.ch) {
		l.readChar()
		goto state_exponent
	}
	if l.ch == '.' {
		return string(l.input[start:l.position]), ErrInvalidDot
	}
	if l.ch == 'e' || l.ch == 'E' {
		return string(l.input[start:l.position]), ErrInvalidScientific
	}
	goto state_validate

state_validate:
	if helpers.IsIdentifierRune(l.ch) {
		return string(l.input[start:l.position]), fmt.Errorf("%w: '%c'", ErrInvalidTrailingChar, l.ch)
	}
	return string(l.input[start:l.position]), nil
}

func (l *Lexer) readString() string {
	start := l.position
	l.readChar() // consume opening "
	for l.ch != '"' && !l.isEOF() {
		if l.ch == '\\' && l.peekChar() != 0 {
			l.readChar() // consume backslash
			l.readChar() // consume escaped char
		} else {
			l.readChar()
		}
	}
	l.readChar() // consume closing "
	end := l.position
	return string(l.input[start:end])
}

func (l *Lexer) readLineComment() string {
	// consume single quote
	l.readChar()
	start := l.position
	for l.ch != '\n' && !l.isEOF() {
		l.readChar()
	}
	return strings.TrimSpace(string(l.input[start:l.position]))
}

func (l *Lexer) readBlockComment() string {
	// consume "/'"
	l.readChar()
	l.readChar()
	var sb strings.Builder
	for !l.isEOF() {
		if l.ch == '\'' && l.peekChar() == '/' {
			l.readChar()
			l.readChar()
			break
		}
		sb.WriteRune(l.ch)
		l.readChar()
	}
	return sb.String()
}
