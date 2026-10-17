package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func TestParseNote(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *ast.Note
		expectErr   bool
		errContains string
	}{
		// --- Relative Notes (parseReltiveNote) ---
		{
			name:  "relative note left with colon",
			input: "note left : some note text\n",
			want: &ast.Note{
				Kind:      ast.NoteTargeted,
				Text:      "some note text",
				Direction: ast.DirectionLeft,
			},
		},
		{
			name:  "relative note right of target with colon",
			input: "note right of MyClass : some note text\n",
			want: &ast.Note{
				Kind:      ast.NoteTargeted,
				Text:      "some note text",
				Direction: ast.DirectionRight,
				Target:    identRefPtr("MyClass"),
			},
		},
		{
			name:  "relative note top of target with color and colon",
			input: "note top of MyClass #green : some note text\n",
			want: &ast.Note{
				Kind:      ast.NoteTargeted,
				Text:      "some note text",
				Direction: ast.DirectionTop,
				Target:    identRefPtr("MyClass"),
				Color:     "#green",
			},
		},
		{
			name:  "relative note bottom on link with colon",
			input: "note bottom on link : some note text\n",
			want: &ast.Note{
				Kind:      ast.NoteLink,
				Text:      "some note text",
				Direction: ast.DirectionBottom,
				Target:    identRefPtr("link"),
			},
		},
		{
			name:  "relative note left with multiline",
			input: "note left\nsome note text\nend note\n",
			want: &ast.Note{
				Kind:      ast.NoteTargeted,
				Text:      "some note text",
				Direction: ast.DirectionLeft,
			},
		},
		{
			name:  "relative note left of target multiline",
			input: "note left of MyClass\nsome note text\nend note",
			want: &ast.Note{
				Kind:      ast.NoteTargeted,
				Text:      "some note text",
				Direction: ast.DirectionLeft,
				Target:    identRefPtr("MyClass"),
			},
		},
		{
			name:        "relative note invalid identifier after direction",
			input:       "note left invalid : text\n",
			expectErr:   true,
			errContains: "Unexpected identifier after direction",
		},
		{
			name:        "relative note expected identifier for target",
			input:       "note left of : text\n",
			expectErr:   true,
			errContains: "Expected ':' or newline after note definition",
		},
		{
			name:        "relative note unexpected identifier for link target",
			input:       "note left on MyClass : text",
			expectErr:   true,
			errContains: "Unexpected identifier for a note link target",
		},

		// --- Inline Alias Notes (parseInlineAliasNote) ---
		{
			name:  "inline alias note",
			input: `note "some note text" as N1`,
			want: &ast.Note{
				Kind:       ast.NoteAlias,
				Text:       "some note text",
				Identifier: identRef("N1"),
			},
		},
		{
			name:  "inline alias note with color",
			input: `note "some note text" as N1 #blue`,
			want: &ast.Note{
				Kind:       ast.NoteAlias,
				Text:       "some note text",
				Identifier: identRef("N1"),
				Color:      "#blue",
			},
		},
		{
			name:        "inline alias note missing as",
			input:       `note "some note text" N1`,
			expectErr:   true,
			errContains: "Expected alias keyword after note text",
		},
		{
			name:        "inline alias note missing identifier",
			input:       `note "some note text" as`,
			expectErr:   true,
			errContains: "Expected identifier after alias keyword",
		},

		// --- Multiline Alias Notes (parseMultilineAliasNote) ---
		{
			name:        "multiline alias note with colon (invalid)",
			input:       "note as N1 : some note text",
			expectErr:   true,
			errContains: "Expected newline after alias keyword",
		},
		{
			name:        "multiline alias note with colon and color (invalid)",
			input:       "note as N1 #red : some note text",
			expectErr:   true,
			errContains: "Expected newline after alias keyword",
		},
		{
			name:        "multiline alias note missing identifier",
			input:       "note as",
			expectErr:   true,
			errContains: "Expected identifier after alias keyword",
		},
		{
			name:  "multiline alias note with newline",
			input: "note as N1\nsome note text\nend note",
			want: &ast.Note{
				Kind:       ast.NoteAlias,
				Text:       "some note text",
				Identifier: identRef("N1"),
			},
		},

		// --- Link Notes (parseLinkNote) ---
		{
			name:  "link note on link with colon",
			input: "note on link : some note text\n",
			want: &ast.Note{
				Kind:   ast.NoteLink,
				Text:   "some note text",
				Target: identRefPtr("link"),
			},
		},
		{
			name:  "link note on link multiline",
			input: "note on link\nsome note text\nend note",
			want: &ast.Note{
				Kind:   ast.NoteLink,
				Text:   "some note text",
				Target: identRefPtr("link"),
			},
		},
		{
			name:        "unexpected identifier after note",
			input:       "note invalid link : text",
			expectErr:   true,
			errContains: "expected direction, string, note position or alias after 'note'",
		},
		{
			name:        "link note expected link after note on",
			input:       "note on invalid : text",
			expectErr:   true,
			errContains: "Expected 'link' after 'note on'",
		},

		// --- Generic Parser / parseNoteBody Errors ---
		{
			name:        "parseNote unexpected starting token",
			input:       "note class : text",
			expectErr:   true,
			errContains: "expected direction, string, note position or alias after 'note'",
		},
		{
			name:        "parseNoteBody unexpected body definition token",
			input:       "note left class : text",
			expectErr:   true,
			errContains: "Unexpected identifier after direction",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream: tokenizer.NewTokenStream(tc.input),
				ast:    &ast.Diagram{},
			}
			tok := p.stream.Emit()
			require.Equal(t, keyword.Note, keyword.Classify(tok.Literal), "First token must be 'note' for test input: %q", tc.input)

			note, err := p.parseNote(tok)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assertASTEqual(t, *tc.want, note)
			}
		})
	}
}
