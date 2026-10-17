package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

func TestDisambiguateIdent_Relationships(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *ast.Relationship
	}{
		{
			name:  "simple solid arrow",
			input: "User --> Order",
			want: &ast.Relationship{
				LHS:       identRef("User"),
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				RHS:       identRef("Order"),
			},
		},
		{
			name:  "contextual keyword as LHS relationship",
			input: "class --> Order",
			want: &ast.Relationship{
				LHS:       identRef("class"),
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				RHS:       identRef("Order"),
			},
		},
		{
			name:  "contextual keyword with dash link",
			input: "class - cls",
			want: &ast.Relationship{
				LHS:       identRef("class"),
				Body:      '-',
				BodyCount: 1,
				RHS:       identRef("cls"),
			},
		},
		{
			name:  "relationship with LHS multiplicity",
			input: `User "1" --> "*" Order`,
			want: &ast.Relationship{
				LHS:       identRef("User"),
				MultLHS:   ast.Cardinality{Raw: "1", Min: 1, Max: 1},
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				MultRHS:   ast.Cardinality{Raw: "*", Min: 0, Max: -1},
				RHS:       identRef("Order"),
			},
		},
		{
			name:  "relationship with left extension arrow and multiplicity",
			input: `User "1" <|-- "*" Order`,
			want: &ast.Relationship{
				LHS:       identRef("User"),
				MultLHS:   ast.Cardinality{Raw: "1", Min: 1, Max: 1},
				LArrow:    '|',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationInheritance,
				MultRHS:   ast.Cardinality{Raw: "*", Min: 0, Max: -1},
				RHS:       identRef("Order"),
			},
		},
		{
			name:  "relationship with left arrow and multiplicity",
			input: `User "1" <-- "*" Order`,
			want: &ast.Relationship{
				LHS:       identRef("User"),
				MultLHS:   ast.Cardinality{Raw: "1", Min: 1, Max: 1},
				LArrow:    '<',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationAssociation,
				MultRHS:   ast.Cardinality{Raw: "*", Min: 0, Max: -1},
				RHS:       identRef("Order"),
			},
		},
		{
			name:  "qualified relationship",
			input: "net.http.Client --> Server",
			want: &ast.Relationship{
				LHS:       identRef("Client", "net", "http"),
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				RHS:       identRef("Server"),
			},
		},
		{
			name:  "qualified relationship with multiplicity",
			input: `net.http.Client "1" --> "*" Server`,
			want: &ast.Relationship{
				LHS:       identRef("Client", "net", "http"),
				MultLHS:   ast.Cardinality{Raw: "1", Min: 1, Max: 1},
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				MultRHS:   ast.Cardinality{Raw: "*", Min: 0, Max: -1},
				RHS:       identRef("Server"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parser{
				Dialect: dialect.NewGoDialect(),
				stream:  tokenizer.NewTokenStream(tc.input),
				ast:     &ast.Diagram{},
			}
			firstTok := p.stream.Emit()
			stmt, err := p.disambiguateIdent(firstTok)
			require.NoError(t, err)
			assertASTEqual(t, tc.want, stmt)
		})
	}
}

func TestDisambiguateIdent_InlineMembers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.Statement
	}{
		{
			name:  "simple inline member",
			input: "User : +id string",
			want: &ast.InlineMember{
				Target: identRef("User"),
				Member: &ast.FieldDeclaration{
					Visibility: ast.VisibilityPublic,
					Field: &dialect.GoField{
						Name: "id",
						Type: dialect.NamedRef("string"),
					},
				},
			},
		},
		{
			name:  "contextual keyword as inline member target",
			input: "class : +field int",
			want: &ast.InlineMember{
				Target: identRef("class"),
				Member: &ast.FieldDeclaration{
					Visibility: ast.VisibilityPublic,
					Field: &dialect.GoField{
						Name: "field",
						Type: dialect.NamedRef("int"),
					},
				},
			},
		},
		{
			name:  "note followed by colon is parsed as inline member on note class",
			input: "note : text",
			want: &ast.InlineMember{
				Target: identRef("note"),
				Member: &ast.FieldDeclaration{
					Visibility: ast.VisibilityUnknown,
					Field: &dialect.GoField{
						Name: "text",
					},
				},
			},
		},
		{
			name:  "qualified inline member",
			input: "net.http.Client : +Do() error",
			want: &ast.InlineMember{
				Target: identRef("Client", "net", "http"),
				Member: &ast.MethodDeclaration{
					Visibility: ast.VisibilityPublic,
					Method: &dialect.GoMethod{
						Name: "Do",
						ReturnType: []dialect.GoParameter{
							{Type: dialect.NamedRef("error")},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parser{
				Dialect: dialect.NewGoDialect(),
				stream:  tokenizer.NewTokenStream(tc.input),
				ast:     &ast.Diagram{},
			}
			firstTok := p.stream.Emit()
			stmt, err := p.disambiguateIdent(firstTok)
			require.NoError(t, err)
			assertASTEqual(t, tc.want, stmt)
		})
	}
}

func TestDisambiguateIdent_Delegation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "class declaration", input: "class Foo"},
		{name: "class with display name and alias", input: `class "Display Name" as Foo`},
		{name: "class with display name and generic", input: `class "Display Name" <T>`},
		{name: "class with display name and stereotype", input: `class "Display Name" <<Service>>`},
		{name: "class with display name and color", input: `class "Display Name" #red`},
		{name: "class with display name and body", input: "class \"Display Name\" {\n}"},
		{name: "package declaration", input: "package mypkg"},
		{name: "package with body", input: "package mypkg {\n}"},
		{name: "anonymous together block", input: "together {\n}"},
		{name: "title statement", input: "title My Title"},
		{name: "hide statement", input: "hide empty members"},
		{name: "scale statement", input: "scale 1.5"},
		{name: "note statement", input: "note left of Foo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parser{
				Dialect: dialect.NewGoDialect(),
				stream:  tokenizer.NewTokenStream(tc.input),
				ast:     &ast.Diagram{},
			}
			firstTok := p.stream.Emit()
			stmt, err := p.disambiguateIdent(firstTok)
			require.NoError(t, err)
			require.Nil(t, stmt, "expected statement to be delegated (nil, nil) for input: %q", tc.input)
		})
	}
}

func TestDisambiguateIdent_Skinparam(t *testing.T) {
	input := "skinparam {\n  ClassFontColor blue\n}"
	p := &Parser{
		Dialect: dialect.NewGoDialect(),
		stream:  tokenizer.NewTokenStream(input),
		ast:     &ast.Diagram{},
	}
	firstTok := p.stream.Emit()
	stmt, err := p.disambiguateIdent(firstTok)
	require.NoError(t, err)
	require.NotNil(t, stmt)
	_, ok := stmt.(*ast.SkinparamBlock)
	require.True(t, ok)
}

func TestDisambiguateIdent_QualifiedTargetError(t *testing.T) {
	input := "com.example.User"
	p := &Parser{
		Dialect: dialect.NewGoDialect(),
		stream:  tokenizer.NewTokenStream(input),
		ast:     &ast.Diagram{},
	}
	firstTok := p.stream.Emit()
	stmt, err := p.disambiguateIdent(firstTok)
	require.Error(t, err)
	require.Nil(t, stmt)
	require.Contains(t, err.Error(), "expected relationship arrow or ':' after qualified target")
}

func TestDisambiguateIdent_CustomPackageSeparatorDash(t *testing.T) {
	t.Run("kebab target with arrow", func(t *testing.T) {
		input := "note-left->text"
		p := &Parser{
			Dialect: dialect.NewGoDialect(),
			stream:  tokenizer.NewTokenStream(input),
			ast:     &ast.Diagram{},
		}
		p.stream.PackageSeparator = "-"
		firstTok := p.stream.Emit()
		stmt, err := p.disambiguateIdent(firstTok)
		require.NoError(t, err)

		want := &ast.Relationship{
			LHS:       identRef("left", "note"),
			Body:      '-',
			RArrow:    '>',
			BodyCount: 1,
			TypeRHS:   ast.RelationAssociation,
			RHS:       identRef("text"),
		}
		assertASTEqual(t, want, stmt)
	})

	t.Run("target with styled arrow", func(t *testing.T) {
		input := "note-[bold]->text"
		p := &Parser{
			Dialect: dialect.NewGoDialect(),
			stream:  tokenizer.NewTokenStream(input),
			ast:     &ast.Diagram{},
		}
		p.stream.PackageSeparator = "-"
		firstTok := p.stream.Emit()
		stmt, err := p.disambiguateIdent(firstTok)
		require.NoError(t, err)

		want := &ast.Relationship{
			LHS:       identRef("note"),
			Body:      '-',
			RArrow:    '>',
			BodyCount: 2,
			Attrs:     []string{"bold"},
			TypeRHS:   ast.RelationAssociation,
			RHS:       identRef("text"),
		}
		assertASTEqual(t, want, stmt)
	})
}
