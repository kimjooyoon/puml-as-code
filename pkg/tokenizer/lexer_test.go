package tokenizer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadIdentifier(t *testing.T) {
	lex := NewLexer("tag13")
	require.Equal(t, "tag13", lex.readIdentifier())

	lex = NewLexer("Foo\\-bar")
	require.Equal(t, "Foo-bar", lex.readIdentifier())

	lex = NewLexer("Foo123")
	require.Equal(t, "Foo123", lex.readIdentifier())
}

func TestReadNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Valid cases
		{"Integer", "123", "123", false},
		{"Decimal", "12.34", "12.34", false},
		{"LeadingZeros", "007", "007", false},
		{"HexLowercase", "0x1a2b", "0x1a2b", false},
		{"HexUppercase", "0X1A2B", "0X1A2B", false},
		{"Binary", "0b1010", "0b1010", false},
		{"Octal", "0o755", "0o755", false},
		{"Scientific", "1e10", "1e10", false},
		{"ScientificNegative", "1.2e-5", "1.2e-5", false},
		{"ScientificPositive", "1E+5", "1E+5", false},
		{"LeadingDecimalPoint", ".123", ".123", false},
		{"TrailingDecimalPoint", "123.", "123.", false},

		// Error / Edge cases
		{"MultipleDots", "1.2.3", "", true},
		{"InvalidHex", "0x12G", "", true},
		{"InvalidBinary", "0b102", "", true},
		{"InvalidOctal", "0o789", "", true},
		{"IncompleteHex", "0x", "", true},
		{"IncompleteScientific", "1e", "", true},
		{"DigitAfterScientific", "1e10.5", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			lit, err := lex.readNumber()
			if tt.wantErr {
				require.Error(t, err, "Expected error for input: %s", tt.input)
			} else {
				require.NoError(t, err, "Unexpected error for input: %s", tt.input)
				require.Equal(t, tt.expected, lit)
			}
		})
	}
}

func TestReadString(t *testing.T) {
	lex := NewLexer("\"a\\\"b\"")
	require.Equal(t, "\"a\\\"b\"", lex.readString())
}

func TestReadLineComment(t *testing.T) {
	lex := NewLexer("' comment here\n")
	line := lex.readLineComment()
	require.Equal(t, "comment here", line)
}

// func TestSetSeparator(t *testing.T) {
// 	lex := NewLexer("set separator ::\nclass A::B")
//
// 	// set
// 	tok := lex.Emit()
// 	require.Equal(t, SET_CMD, tok.Type)
//
// 	// separator
// 	tok = lex.Emit()
// 	require.Equal(t, IDENTIFIER, tok.Type)
// 	require.Equal(t, "separator", tok.Literal)
//
// 	// ::
// 	tok = lex.Emit()
// 	require.Equal(t, IDENTIFIER, tok.Type)
// 	require.Equal(t, "::", tok.Literal)
// 	require.Equal(t, []rune("::"), lex.packageSeparator)
//
// 	// NEWLINE
// 	tok = lex.Emit()
// 	require.Equal(t, NEWLINE, tok.Type)
//
// 	// class
// 	tok = lex.Emit()
// 	require.Equal(t, CLASS, tok.Type)
//
// 	// A
// 	tok = lex.Emit()
// 	require.Equal(t, IDENTIFIER, tok.Type)
// 	require.Equal(t, "A", tok.Literal)
//
// 	// :: (as separator)
// 	tok = lex.Emit()
// 	require.Equal(t, PACKAGE_SEPARATOR, tok.Type)
// 	require.Equal(t, "::", tok.Literal)
//
// 	// B
// 	tok = lex.Emit()
// 	require.Equal(t, IDENTIFIER, tok.Type)
// 	require.Equal(t, "B", tok.Literal)
// }
