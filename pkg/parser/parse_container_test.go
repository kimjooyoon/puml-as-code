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
		name        string
		input       string
		kwType      keyword.KeywordKind
		want        ast.Container
		expectErr   bool
		errContains string
	}{
		{
			name:        "together empty error",
			input:       "together",
			kwType:      keyword.Together,
			expectErr:   true,
			errContains: "Expected container body to end",
		},
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
						Identifier: "A",
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
				Identifier: "mypkg",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package string identifier and alias",
			input:  `package "My Package" as mypkg` + "\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: "mypkg",
				Alias:      "\"My Package\"",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package identifier and string alias",
			input:  `package mypkg as "My Package"` + "\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: "mypkg",
				Alias:      "\"My Package\"",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package identical identifier and alias",
			input:  "package mypkg as otherpkg\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: "otherpkg",
				Alias:      "mypkg",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package with stereotype",
			input:  "package mypkg <<Service>>\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: "mypkg",
				Stereotype: "Service",
				Kind:       ast.ContainerPackage,
			},
		},
		{
			name:   "package with stereotype and color",
			input:  "package mypkg <<Service>> #green\n",
			kwType: keyword.Package,
			want: ast.Container{
				Identifier: "mypkg",
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
				Identifier: "mypkg",
				Kind:       ast.ContainerPackage,
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
						Kind:       ast.EntityClass,
					},
					&ast.Container{
						Kind: ast.ContainerTogether,
						Statements: []ast.Statement{
							&ast.Entity{
								Identifier: "B",
								Kind:       ast.EntityClass,
							},
						},
					},
					&ast.Relationship{
						LHS:       ast.TargetRef{Entity: "A"},
						RHS:       ast.TargetRef{Entity: "B"},
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
				Identifier: "p",
				Kind:       ast.ContainerPackage,
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "folder",
						Kind:       ast.EntityClass,
					},
					&ast.Relationship{
						LHS:       ast.TargetRef{Entity: "folder"},
						RHS:       ast.TargetRef{Entity: "p"},
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
				Identifier: "p",
				Kind:       ast.ContainerPackage,
				Statements: []ast.Statement{
					&ast.Container{
						Identifier: "p",
						Kind:       ast.ContainerPackage,
					},
				},
			},
		},
		{
			name:        "package incomplete alias",
			input:       "package mypkg as\n",
			kwType:      keyword.Package,
			expectErr:   true,
			errContains: "Expected identifier for container name",
		},
		{
			name:        "package unclosed stereotype",
			input:       "package mypkg <<Service\n",
			kwType:      keyword.Package,
			expectErr:   true,
			errContains: "unexpected EOF",
		},
		{
			name:        "package unclosed body",
			input:       "package mypkg { class A",
			kwType:      keyword.Package,
			expectErr:   true,
			errContains: "unexpected EOF",
		},
		{
			name:        "package unexpected token in body",
			input:       "package mypkg { @invalid }",
			kwType:      keyword.Package,
			expectErr:   true,
			errContains: "Expected a statement in a container body",
		},
		{
			name:   "package with same line body and color",
			input:  "package mypkg #red { class A }",
			kwType: keyword.Package,
			want: ast.Container{
				Kind:       ast.ContainerPackage,
				Identifier: "mypkg",
				Color:      "#red",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
						Kind:       ast.EntityClass,
					},
				},
			},
		},
		{
			name:        "package body class error",
			input:       "package mypkg { class MyClass as }",
			kwType:      keyword.Package,
			expectErr:   true,
			errContains: "Expected token for entity identifier or alias",
		},
		// --- Container keyword coverage ---
		{
			name:   "folder with body",
			input:  "folder myfolder { class A }",
			kwType: keyword.Folder,
			want: ast.Container{
				Kind:       ast.ContainerFolder,
				Identifier: "myfolder",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "myframe",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "myrect",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "mycloud",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "mydb",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "mynode",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "myns",
				Statements: []ast.Statement{
					&ast.Entity{
						Identifier: "A",
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
				Identifier: "myfolder",
			},
		},
		{
			name:   "namespace with alias and stereotype",
			input:  `namespace myns as "My Namespace" <<API>>` + "\n",
			kwType: keyword.Namespace,
			want: ast.Container{
				Kind:       ast.ContainerNamespace,
				Identifier: "myns",
				Alias:      "\"My Namespace\"",
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
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				assertASTEqual(t, tc.want, got)
			}
		})
	}
}

