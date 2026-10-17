// Package tokenizer provides tools of tokenization for PUML source code.
package tokenizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"yur4uwe/pac/internal/helpers"
)

//go:generate enumer -type=TokenType -transform=upper -json
type TokenType byte

var _ json.Unmarshaler = (*TokenType)(nil)

const (
	ILLEGAL TokenType = iota
	EOF
	NEWLINE
	IDENTIFIER
	STRING
	NUMBER

	LBRACE      // {
	RBRACE      // }
	LPAREN      // (
	RPAREN      // )
	LBRACKET    // [
	RBRACKET    // ]
	LANGLE      // <
	RANGLE      // >
	SEMICOLON   // ;
	COLON       // :
	COMMA       // ,
	DOT         // .
	EQUALS      // =
	PLUS        // +
	DASH        // -
	TILDE       // ~
	HASH        // #
	PIPE        // |
	ASTERISK    // *
	SLASH       // /
	BACKSLASH   // \
	CARET       // ^
	DOLLAR      // $
	PERCENT     // %
	AT          // @
	EXCLAMATION // !
	UNDERSCORE  // _

	COMMENT
)

type Pos struct {
	Line   uint
	Col    uint
	Offset uint
}

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

type SourceSpan struct {
	Start, End Pos
}

type Token struct {
	Type    TokenType
	Literal string
	Span    SourceSpan
}

func (t Token) EndOffset() uint {
	return t.Span.End.Offset
}

func (t Token) EndPos() Pos {
	return t.Span.End
}

func (t Token) IsZero() bool {
	return t == Token{}
}

var singleCharTokens = map[rune]TokenType{
	'\n': NEWLINE,
	'(':  LPAREN,
	')':  RPAREN,
	'{':  LBRACE,
	'}':  RBRACE,
	'[':  LBRACKET,
	']':  RBRACKET,
	'<':  LANGLE,
	'>':  RANGLE,
	',':  COMMA,
	';':  SEMICOLON,
	':':  COLON,
	'.':  DOT,
	'=':  EQUALS,
	'+':  PLUS,
	'-':  DASH,
	'~':  TILDE,
	'#':  HASH,
	'|':  PIPE,
	'*':  ASTERISK,
	'\\': BACKSLASH,
	'^':  CARET,
	'$':  DOLLAR,
	'%':  PERCENT,
	'@':  AT,
	'!':  EXCLAMATION,
	'_':  UNDERSCORE,
}

// ResolveUnambiguousToken handles tokens with obvious, unambiguous identification.
func ResolveUnambiguousToken(l *Lexer) (Token, bool) {
	if l.isEOF() {
		return Token{Type: EOF, Literal: "", Span: SourceSpan{Start: l.getPos(), End: l.getPos()}}, true
	}

	// Special case for Windows CRLF line endings
	if l.ch == '\r' {
		crlfTok := Token{
			Type:    NEWLINE,
			Literal: "\n",
			Span:    SourceSpan{Start: l.getPos()},
		}
		l.readChar()
		if l.ch == '\n' {
			l.readChar()
			crlfTok.Span.End = l.getPos()
			return crlfTok, true
		}
	}

	if tt, ok := singleCharTokens[l.ch]; ok {
		return l.consumeChar(tt, l.ch), true
	}

	start := l.getPos()
	token := Token{
		Type:    ILLEGAL,
		Literal: "",
		Span:    SourceSpan{Start: start},
	}
	switch l.ch {
	case '"':
		token.Type = STRING
		token.Literal = l.readString()
	case '\'':
		token.Type = COMMENT
		token.Literal = l.readLineComment()
	case '/':
		if l.peekChar() == '\'' {
			token.Type = COMMENT
			token.Literal = l.readBlockComment()
		} else {
			token.Type = SLASH
			token.Literal = string(l.readChar())
		}
	}

	if token.Type != ILLEGAL {
		token.Span.End = l.getPos()
		return token, true
	}

	return Token{}, false
}

// ResolveAmbiguousToken handles identifiers, keywords and numbers.
func ResolveAmbiguousToken(l *Lexer) Token {
	if helpers.IsIdentifierRune(l.ch) || l.ch == '\\' {
		start := l.getPos()
		lit := l.readIdentifier()
		span := SourceSpan{Start: start, End: l.getPos()}
		return Token{Type: IDENTIFIER, Literal: lit, Span: span}
	}

	if unicode.IsDigit(l.ch) {
		start := l.getPos()
		lit, err := l.readNumber()
		if err == nil {
			span := SourceSpan{Start: start, End: l.getPos()}
			return Token{Type: NUMBER, Literal: lit, Span: span}
		}
		// If the error is about a trailing identifier character, it means this is
		// an identifier that just happens to start with digits (like a hex color 00FFFF).
		// We continue reading it as an identifier and combine the literals.
		if errors.Is(err, ErrInvalidTrailingChar) {
			rest := l.readIdentifier()
			fullLit := lit + rest
			span := SourceSpan{Start: start, End: l.getPos()}
			return Token{Type: IDENTIFIER, Literal: fullLit, Span: span}
		}
		span := SourceSpan{Start: start, End: l.getPos()}
		return Token{Type: ILLEGAL, Literal: err.Error(), Span: span}
	}

	return l.consumeChar(ILLEGAL, l.ch)
}

func SpanEnclosing(first, last Token) SourceSpan {
	if first.Span.Start.Offset > last.Span.Start.Offset {
		first, last = last, first
	}
	return SourceSpan{
		Start: first.Span.Start,
		End:   last.EndPos(),
	}
}

func TokenSliceSpan(toks []Token) (span SourceSpan, ok bool) {
	if len(toks) == 0 {
		return SourceSpan{}, false
	}
	return SpanEnclosing(toks[0], toks[len(toks)-1]), true
}

// MatchTokenPrefix checks whether the beginning of lineToks matches target.
// It joins the literals of leading tokens until they match target (ignoring spaces),
// allowing both single-token and multi-token representations (e.g. ["!", "endif"],
// ["end", "note"], ["end", "legend"], ["endlegend"], ["alt"]).
// It returns the number of line tokens matched, and whether a match was found.
func MatchTokenPrefix(lineToks []Token, target string) (int, bool) {
	if len(lineToks) == 0 || target == "" {
		return 0, false
	}

	cleanTarget := strings.ToLower(strings.ReplaceAll(target, " ", ""))
	var accumulated strings.Builder

	for i, tok := range lineToks {
		accumulated.WriteString(strings.ToLower(tok.Literal))
		if accumulated.String() == cleanTarget {
			return i + 1, true
		}
		if accumulated.Len() >= len(cleanTarget) {
			return 0, false
		}
	}
	return 0, false
}

// MatchTokenLine checks whether lineToks matches target exactly with no trailing tokens on the line.
// It returns true only if the entire line of tokens forms the target (ignoring spaces).
func MatchTokenLine(lineToks []Token, target string) bool {
	consumed, ok := MatchTokenPrefix(lineToks, target)
	return ok && consumed == len(lineToks)
}
