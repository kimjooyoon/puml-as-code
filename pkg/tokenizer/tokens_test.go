package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// assertTokenType checks token type with descriptive error message
func assertTokenType(t *testing.T, expected, actual TokenType) {
	t.Helper()
	require.Equal(t, expected, actual, "expected %s, got %s", expected.String(), actual.String())
}

func TestResolveUnambiguousToken_EOF(t *testing.T) {
	l := NewLexer("")
	tok, resolved := ResolveUnambiguousToken(l)

	require.True(t, resolved)
	assertTokenType(t, EOF, tok.Type)
}

func TestResolveUnambiguousToken_String(t *testing.T) {
	l := NewLexer(`"hello world"`)
	tok, resolved := ResolveUnambiguousToken(l)

	require.True(t, resolved)
	assertTokenType(t, STRING, tok.Type)
	require.Equal(t, `"hello world"`, tok.Literal)
}

func TestResolveUnambiguousToken_Physical(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected TokenType
	}{
		{"left bracket", "[", LBRACKET},
		{"right bracket", "]", RBRACKET},
		{"left paren", "(", LPAREN},
		{"right paren", ")", RPAREN},
		{"left brace", "{", LBRACE},
		{"right brace", "}", RBRACE},
		{"left angle", "<", LANGLE},
		{"right angle", ">", RANGLE},
		{"comma", ",", COMMA},
		{"semicolon", ";", SEMICOLON},
		{"colon", ":", COLON},
		{"dot", ".", DOT},
		{"equals", "=", EQUALS},
		{"plus", "+", PLUS},
		{"hyphen", "-", DASH},
		{"tilde", "~", TILDE},
		{"hash", "#", HASH},
		{"vbar", "|", PIPE},
		{"asterisk", "*", ASTERISK},
		{"slash", "/", SLASH},
		{"backslash", "\\", BACKSLASH},
		{"caret", "^", CARET},
		{"dollar", "$", DOLLAR},
		{"percent", "%", PERCENT},
		{"at", "@", AT},
		{"exclamation", "!", EXCLAMATION},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := NewLexer(tc.input)
			tok, resolved := ResolveUnambiguousToken(l)

			require.True(t, resolved)
			assertTokenType(t, tc.expected, tok.Type)
			require.Equal(t, tc.input, tok.Literal)
		})
	}
}

func TestResolveUnambiguousToken_Comment(t *testing.T) {
	l := NewLexer("' this is a comment")
	tok, resolved := ResolveUnambiguousToken(l)

	require.True(t, resolved)
	assertTokenType(t, COMMENT, tok.Type)
}

func TestResolveUnambiguousToken_Unresolved(t *testing.T) {
	l := NewLexer("foo")
	_, resolved := ResolveUnambiguousToken(l)

	require.False(t, resolved, "identifiers should not be resolved as unambiguous")
}

func TestResolveAmbiguousToken_Identifier(t *testing.T) {
	l := NewLexer("class")
	tok := ResolveAmbiguousToken(l)
	assertTokenType(t, IDENTIFIER, tok.Type)
	require.Equal(t, "class", tok.Literal)

	l = NewLexer("foo_bar")
	tok = ResolveAmbiguousToken(l)
	assertTokenType(t, IDENTIFIER, tok.Type)
	require.Equal(t, "foo_bar", tok.Literal)
}

func TestResolveAmbiguousToken_Number(t *testing.T) {
	l := NewLexer("123")
	tok := ResolveAmbiguousToken(l)
	assertTokenType(t, NUMBER, tok.Type)
	require.Equal(t, "123", tok.Literal)

	l = NewLexer("123.456")
	tok = ResolveAmbiguousToken(l)
	assertTokenType(t, NUMBER, tok.Type)
	require.Equal(t, "123.456", tok.Literal)

	l = NewLexer("0x123")
	tok = ResolveAmbiguousToken(l)
	assertTokenType(t, NUMBER, tok.Type)
	require.Equal(t, "0x123", tok.Literal)

	l = NewLexer("0o123")
	tok = ResolveAmbiguousToken(l)
	assertTokenType(t, NUMBER, tok.Type)
	require.Equal(t, "0o123", tok.Literal)

	l = NewLexer("0b101")
	tok = ResolveAmbiguousToken(l)
	assertTokenType(t, NUMBER, tok.Type)
	require.Equal(t, "0b101", tok.Literal)
}

func TestMatchTokenLineAndPrefix(t *testing.T) {
	tok := func(lit string, tt TokenType) Token {
		return Token{Literal: lit, Type: tt}
	}

	t.Run("MatchTokenLine", func(t *testing.T) {
		// Exact line match
		require.True(t, MatchTokenLine([]Token{tok("end", IDENTIFIER), tok("note", IDENTIFIER)}, "end note"))
		require.True(t, MatchTokenLine([]Token{tok("endnote", IDENTIFIER)}, "end note"))
		require.True(t, MatchTokenLine([]Token{tok("!", EXCLAMATION), tok("endif", IDENTIFIER)}, "!endif"))
		require.True(t, MatchTokenLine([]Token{tok("end", IDENTIFIER)}, "end"))

		// Trailing tokens must fail MatchTokenLine
		require.False(t, MatchTokenLine([]Token{tok("end", IDENTIFIER), tok("note", IDENTIFIER)}, "end"))
		require.False(t, MatchTokenLine([]Token{tok("end", IDENTIFIER), tok("note", IDENTIFIER), tok("foo", IDENTIFIER)}, "end note"))
		require.False(t, MatchTokenLine([]Token{tok("!", EXCLAMATION), tok("endif", IDENTIFIER), tok("label", IDENTIFIER)}, "!endif"))
	})

	t.Run("MatchTokenPrefix", func(t *testing.T) {
		// Prefix match allowing trailing tokens
		consumed, ok := MatchTokenPrefix([]Token{tok("!", EXCLAMATION), tok("if", IDENTIFIER), tok("(", LPAREN)}, "!if")
		require.True(t, ok)
		require.Equal(t, 2, consumed)

		consumed, ok = MatchTokenPrefix([]Token{tok("alt", IDENTIFIER), tok("condition", IDENTIFIER)}, "alt")
		require.True(t, ok)
		require.Equal(t, 1, consumed)

		// Partial token name must not match
		_, ok = MatchTokenPrefix([]Token{tok("alternative", IDENTIFIER)}, "alt")
		require.False(t, ok)
		_, ok = MatchTokenPrefix([]Token{tok("!", EXCLAMATION), tok("ifdef", IDENTIFIER)}, "!if")
		require.False(t, ok)
	})
}

func TestSourceFidelity(t *testing.T) {
	// Helper to collect all tokens safely up to EOF with an iteration guard against infinite loops
	tokenizeAll := func(t *testing.T, input string) []Token {
		t.Helper()
		l := NewLexer(input)
		var tokens []Token
		const maxIterations = 200
		for i := range maxIterations {
			tok := l.Emit()
			tokens = append(tokens, tok)
			if tok.Type == EOF {
				break
			}
			if i == maxIterations-1 {
				t.Fatalf("possible infinite loop: exceeded %d iterations on input %q", maxIterations, input)
			}
		}
		return tokens
	}

	assertTokenFidelity := func(t *testing.T, input string, tok Token, expectedType TokenType, expectedLit, expectedRaw string) {
		t.Helper()
		require.Equal(t, expectedType, tok.Type, "token type mismatch")
		require.Equal(t, expectedLit, tok.Literal, "semantic literal mismatch")

		// Verify source fidelity invariant: source[Span.Start.Offset : Span.End.Offset] == expectedRaw
		require.LessOrEqual(t, int(tok.Span.End.Offset), len(input), "token End.Offset exceeds input bounds")
		runes := []rune(input)
		rawSlice := string(runes[tok.Span.Start.Offset:tok.Span.End.Offset])
		require.Equal(t, expectedRaw, rawSlice, "raw source slice mismatch")
	}

	t.Run("strings", func(t *testing.T) {
		t.Run("basic quoted string", func(t *testing.T) {
			input := `"hello world"`
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], STRING, `"hello world"`, `"hello world"`)
		})

		t.Run("string with escape sequences", func(t *testing.T) {
			input := `"hello \"world\""`
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], STRING, `"hello \"world\""`, `"hello \"world\""`)
		})

		t.Run("empty string", func(t *testing.T) {
			input := `""`
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], STRING, `""`, `""`)
		})

		t.Run("non-ascii unicode string", func(t *testing.T) {
			input := `"Привіт"`
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1, "token count mismatch")
			assertTokenFidelity(t, input, toks[0], STRING, `"Привіт"`, `"Привіт"`)
			require.Equal(t, uint(len([]rune(input))), toks[0].Span.End.Offset, "length mismatch")
		})
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("single line comment with whitespace", func(t *testing.T) {
			input := "'   some comment text\n"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], COMMENT, "some comment text", "'   some comment text")
		})

		t.Run("multiline block comment", func(t *testing.T) {
			input := "/' line 1\n   line 2\n   line 3 '/"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], COMMENT, " line 1\n   line 2\n   line 3 ", input)
			require.Equal(t, uint(0), toks[0].Span.Start.Line, "start line should be 0")
			require.Equal(t, uint(2), toks[0].Span.End.Line, "end line should be 2 for 3-line block comment")
			require.Equal(t, uint(2), toks[0].EndPos().Line, "EndPos().Line must reflect multiline end line")
		})
	})

	t.Run("identifiers and numbers", func(t *testing.T) {
		t.Run("identifier", func(t *testing.T) {
			input := "myIdentifier"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], IDENTIFIER, "myIdentifier", "myIdentifier")
		})

		t.Run("number", func(t *testing.T) {
			input := "12345"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], NUMBER, "12345", "12345")
		})
	})

	t.Run("single character tokens and operators", func(t *testing.T) {
		t.Run("punctuation", func(t *testing.T) {
			input := "{:+"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 3)
			assertTokenFidelity(t, input, toks[0], LBRACE, "{", "{")
			assertTokenFidelity(t, input, toks[1], COLON, ":", ":")
			assertTokenFidelity(t, input, toks[2], PLUS, "+", "+")
		})

		t.Run("standalone slash operator", func(t *testing.T) {
			input := "/ foo"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 2)
			assertTokenFidelity(t, input, toks[0], SLASH, "/", "/")
			require.Equal(t, IDENTIFIER, toks[1].Type, "next token should be IDENTIFIER")
		})
	})

	t.Run("newlines and crlf", func(t *testing.T) {
		t.Run("standard newline", func(t *testing.T) {
			input := "\n"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], NEWLINE, "\n", "\n")
		})

		t.Run("crlf newline", func(t *testing.T) {
			input := "\r\n"
			toks := tokenizeAll(t, input)
			require.GreaterOrEqual(t, len(toks), 1)
			assertTokenFidelity(t, input, toks[0], NEWLINE, "\n", "\r\n")
		})
	})
}
