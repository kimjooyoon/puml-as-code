package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

func TestParseUnhandled(t *testing.T) {
	newParser := func() *Parser {
		return &Parser{Dialect: dialect.NewGoDialect()}
	}

	t.Run("single-line keywords", func(t *testing.T) {
		t.Run("caption with text", func(t *testing.T) {
			input := "@startuml\ncaption Figure 1: Architecture Overview\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "caption Figure 1: Architecture Overview", uh.Raw)
		})

		t.Run("bare caption without trailing tokens", func(t *testing.T) {
			input := "@startuml\ncaption\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "caption", uh.Raw)
		})

		t.Run("sprite single-line", func(t *testing.T) {
			input := "@startuml\nsprite $my_icon my_icon.png\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "sprite $my_icon my_icon.png", uh.Raw)
		})
	})

	t.Run("fallback single-line directives", func(t *testing.T) {
		t.Run("!define directive", func(t *testing.T) {
			input := "@startuml\n!define FOO(x) class x\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!define FOO(x) class x", uh.Raw)
		})

		t.Run("!global directive", func(t *testing.T) {
			input := "@startuml\n!global $VAR = 42\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!global $VAR = 42", uh.Raw)
		})
	})

	t.Run("blocked directives", func(t *testing.T) {
		t.Run("!function block", func(t *testing.T) {
			input := "@startuml\n!function $my_func($a)\n  !return $a + 1\n!endfunction\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!function $my_func($a)\n  !return $a + 1\n!endfunction", uh.Raw)
		})

		t.Run("!procedure block", func(t *testing.T) {
			input := "@startuml\n!procedure $my_proc()\n  class GeneratedClass\n!endprocedure\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!procedure $my_proc()\n  class GeneratedClass\n!endprocedure", uh.Raw)
		})

		t.Run("!definelong block", func(t *testing.T) {
			input := "@startuml\n!definelong MACRO_NAME\n  class InsideMacro\n!enddefinelong\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!definelong MACRO_NAME\n  class InsideMacro\n!enddefinelong", uh.Raw)
		})

		t.Run("!while block", func(t *testing.T) {
			input := "@startuml\n!while ($val > 0)\n  class C\n!endwhile\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!while ($val > 0)\n  class C\n!endwhile", uh.Raw)
		})

		t.Run("!foreach block", func(t *testing.T) {
			input := "@startuml\n!foreach $item in $list\n  class Item\n!endfor\n@enduml"
			p := newParser()
			diag, err := p.Parse(input)
			require.NoError(t, err)
			require.Len(t, diag.Statements, 3)

			uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
			require.True(t, ok)
			require.Equal(t, "!foreach $item in $list\n  class Item\n!endfor", uh.Raw)
		})
	})

	t.Run("nested !if block", func(t *testing.T) {
		input := `@startuml
!if (A)
  class ClassA
  !if (B)
    class ClassB
  !else
    class ClassElse
  !endif
  class ClassAfterInner
!endif
@enduml`
		p := newParser()
		diag, err := p.Parse(input)
		require.NoError(t, err)
		require.Len(t, diag.Statements, 3)

		uh, ok := diag.Statements[1].(*ast.UnhandledStatement)
		require.True(t, ok)
		expected := `!if (A)
  class ClassA
  !if (B)
    class ClassB
  !else
    class ClassElse
  !endif
  class ClassAfterInner
!endif`
		require.Equal(t, expected, uh.Raw)
	})

	t.Run("unterminated block errors", func(t *testing.T) {
		t.Run("unterminated at EOF", func(t *testing.T) {
			input := "@startuml\n!function $f()\n  line 1\n"
			p := newParser()
			_, err := p.Parse(input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unterminated block statement for !function")
		})

		t.Run("unterminated hitting @enduml", func(t *testing.T) {
			input := "@startuml\n!function $f()\n  line 1\n@enduml"
			p := newParser()
			_, err := p.Parse(input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unterminated block statement for !function")
		})

		t.Run("unterminated inside container hitting scope delimiter", func(t *testing.T) {
			input := "@startuml\npackage mypkg {\n  !if (flag)\n    class Inside\n}\n@enduml"
			p := newParser()
			_, err := p.Parse(input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "unterminated block statement for !if (hit enclosing scope delimiter)")
		})
	})

	t.Run("positive whitelist matching: non-class-diagram syntax is rejected", func(t *testing.T) {
		input := "@startuml\nparticipant Alice\n@enduml"
		p := newParser()
		_, err := p.Parse(input)
		require.Error(t, err)
	})

	t.Run("unhandled statements inside container", func(t *testing.T) {
		input := "@startuml\npackage mypkg {\n  caption Figure in Package\n  sprite $icon icon.png\n}\n@enduml"
		p := newParser()
		diag, err := p.Parse(input)
		require.NoError(t, err)
		require.Len(t, diag.Statements, 3)

		cont, ok := diag.Statements[1].(*ast.Container)
		require.True(t, ok)
		require.Len(t, cont.Statements, 2)

		uh1, ok := cont.Statements[0].(*ast.UnhandledStatement)
		require.True(t, ok)
		require.Equal(t, "caption Figure in Package", uh1.Raw)

		uh2, ok := cont.Statements[1].(*ast.UnhandledStatement)
		require.True(t, ok)
		require.Equal(t, "sprite $icon icon.png", uh2.Raw)
	})

	t.Run("generic opener and closer pattern matching (non-directive and directive)", func(t *testing.T) {
		tok := func(lit string, tt tokenizer.TokenType) tokenizer.Token {
			return tokenizer.Token{Literal: lit, Type: tt}
		}

		// Directives with single and multi-token
		require.True(t, isMatchingOpener([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("if", tokenizer.IDENTIFIER)}, "!if"))
		require.True(t, isMatchingOpener([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("while", tokenizer.IDENTIFIER)}, "!while"))
		require.False(t, isMatchingOpener([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("ifdef", tokenizer.IDENTIFIER)}, "!if"))
		require.False(t, isMatchingOpener([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("elseif", tokenizer.IDENTIFIER)}, "!if"))

		require.True(t, isMatchingCloser([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("endif", tokenizer.IDENTIFIER)}, "!endif"))
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("end", tokenizer.IDENTIFIER), tok("function", tokenizer.IDENTIFIER)}, "!endfunction"))
		require.False(t, isMatchingCloser([]tokenizer.Token{tok("!", tokenizer.EXCLAMATION), tok("elseif", tokenizer.IDENTIFIER)}, "!endif"))

		// Non-directive openers (sequence/activity/state diagram syntax)
		require.True(t, isMatchingOpener([]tokenizer.Token{tok("alt", tokenizer.IDENTIFIER), tok("x > 0", tokenizer.IDENTIFIER)}, "alt"))
		require.True(t, isMatchingOpener([]tokenizer.Token{tok("loop", tokenizer.IDENTIFIER), tok("1000", tokenizer.NUMBER)}, "loop"))
		require.False(t, isMatchingOpener([]tokenizer.Token{tok("alternative", tokenizer.IDENTIFIER)}, "alt"))

		// Non-directive closers (single word, two words, compound)
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("end", tokenizer.IDENTIFIER)}, "end"))
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("end", tokenizer.IDENTIFIER), tok("note", tokenizer.IDENTIFIER)}, "end note"))
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("endnote", tokenizer.IDENTIFIER)}, "end note"))
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("end", tokenizer.IDENTIFIER), tok("legend", tokenizer.IDENTIFIER)}, "endlegend"))
		require.True(t, isMatchingCloser([]tokenizer.Token{tok("endlegend", tokenizer.IDENTIFIER)}, "endlegend"))

		// Distinct closers: "end" must NOT match compound closers like "end note"
		require.False(t, isMatchingCloser([]tokenizer.Token{tok("end", tokenizer.IDENTIFIER), tok("note", tokenizer.IDENTIFIER)}, "end"))
		require.False(t, isMatchingCloser([]tokenizer.Token{tok("end", tokenizer.IDENTIFIER), tok("legend", tokenizer.IDENTIFIER)}, "end"))
	})
}
