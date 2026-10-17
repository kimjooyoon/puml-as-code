package dialect

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/tokenizer"
)

// LaxDialect implements [Dialect] with a lax parsing strategy.
// It is unfit for code generation as doens't provide enough information
// to determine structure of the AST.
// It never returns ErrParsingDialect and always returns a valid ast,
// though sparce in information.
type LaxDialect struct{}

// Name implements [Dialect].
func (l LaxDialect) Name() string {
	return "lax"
}

// ParseField implements [Dialect].
func (l LaxDialect) ParseField(toks []tokenizer.Token) (ast.Field, error) {
	return &LaxField{
		Text: StringifyTokenSlice(toks),
	}, nil
}

// ParseMethod implements [Dialect].
func (l LaxDialect) ParseMethod(toks []tokenizer.Token) (ast.Method, error) {
	return &LaxMethod{
		Text: StringifyTokenSlice(toks),
	}, nil
}

var _ Dialect = LaxDialect{}

func StringifyTokenSlice(toks []tokenizer.Token) string {
	if len(toks) == 0 {
		return ""
	}

	firstTok := toks[0]
	lastTok := toks[len(toks)-1]
	bufLen := lastTok.Span.End.Offset - firstTok.Span.Start.Offset

	var sb strings.Builder
	sb.Grow(int(bufLen) + len(toks))

	for i, tok := range toks {
		sb.WriteString(tok.Literal)

		if i < len(toks)-1 && needsSpaceBetween(toks[i], toks[i+1]) {
			sb.WriteRune(' ')
		}
	}

	return sb.String()
}

func needsSpaceBetween(curr, next tokenizer.Token) bool {
	// 1. Never add spaces before commas, semicolons, colons, or closing delimiters
	switch next.Type {
	case tokenizer.COMMA, tokenizer.SEMICOLON, tokenizer.COLON, tokenizer.RPAREN, tokenizer.RBRACKET:
		return false
	}

	// 2. Never add space after opening delimiters or dots
	switch curr.Type {
	case tokenizer.LPAREN, tokenizer.LBRACKET, tokenizer.DOT:
		return false
	}

	// 3. Always add space after commas
	if curr.Type == tokenizer.COMMA {
		return true
	}

	// 4. Always add space after colons (unless C++ :: scope resolution)
	if curr.Type == tokenizer.COLON && next.Type != tokenizer.COLON {
		return true
	}

	// 5. Always add space around operators (=, ==)
	if curr.Type == tokenizer.EQUALS || next.Type == tokenizer.EQUALS {
		return true
	}

	// 6. Separate adjacent word-like tokens: "id string", "int count"
	if isWordLike(curr.Type) && isWordLike(next.Type) {
		return true
	}

	// 7. Space after closing paren if followed by return type: ") error", ") *User", ") (int, error)", ") []byte"
	if curr.Type == tokenizer.RPAREN {
		if isWordLike(next.Type) || next.Type == tokenizer.ASTERISK || next.Type == tokenizer.LPAREN || next.Type == tokenizer.LBRACKET {
			return true
		}
	}

	// 8. Space before pointer in field declarations: "user *User"
	if isWordLike(curr.Type) && next.Type == tokenizer.ASTERISK {
		return true
	}

	// 9. Space after closing angle when followed by an identifier: "List<String> items"
	if curr.Type == tokenizer.RANGLE && isWordLike(next.Type) {
		return true
	}

	// 10. Respect source whitespace between tokens if present, unless explicitly avoided above
	endCurr := curr.Span.End.Offset
	if next.Span.Start.Offset > endCurr {
		// Avoid space before array brackets in types like "int[]" or "string[]"
		if isWordLike(curr.Type) && next.Type == tokenizer.LBRACKET {
			return true
		}
		// Avoid space between pointer and type: "*User"
		if curr.Type == tokenizer.ASTERISK {
			return false
		}
		return true
	}

	return false
}

func isWordLike(t tokenizer.TokenType) bool {
	return t == tokenizer.IDENTIFIER || t == tokenizer.NUMBER || t == tokenizer.STRING
}
