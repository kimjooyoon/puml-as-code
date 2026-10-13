package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func TestParseLayoutDirectives(t *testing.T) {
	newParser := func() *Parser {
		return &Parser{Dialect: dialect.NewGoDialect()}
	}

	t.Run("header single line forms", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected ast.TextBlock
		}{
			{
				name:     "plain header",
				input:    "@startuml\nheader My Header\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "My Header"},
			},
			{
				name:     "left header",
				input:    "@startuml\nleft header My Left Header\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "My Left Header", HorizontalAlignment: "left"},
			},
			{
				name:     "center header",
				input:    "@startuml\ncenter header My Centered Header\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "My Centered Header", HorizontalAlignment: "center"},
			},
			{
				name:     "header right (trailing alignment)",
				input:    "@startuml\nheader right My Right Header\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "My Right Header", HorizontalAlignment: "right"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				p := newParser()
				diag, err := p.Parse(tc.input)
				require.NoError(t, err)
				require.Len(t, diag.Statements, 3) // bounds(start), header, bounds(end)

				tb, ok := diag.Statements[1].(*ast.TextBlock)
				require.True(t, ok)
				require.Equal(t, tc.expected.Kind, tb.Kind)
				require.Equal(t, tc.expected.Text, tb.Text)
				require.Equal(t, tc.expected.HorizontalAlignment, tb.HorizontalAlignment)
				require.Equal(t, tc.expected.VerticalAlignment, tb.VerticalAlignment)
			})
		}
	})

	t.Run("header multi-line block forms", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected ast.TextBlock
		}{
			{
				name:     "plain block with endheader",
				input:    "@startuml\nheader\n  line 1\n  line 2\nendheader\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "  line 1\n  line 2"},
			},
			{
				name:     "block with end header space",
				input:    "@startuml\nheader\n  line 1\nend header\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "  line 1"},
			},
			{
				name:     "preceding alignment block",
				input:    "@startuml\nleft header\n  line 1\nendheader\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "  line 1", HorizontalAlignment: "left"},
			},
			{
				name:     "trailing alignment block",
				input:    "@startuml\nheader center\n  line 1\nendheader\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockHeader, Text: "  line 1", HorizontalAlignment: "center"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				p := newParser()
				diag, err := p.Parse(tc.input)
				require.NoError(t, err)
				require.Len(t, diag.Statements, 3)

				tb, ok := diag.Statements[1].(*ast.TextBlock)
				require.True(t, ok)
				require.Equal(t, tc.expected.Kind, tb.Kind)
				require.Equal(t, tc.expected.Text, tb.Text)
				require.Equal(t, tc.expected.HorizontalAlignment, tb.HorizontalAlignment)
				require.Equal(t, tc.expected.VerticalAlignment, tb.VerticalAlignment)
			})
		}
	})

	t.Run("footer forms", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected ast.TextBlock
		}{
			{
				name:     "single line footer",
				input:    "@startuml\nfooter Confidential\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockFooter, Text: "Confidential"},
			},
			{
				name:     "center footer single line",
				input:    "@startuml\ncenter footer Page 1\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockFooter, Text: "Page 1", HorizontalAlignment: "center"},
			},
			{
				name:     "footer right single line",
				input:    "@startuml\nfooter right Page 1\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockFooter, Text: "Page 1", HorizontalAlignment: "right"},
			},
			{
				name:     "block footer endfooter",
				input:    "@startuml\nfooter\n  Draft\n  v1.0\nendfooter\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockFooter, Text: "  Draft\n  v1.0"},
			},
			{
				name:     "block footer end footer",
				input:    "@startuml\nfooter\n  Draft\nend footer\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockFooter, Text: "  Draft"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				p := newParser()
				diag, err := p.Parse(tc.input)
				require.NoError(t, err)
				require.Len(t, diag.Statements, 3)

				tb, ok := diag.Statements[1].(*ast.TextBlock)
				require.True(t, ok)
				require.Equal(t, tc.expected.Kind, tb.Kind)
				require.Equal(t, tc.expected.Text, tb.Text)
				require.Equal(t, tc.expected.HorizontalAlignment, tb.HorizontalAlignment)
				require.Equal(t, tc.expected.VerticalAlignment, tb.VerticalAlignment)
			})
		}
	})

	t.Run("legend forms", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected ast.TextBlock
		}{
			{
				name:     "legend block with trailing alignment",
				input:    "@startuml\nlegend right\n  Short legend\n  multiple lines\nendlegend\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "  Short legend\n  multiple lines", HorizontalAlignment: "right"},
			},
			{
				name:     "legend block with end legend space",
				input:    "@startuml\nlegend\n  Short legend\nend legend\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "  Short legend"},
			},
			{
				name:     "legend block with preceding alignment",
				input:    "@startuml\nright legend\n  Short legend\nendlegend\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "  Short legend", HorizontalAlignment: "right"},
			},
			{
				name:     "legend single line",
				input:    "@startuml\nlegend Single line note\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "Single line note"},
			},
			{
				name:     "legend with vertical and horizontal alignment",
				input:    "@startuml\nlegend top left\n  Top left legend\nendlegend\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "  Top left legend", VerticalAlignment: "top", HorizontalAlignment: "left"},
			},
			{
				name:     "legend with preceding vertical alignment",
				input:    "@startuml\nbottom legend\n  Bottom legend\nendlegend\n@enduml",
				expected: ast.TextBlock{Kind: ast.BlockLegend, Text: "  Bottom legend", VerticalAlignment: "bottom"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				p := newParser()
				diag, err := p.Parse(tc.input)
				require.NoError(t, err)
				require.Len(t, diag.Statements, 3)

				tb, ok := diag.Statements[1].(*ast.TextBlock)
				require.True(t, ok)
				require.Equal(t, tc.expected.Kind, tb.Kind)
				require.Equal(t, tc.expected.Text, tb.Text)
				require.Equal(t, tc.expected.HorizontalAlignment, tb.HorizontalAlignment)
				require.Equal(t, tc.expected.VerticalAlignment, tb.VerticalAlignment)
			})
		}
	})

	t.Run("caption single line", func(t *testing.T) {
		input := "@startuml\ncaption Figure 1: Architecture Overview\n@enduml"
		p := newParser()
		diag, err := p.Parse(input)
		require.NoError(t, err)
		require.Len(t, diag.Statements, 3)

		uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
		require.True(t, ok)
		require.Equal(t, "caption Figure 1: Architecture Overview", uh.Raw)
	})

	t.Run("title single line and block", func(t *testing.T) {
		t.Run("single line title", func(t *testing.T) {
			input := "@startuml\ntitle My Great Diagram\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Equal(t, "My Great Diagram", p.ast.Title)

			tb, ok := diag.Statements[1].(*ast.TextBlock)
			require.True(t, ok)
			require.Equal(t, ast.BlockTitle, tb.Kind)
			require.Equal(t, "My Great Diagram", tb.Text)
		})

		t.Run("title rejects alignment", func(t *testing.T) {
			input := "@startuml\ncenter title Centered Title\n@enduml"
			p := newParser()
			_, err := p.Parse(input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "Title alignment not supported")
		})

		t.Run("block title", func(t *testing.T) {
			input := "@startuml\ntitle\n  Line 1\n  Line 2\nend title\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Equal(t, "  Line 1\n  Line 2", p.ast.Title)

			tb, ok := diag.Statements[1].(*ast.TextBlock)
			require.True(t, ok)
			require.Equal(t, ast.BlockTitle, tb.Kind)
			require.Equal(t, "  Line 1\n  Line 2", tb.Text)
		})
	})

	t.Run("diagram direction is not confused with layout alignment", func(t *testing.T) {
		input := "@startuml\nleft to right direction\nclass A\n@enduml"
		p := newParser()
		diag, err := p.Parse(input)
		require.NoError(t, err)

		dir, ok := diag.Statements[1].(*ast.DirectionCommand)
		require.True(t, ok)
		require.Equal(t, ast.LeftToRightDirection, dir.Direction)
	})

	t.Run("layout directive inside container", func(t *testing.T) {
		input := `@startuml
package mypkg {
  legend
    Package legend
  endlegend
  class Foo
}
@enduml`
		p := newParser()
		diag, err := p.Parse(input)
		require.NoError(t, err)

		cont, ok := diag.Statements[1].(*ast.Container)
		require.True(t, ok)
		require.Len(t, cont.Statements, 2)

		tb, ok := cont.Statements[0].(*ast.TextBlock)
		require.True(t, ok)
		require.Equal(t, ast.BlockLegend, tb.Kind)
		require.Equal(t, "    Package legend", tb.Text)

		ent, ok := cont.Statements[1].(*ast.Entity)
		require.True(t, ok)
		require.Equal(t, "Foo", ent.Identifier)
	})

	t.Run("unterminated block errors", func(t *testing.T) {
		t.Run("unterminated header at EOF", func(t *testing.T) {
			input := "header\n  unclosed header\n"
			p := Parser{
				stream: tokenizer.NewTokenStream(input),
			}
			kwTok := p.stream.Emit()
			_, err := p.parseLayoutStatement(kwTok, nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unterminated block statement for header")
		})
	})
}
