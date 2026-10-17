package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func TestParseEntity(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		kwType keyword.KeywordKind
		want   ast.Statement
	}{
		{
			name:   "simple class",
			input:  "class MyClass",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
			},
		},
		{
			name:   "abstract class",
			input:  "abstract class MyAbstractClass",
			kwType: keyword.Abstract,
			want: &ast.Entity{
				Identifier: identRef("MyAbstractClass"),
				Kind:       ast.EntityAbstractClass,
			},
		},
		{
			name:   "class with alias",
			input:  "class MyClass as \"MC\"",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Alias:      tokString("MC"),
				Kind:       ast.EntityClass,
			},
		},
		{
			name:   "class with stereotype",
			input:  "class MyClass <<Service>>",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
				Stereotype: "Service",
			},
		},
		{
			name:   "class with color",
			input:  "class MyClass #FF0000",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
				Color:      "#FF0000",
			},
		},
		{
			name:   "class with empty body",
			input:  "class MyClass {\n}",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
			},
		},
		{
			name:   "class def with all modifiers",
			input:  "class MyClass as \"MC\" <T> <<Database>> #FF0000",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Alias:      tokString("MC"),
				Kind:       ast.EntityClass,
				Generic:    "T",
				Stereotype: "Database",
				Color:      "#FF0000",
			},
		},
		{
			name:   "class with field member",
			input:  "class MyClass {\n  +field int\n}",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
				Members: []ast.Member{
					&ast.FieldDeclaration{
						Visibility: ast.VisibilityPublic,
						Field: &dialect.GoField{
							Name: "field",
							Type: dialect.NamedRef("int"),
						},
					},
				},
			},
		},
		{
			name:   "class with package separator",
			input:  "class net.http.Client\n",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("Client", "net", "http"),
				Kind:       ast.EntityClass,
			},
		},
		{
			name:   "sketch field inside body",
			input:  "class MyClass {\n  untypedField\n}",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("MyClass"),
				Kind:       ast.EntityClass,
				Members: []ast.Member{
					&ast.FieldDeclaration{
						Field: &dialect.GoField{
							Name: "untypedField",
							Type: nil,
						},
					},
				},
			},
		},
		{
			name:   "class with generic",
			input:  "class List<T>",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("List"),
				Kind:       ast.EntityClass,
				Generic:    "T",
			},
		},
		{
			name:   "class with multiple generics",
			input:  "class Map<K, V>",
			kwType: keyword.Class,
			want: &ast.Entity{
				Identifier: identRef("Map"),
				Kind:       ast.EntityClass,
				Generic:    "K, V",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			tok := p.stream.Emit()
			require.Equal(t, tc.kwType, keyword.Classify(tok.Literal))

			got, err := p.parseEntity(tok)
			require.NoError(t, err)
			assertASTEqual(t, tc.want, got)
		})
	}
}

func TestParseEntity_Errors(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		kwType keyword.KeywordKind
	}{
		{
			name:   "class with invalid alias (not a string)",
			input:  "class MyClass as MC",
			kwType: keyword.Class,
		},
		{
			name:   "invalid alias",
			input:  "class MyClass as",
			kwType: keyword.Class,
		},
		{
			name:   "unclosed stereotype",
			input:  "class MyClass <<Stereo",
			kwType: keyword.Class,
		},
		{
			name:   "unclosed body",
			input:  "class MyClass {",
			kwType: keyword.Class,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			tok := p.stream.Emit()
			require.Equal(t, tc.kwType, keyword.Classify(tok.Literal))

			got, err := p.parseEntity(tok)
			require.Error(t, err)
			require.Nil(t, got)
		})
	}
}

func TestParseEntity_Recovery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		kwType   keyword.KeywordKind
		checkBad func(t *testing.T, stmt ast.Statement, p *Parser)
	}{
		{
			name:   "invalid member inside body",
			input:  "class MyClass {\n  name : invalid\n}",
			kwType: keyword.Class,
			checkBad: func(t *testing.T, stmt ast.Statement, p *Parser) {
				ent, ok := stmt.(*ast.Entity)
				require.True(t, ok, "expected statement to be *ast.Entity")
				require.Len(t, ent.Members, 1)
				bad, ok := ent.Members[0].(*ast.BadMember)
				require.True(t, ok, "expected member to be *ast.BadMember")
				require.NotNil(t, bad.Diagnostic)
				require.True(t, p.ast.HasErrors())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			tok := p.stream.Emit()
			require.Equal(t, tc.kwType, keyword.Classify(tok.Literal))

			got, err := p.parseEntity(tok)
			require.NoError(t, err)
			tc.checkBad(t, got, &p)
		})
	}
}

func TestParseFieldOrMethod(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		entryType   tokenizer.TokenType
		initialName string
		want        ast.Member
		expectErr   bool
	}{
		// Terminate fields with a newline because
		// parser will only stop collection when it sees a newline
		{
			name:        "simple field",
			input:       "name int\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "name",
			want: &ast.FieldDeclaration{
				Field: &dialect.GoField{
					Name: "name",
					Type: dialect.NamedRef("int"),
				},
			},
		},
		{
			name:        "pointer field",
			input:       "name *MyType\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "name",
			want: &ast.FieldDeclaration{
				Field: &dialect.GoField{
					Name: "name",
					Type: dialect.PointerTo(dialect.NamedRef("MyType")),
				},
			},
		},
		{
			name:        "slice field",
			input:       "name []int\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "name",
			want: &ast.FieldDeclaration{
				Field: &dialect.GoField{
					Name: "name",
					Type: dialect.SliceOf(dialect.NamedRef("int")),
				},
			},
		},
		{
			name:        "simple method",
			input:       "Method()\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "Method",
			want: &ast.MethodDeclaration{
				Method: &dialect.GoMethod{
					Name: "Method",
				},
			},
		},
		{
			name:        "method with parameters and return",
			input:       "Method(a int) error\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "Method",
			want: &ast.MethodDeclaration{
				Method: &dialect.GoMethod{
					Name: "Method",
					Parameters: []dialect.GoParameter{
						{Name: "a", Type: dialect.NamedRef("int")},
					},
					ReturnType: []dialect.GoParameter{
						{Type: dialect.NamedRef("error")},
					},
				},
			},
		},
		{
			name:        "method defined as method modifier but no params",
			input:       "{method} Method()\n",
			entryType:   tokenizer.LBRACE,
			initialName: "",
			want: &ast.MethodDeclaration{
				Modifiers: []string{"method"},
				Method: &dialect.GoMethod{
					Name: "Method",
				},
			},
		},
		{
			name:        "both field and method modifiers",
			input:       "{field} {method} Name()\n",
			entryType:   tokenizer.LBRACE,
			initialName: "",
			want:        nil,
			expectErr:   true,
		},
		{
			name:        "sketch field without type",
			input:       "name\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "name",
			want: &ast.FieldDeclaration{
				Field: &dialect.GoField{
					Name: "name",
					Type: nil,
				},
			},
			expectErr: false,
		},
		{
			name:        "invalid field (invalid syntax)",
			input:       "name : int\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "name",
			want:        nil,
			expectErr:   true,
		},
		{
			name:        "invalid method (unclosed param parenthesis)",
			input:       "Method(a int\n",
			entryType:   tokenizer.IDENTIFIER,
			initialName: "Method",
			want:        nil,
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
			}

			var entryTok tokenizer.Token
			var mod *string
			vis := ast.VisibilityUnknown

			if tc.entryType == tokenizer.LBRACE {
				m, err := p.tryReadModifier()
				require.NoError(t, err)
				mod = &m
				entryTok = tokenizer.Token{Type: tokenizer.LBRACE}
			} else {
				entryTok = p.stream.Emit()
				require.Equal(t, tc.entryType, entryTok.Type)
				if entryTok.Type == tokenizer.IDENTIFIER {
					require.Equal(t, tc.initialName, entryTok.Literal)
				}
			}

			got, err := p.parseFieldOrMethod(mod, vis, entryTok)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assertASTEqual(t, tc.want, got)
			}
		})
	}
}

func TestParseEntityMember(t *testing.T) {
	tt := []struct {
		name      string
		input     string
		want      ast.Member
		expectErr bool
	}{
		{
			name:  "class separator dots",
			input: ".. separator ..",
			want: &ast.ClassSeparator{
				Label:     "separator",
				Type:      '.',
				TypeCount: 4,
			},
		},
		{
			name:  "class separator hyphens",
			input: "-- section --",
			want: &ast.ClassSeparator{
				Label:     "section",
				Type:      '-',
				TypeCount: 4,
			},
		},
		{
			name:  "public field member",
			input: "+field int\n",
			want: &ast.FieldDeclaration{
				Visibility: ast.VisibilityPublic,
				Field: &dialect.GoField{
					Name: "field",
					Type: dialect.NamedRef("int"),
				},
			},
		},
		{
			name:  "private method member",
			input: "-Method()\n",
			want: &ast.MethodDeclaration{
				Visibility: ast.VisibilityPrivate,
				Method: &dialect.GoMethod{
					Name: "Method",
				},
			},
		},
		{
			name:  "field with modifier",
			input: "{field} myField int\n",
			want: &ast.FieldDeclaration{
				Modifiers: []string{"field"},
				Field: &dialect.GoField{
					Name: "myField",
					Type: dialect.NamedRef("int"),
				},
			},
		},
		{
			name:      "unclosed modifier",
			input:     "{field myField int",
			expectErr: true,
		},
		{
			name:      "unexpected token in body",
			input:     "@invalid",
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
			}
			got, err := p.parseEntityMember()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assertASTEqual(t, tc.want, got)
			}
		})
	}
}

func TestParseInlineMember(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantTargetName string
		wantMethod     bool
		wantField      bool
		wantModifiers  []string
		wantVisibility ast.VisibilityKind
	}{
		{
			name:           "field with visibility",
			input:          "Foo : +id string\n",
			wantTargetName: "Foo",
			wantField:      true,
			wantVisibility: ast.VisibilityPublic,
		},
		{
			name:           "method with visibility and params",
			input:          "Foo : +DoWork(ctx Context) error\n",
			wantTargetName: "Foo",
			wantMethod:     true,
			wantVisibility: ast.VisibilityPublic,
		},
		{
			name:           "field with static modifier",
			input:          "Foo : {static} count int\n",
			wantTargetName: "Foo",
			wantField:      true,
			wantModifiers: []string{
				"static",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			tok := p.stream.Emit()
			for tok.Type == tokenizer.NEWLINE {
				tok = p.stream.Emit()
			}
			got, err := p.parseInlineMember(tok)
			require.NoError(t, err, "unexpected error: %s", err)
			member, ok := got.(*ast.InlineMember)
			require.True(t, ok, "expected inline member but got %T", got)
			require.Equal(t, tc.wantTargetName, member.Target.EntityName(), "expected entity %q but got %q", tc.wantTargetName, member.Target.EntityName())
			if tc.wantMethod {
				method, ok := member.Member.(*ast.MethodDeclaration)
				require.True(t, ok)
				require.Equal(t, tc.wantModifiers, method.Modifiers)
				require.Equal(t, tc.wantVisibility, method.Visibility)
			} else {
				field, ok := member.Member.(*ast.FieldDeclaration)
				require.True(t, ok)
				require.Equal(t, tc.wantModifiers, field.Modifiers)
				require.Equal(t, tc.wantVisibility, field.Visibility)
			}
		})
	}
}

func TestParseEntityAndContainerTags(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStereo string
		wantTags   []string
	}{
		{
			name:       "pre-stereotype tags",
			input:      "class Foo $tag1 $tag2 <<stereo>>\n",
			wantStereo: "stereo",
			wantTags:   []string{"tag1", "tag2"},
		},
		{
			name:       "post-stereotype tags",
			input:      "class Foo <<stereo>> $tag3\n",
			wantStereo: "stereo",
			wantTags:   []string{"tag3"},
		},
		{
			name:       "precedence: pre-stereotype tags override post-stereotype tags",
			input:      "class Foo $pre <<stereo>> $post\n",
			wantStereo: "stereo",
			wantTags:   []string{"pre"},
		},
		{
			name:       "standalone tag without stereotype",
			input:      "class Foo $alone\n",
			wantStereo: "",
			wantTags:   []string{"alone"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			tok := p.stream.Emit()
			got, err := p.parseEntity(tok)
			require.NoError(t, err)
			ent, ok := got.(*ast.Entity)
			require.True(t, ok)
			require.Equal(t, tc.wantStereo, ent.Stereotype)
			require.Equal(t, tc.wantTags, ent.Tags)
		})
	}
}
