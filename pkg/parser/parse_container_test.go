package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/parser/keyword"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func TestParseContainer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		kwType keyword.KeywordKind
		want   ast.Container
	}{
		{
			name:   "together empty newline",
			input:  "together\n",
			kwType: keyword.Together,
			want: ast.Container{
				Kind: ast.ContainerTogether,
			},
		},
		{
			name:   "together empty body",
			input:  "together {}",
			kwType: keyword.Together,
			want: ast.Container{
				Kind: ast.ContainerTogether,
			},
		},
		{
			name:   "together with single class",
			input:  "together { class A }",
			kwType: keyword.Together,
			want: ast.Container{
				Kind: ast.ContainerTogether,
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "package simple identifier",
			input:  "package mypkg\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package string identifier and alias",
			input:  `package "My Package" as mypkg` + "\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Alias:      tokString("My Package"),
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package identifier and string alias",
			input:  `package mypkg as "My Package"` + "\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Alias:      tokString("My Package"),
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package identical identifier and alias",
			input:  "package mypkg as otherpkg\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("otherpkg"),
				Alias:      tokIdent("mypkg"),
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package with stereotype",
			input:  "package mypkg <<Service>>\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Stereotype: "Service",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package with stereotype and color",
			input:  "package mypkg <<Service>> #green\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Stereotype: "Service",
				Color:      "#green",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package body with relationship and nested package",
			input:  "package mypkg { class A together { class B } A -> B\n}",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("mypkg"),
				Kind:       ast.ContainerPackage,
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
					&ast.Container{
						Kind: ast.ContainerTogether,
						Statements: []ast.Statement{
							&ast.Entity{
								Identifier: identRef("B"),
								Kind:       ast.EntityClass,
							},
						},
					},
					&ast.Relationship{
						LHS:       identRef("A"),
						RHS:       identRef("B"),
						Body:      '-',
						RArrow:    '>',
						BodyCount: 1,
						TypeRHS:   ast.RelationAssociation,
					},
				},
			},
		},
		{
			name:   "package with the keyword as a name in relationship",
			input:  "package p { class folder {} folder --> p }",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("p"),
				Kind:       ast.ContainerPackage,
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("folder"),
						Kind:       ast.EntityClass,
					},
					&ast.Relationship{
						LHS:       identRef("folder"),
						RHS:       identRef("p"),
						Body:      '-',
						BodyCount: 2,
						RArrow:    '>',
						TypeRHS:   ast.RelationAssociation,
					},
				},
			},
		},
		{
			name:   "expect to correctly parse nested containers",
			input:  "package p.p {}",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: identRef("p", "p"),
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package with same line body and color",
			input:  "package mypkg #red { class A }",
			kwType: keyword.Package,
			want: ast.Container{
				Kind:       ast.ContainerPackage,
				Identifier: identRef("mypkg"),
				Color:      "#red",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		// --- Container keyword coverage ---
		{
			name:   "folder with body",
			input:  "folder myfolder { class A }",
			kwType: keyword.Folder,
			want: ast.Container{
				Kind:       ast.ContainerFolder,
				Identifier: identRef("myfolder"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "frame with body",
			input:  "frame myframe { class A }",
			kwType: keyword.Frame,
			want: ast.Container{
				Kind:       ast.ContainerFrame,
				Identifier: identRef("myframe"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "rectangle with body",
			input:  "rectangle myrect { class A }",
			kwType: keyword.Rectangle,
			want: ast.Container{
				Kind:       ast.ContainerRectangle,
				Identifier: identRef("myrect"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "cloud with body",
			input:  "cloud mycloud { class A }",
			kwType: keyword.Cloud,
			want: ast.Container{
				Kind:       ast.ContainerCloud,
				Identifier: identRef("mycloud"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "database with body",
			input:  "database mydb { class A }",
			kwType: keyword.Database,
			want: ast.Container{
				Kind:       ast.ContainerDatabase,
				Identifier: identRef("mydb"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "node with body",
			input:  "node mynode { class A }",
			kwType: keyword.Node,
			want: ast.Container{
				Kind:       ast.ContainerNode,
				Identifier: identRef("mynode"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "namespace with body",
			input:  "namespace myns { class A }",
			kwType: keyword.Namespace,
			want: ast.Container{
				Kind:       ast.ContainerNamespace,
				Identifier: identRef("myns"),
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: identRef("A"),
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:   "folder empty body",
			input:  "folder myfolder {}",
			kwType: keyword.Folder,
			want: ast.Container{
				Kind:       ast.ContainerFolder,
				Identifier: identRef("myfolder"),
			},
		},
		{
			name:   "namespace with alias and stereotype",
			input:  `namespace myns as "My Namespace" <<API>>` + "\n",
			kwType: keyword.Namespace,
			want: ast.Container{
				Kind:       ast.ContainerNamespace,
				Identifier: identRef("myns"),
				Alias:      tokString("My Namespace"),
				Stereotype: "API",
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

			got, err := p.parseContainer(tok)
			require.NoError(t, err)
			assertASTEqual(t, tc.want, got)
		})
	}
}

func TestParseContainer_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		kwType      keyword.KeywordKind
		errContains string
	}{
		{
			name:        "together empty error",
			input:       "together",
			kwType:      keyword.Together,
			errContains: "Expected container body to end",
		},
		{
			name:        "package incomplete alias",
			input:       "package mypkg as\n",
			kwType:      keyword.Package,
			errContains: "Expected identifier or string for container alias",
		},
		{
			name:        "package unclosed stereotype",
			input:       "package mypkg <<Service\n",
			kwType:      keyword.Package,
			errContains: "unexpected EOF",
		},
		{
			name:        "package unclosed body",
			input:       "package mypkg { class A",
			kwType:      keyword.Package,
			errContains: "unexpected EOF",
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

			got, err := p.parseContainer(tok)
			require.Error(t, err)
			require.Nil(t, got)
			if tc.errContains == "" {
				t.Fatal("expected error but got none")
			}
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}

func TestParseContainer_Recovery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		kwType   keyword.KeywordKind
		checkBad func(t *testing.T, container *ast.Container, p *Parser)
	}{
		{
			name:   "package unexpected token in body",
			input:  "package mypkg { @invalid }",
			kwType: keyword.Package,
			checkBad: func(t *testing.T, container *ast.Container, p *Parser) {
				require.NotNil(t, container)
				require.Len(t, container.Statements, 1)
				bad, ok := container.Statements[0].(*ast.BadStatement)
				require.True(t, ok, "expected statement to be *ast.BadStatement")
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

			got, err := p.parseContainer(tok)
			require.NoError(t, err)
			tc.checkBad(t, got, &p)
		})
	}
}
