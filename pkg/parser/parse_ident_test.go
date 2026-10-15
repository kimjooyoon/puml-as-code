package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

func TestDisambiguateIdent(t *testing.T) {
	t.Run("relationships", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			check func(t *testing.T, stmt ast.Statement, err error)
		}{
			{
				name:  "simple solid arrow",
				input: "User --> Order",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "User", rel.LHS.Entity)
					require.Equal(t, "Order", rel.RHS.Entity)
				},
			},
			{
				name:  "contextual keyword as LHS relationship",
				input: "class --> Order",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "class", rel.LHS.Entity)
					require.Equal(t, "Order", rel.RHS.Entity)
				},
			},
			{
				name:  "contextual keyword with dash link",
				input: "class - cls",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "class", rel.LHS.Entity)
					require.Equal(t, "cls", rel.RHS.Entity)
				},
			},
			{
				name:  "relationship with LHS multiplicity",
				input: `User "1" --> "*" Order`,
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "User", rel.LHS.Entity)
					require.Equal(t, "1", rel.MultLHS.Raw)
					require.Equal(t, "*", rel.MultRHS.Raw)
					require.Equal(t, "Order", rel.RHS.Entity)
				},
			},
			{
				name:  "relationship with left extension arrow and multiplicity",
				input: `User "1" <|-- "*" Order`,
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "User", rel.LHS.Entity)
					require.Equal(t, '|', rel.LArrow)
					require.Equal(t, "Order", rel.RHS.Entity)
				},
			},
			{
				name:  "relationship with left arrow and multiplicity",
				input: `User "1" <-- "*" Order`,
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, "User", rel.LHS.Entity)
					require.Equal(t, '<', rel.LArrow)
				},
			},
			{
				name:  "qualified relationship",
				input: "net.http.Client --> Server",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, []string{"net", "http"}, rel.LHS.PackagePath)
					require.Equal(t, "Client", rel.LHS.Entity)
					require.Equal(t, "Server", rel.RHS.Entity)
				},
			},
			{
				name:  "qualified relationship with multiplicity",
				input: `net.http.Client "1" --> "*" Server`,
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					rel, ok := stmt.(*ast.Relationship)
					require.True(t, ok)
					require.Equal(t, []string{"net", "http"}, rel.LHS.PackagePath)
					require.Equal(t, "Client", rel.LHS.Entity)
					require.Equal(t, "1", rel.MultLHS.Raw)
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
				tc.check(t, stmt, err)
			})
		}
	})

	t.Run("inline members", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			check func(t *testing.T, stmt ast.Statement, err error)
		}{
			{
				name:  "simple inline member",
				input: "User : +id string",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					ent, ok := stmt.(*ast.Entity)
					require.True(t, ok)
					require.Equal(t, "User", ent.Identifier)
					require.Len(t, ent.Members, 1)
				},
			},
			{
				name:  "contextual keyword as inline member target",
				input: "class : +field int",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					ent, ok := stmt.(*ast.Entity)
					require.True(t, ok)
					require.Equal(t, "class", ent.Identifier)
					require.Len(t, ent.Members, 1)
				},
			},
			{
				name:  "note followed by colon is parsed as inline member on note class",
				input: "note : text",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					ent, ok := stmt.(*ast.Entity)
					require.True(t, ok)
					require.Equal(t, "note", ent.Identifier)
					require.Len(t, ent.Members, 1)
				},
			},
			{
				name:  "qualified inline member",
				input: "net.http.Client : +Do() error",
				check: func(t *testing.T, stmt ast.Statement, err error) {
					require.NoError(t, err)
					// Wrapped in container hierarchy
					cont, ok := stmt.(*ast.Container)
					require.True(t, ok)
					require.Equal(t, "net", cont.Identifier)
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
				tc.check(t, stmt, err)
			})
		}
	})

	t.Run("delegation to keyword resolution (returns nil, nil)", func(t *testing.T) {
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
	})

	t.Run("skinparam block", func(t *testing.T) {
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
	})

	t.Run("qualified target syntax error without relationship or colon", func(t *testing.T) {
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
	})

	t.Run("custom package separator set to dash", func(t *testing.T) {
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
			rel, ok := stmt.(*ast.Relationship)
			require.True(t, ok)
			require.Equal(t, []string{"note"}, rel.LHS.PackagePath)
			require.Equal(t, "left", rel.LHS.Entity)
			require.Equal(t, "text", rel.RHS.Entity)
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
			rel, ok := stmt.(*ast.Relationship)
			require.True(t, ok)
			require.Equal(t, "note", rel.LHS.Entity)
			require.Equal(t, []string{"bold"}, rel.Attrs)
			require.Equal(t, "text", rel.RHS.Entity)
		})
	})
}
