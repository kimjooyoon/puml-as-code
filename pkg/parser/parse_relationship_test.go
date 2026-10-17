package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func TestParseArrowTokens(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *ast.Relationship
		expectErr   bool
		errContains string
	}{
		{
			name:  "simple solid right arrow",
			input: "-->",
			want: &ast.Relationship{
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
			},
		},
		{
			name:  "simple solid right arrow short",
			input: "->",
			want: &ast.Relationship{
				Body:      '-',
				RArrow:    '>',
				BodyCount: 1,
				TypeRHS:   ast.RelationAssociation,
			},
		},
		{
			name:  "simple dotted right arrow",
			input: "..>",
			want: &ast.Relationship{
				Body:      '.',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationDependency,
			},
		},
		{
			name:  "double-headed arrow solid",
			input: "<-->",
			want: &ast.Relationship{
				LArrow:    '<',
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeLHS:   ast.RelationAssociation,
				TypeRHS:   ast.RelationAssociation,
			},
		},
		{
			name:  "double-headed arrow dotted",
			input: "<..>",
			want: &ast.Relationship{
				LArrow:    '<',
				Body:      '.',
				RArrow:    '>',
				BodyCount: 2,
				TypeLHS:   ast.RelationDependency,
				TypeRHS:   ast.RelationDependency,
			},
		},
		{
			name:  "left extension solid right arrow",
			input: "<|-->",
			want: &ast.Relationship{
				LArrow:    '|',
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeLHS:   ast.RelationInheritance,
				TypeRHS:   ast.RelationAssociation,
			},
		},
		{
			name:  "solid right extension arrow",
			input: "--|>",
			want: &ast.Relationship{
				Body:      '-',
				RArrow:    '|',
				BodyCount: 2,
				TypeRHS:   ast.RelationInheritance,
			},
		},
		{
			name:  "double extension solid",
			input: "<|--|>",
			want: &ast.Relationship{
				LArrow:    '|',
				Body:      '-',
				RArrow:    '|',
				BodyCount: 2,
				TypeLHS:   ast.RelationInheritance,
				TypeRHS:   ast.RelationInheritance,
			},
		},
		{
			name:  "curly braces left/right",
			input: "}--{",
			want: &ast.Relationship{
				LArrow:    '}',
				Body:      '-',
				RArrow:    '{',
				BodyCount: 2,
			},
		},
		{
			name:  "curly braces dotted left/right",
			input: "}..{",
			want: &ast.Relationship{
				LArrow:    '}',
				Body:      '.',
				RArrow:    '{',
				BodyCount: 2,
			},
		},
		{
			name:  "lolipop right arrowhead",
			input: "--()",
			want: &ast.Relationship{
				Body:      '-',
				RArrow:    '(',
				BodyCount: 2,
			},
		},
		{
			name:        "less-than pipe greater-than diamond/extension",
			input:       "<|>",
			expectErr:   true,
			errContains: "Unexpected token as the relationship body",
			want: &ast.Relationship{
				LArrow: '|',
			},
		},
		{
			name:  "custom 'x' left/right",
			input: "x--x",
			want: &ast.Relationship{
				LArrow:    'x',
				Body:      '-',
				RArrow:    'x',
				BodyCount: 2,
			},
		},
		{
			name:  "custom 'o' left/right",
			input: "o--o",
			want: &ast.Relationship{
				LArrow:    'o',
				Body:      '-',
				RArrow:    'o',
				BodyCount: 2,
				TypeLHS:   ast.RelationAggregation,
				TypeRHS:   ast.RelationAggregation,
			},
		},
		{
			name:  "asterisk left/right",
			input: "*--*",
			want: &ast.Relationship{
				LArrow:    '*',
				Body:      '-',
				RArrow:    '*',
				BodyCount: 2,
				TypeLHS:   ast.RelationComposition,
				TypeRHS:   ast.RelationComposition,
			},
		},
		{
			name:  "plus left/right",
			input: "+--+",
			want: &ast.Relationship{
				LArrow:    '+',
				Body:      '-',
				RArrow:    '+',
				BodyCount: 2,
			},
		},
		{
			name:  "caret left/right",
			input: "^--^",
			want: &ast.Relationship{
				LArrow:    '^',
				Body:      '-',
				RArrow:    '^',
				BodyCount: 2,
			},
		},
		{
			name:  "hash left/right",
			input: "#--#",
			want: &ast.Relationship{
				LArrow:    '#',
				Body:      '-',
				RArrow:    '#',
				BodyCount: 2,
			},
		},
		{
			name:  "left direction",
			input: "-left->",
			want: &ast.Relationship{
				Body:      '-',
				Direction: ast.DirectionLeft,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "right direction",
			input: "-right->",
			want: &ast.Relationship{
				Body:      '-',
				Direction: ast.DirectionRight,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "up direction",
			input: "-up->",
			want: &ast.Relationship{
				Body:      '-',
				Direction: ast.DirectionTop,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "down direction",
			input: "-down->",
			want: &ast.Relationship{
				Body:      '-',
				Direction: ast.DirectionBottom,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "single attribute",
			input: "-[foo]->",
			want: &ast.Relationship{
				Body:      '-',
				Attrs:     []string{"foo"},
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "multiple attributes",
			input: "-[foo,bar]->",
			want: &ast.Relationship{
				Body:      '-',
				Attrs:     []string{"foo", "bar"},
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "attributes and direction",
			input: "-[foo]left->",
			want: &ast.Relationship{
				Body:      '-',
				Attrs:     []string{"foo"},
				Direction: ast.DirectionLeft,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "direction and attributes",
			input: "-left[foo]->",
			want: &ast.Relationship{
				Body:      '-',
				Attrs:     []string{"foo"},
				Direction: ast.DirectionLeft,
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "multitoken attributes",
			input: "-[#foo,%bar]->",
			want: &ast.Relationship{
				Body:      '-',
				Attrs:     []string{"#foo", "%bar"},
				RArrow:    '>',
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "no arrowheads solid",
			input: "--",
			want: &ast.Relationship{
				Body:      '-',
				TypeLHS:   ast.RelationAssociation,
				TypeRHS:   ast.RelationAssociation,
				BodyCount: 2,
			},
		},
		{
			name:  "no arrowheads dotted",
			input: "..",
			want: &ast.Relationship{
				Body:      '.',
				BodyCount: 2,
			},
		},
		{
			name:  "preserve body count",
			input: "----",
			want: &ast.Relationship{
				Body:      '-',
				BodyCount: 4,
				TypeLHS:   ast.RelationAssociation,
				TypeRHS:   ast.RelationAssociation,
			},
		},
		{
			name:        "empty input error",
			input:       "",
			expectErr:   true,
			errContains: "Unexpected token at the start of relationship definition",
		},
		{
			name:        "invalid starting token error",
			input:       "@",
			expectErr:   true,
			errContains: "Unexpected token at the start of relationship definition",
		},
		{
			name:        "missing trailing body rune error",
			input:       "-up>",
			expectErr:   true,
			errContains: "Unexpected token in body relationship definition",
		},
		{
			name:        "invalid top direction value error",
			input:       "-top->",
			expectErr:   true,
			errContains: "Unexpected direction in relationship",
		},
		{
			name:        "invalid bottom direction value error",
			input:       "-bottom->",
			expectErr:   true,
			errContains: "Unexpected direction in relationship",
		},
		{
			name:        "invalid identifier error",
			input:       "-foo->",
			expectErr:   true,
			errContains: "Unexpected identifier in relationship definition",
		},
		{
			name:        "unexpected separator inside direction error",
			input:       "-left-[foo]->",
			expectErr:   true,
			errContains: "Cannot separate direction and attributes with a body token",
		},
		{
			name:        "unclosed pipe extension error",
			input:       "--|",
			expectErr:   true,
			errContains: "Expected '|>' after relationship",
		},
		{
			name:        "lolipop right after direction error",
			input:       "-left-()",
			expectErr:   true,
			errContains: "Lolipop interface cannot contain direction or attributes",
		},
		{
			name:        "lolipop right after attributes error",
			input:       "-[foo]-()",
			expectErr:   true,
			errContains: "Lolipop interface cannot contain direction or attributes",
		},
		{
			name:        "mixed body types error",
			input:       "-.-",
			expectErr:   true,
			errContains: "Different body type runes in relationship",
		},
		{
			name:        "unclosed attributes at EOF error",
			input:       "-[foo",
			expectErr:   true,
			errContains: "Unexpected break in relationship attribute container",
			want: &ast.Relationship{
				Body:      '-',
				BodyCount: 1,
			},
		},
		{
			name:        "unclosed attributes at newline error",
			input:       "-[foo\n",
			expectErr:   true,
			errContains: "Unexpected break in relationship attribute container",
			want: &ast.Relationship{
				Body:      '-',
				BodyCount: 1,
			},
		},
		{
			name:        "missing attribute (double comma)",
			input:       "-[foo,,bar]->",
			expectErr:   true,
			errContains: "Unexpected comma in relationship attribute container",
			want: &ast.Relationship{
				Body:      '-',
				BodyCount: 1,
				Attrs:     []string{"foo"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream: tokenizer.NewTokenStream(tc.input),
			}
			var got ast.Relationship
			err := p.parseArrowTokens(&got)
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			if tc.want != nil {
				require.Equal(t, *tc.want, got)
			}
		})
	}
}

func TestParseRelationship(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      ast.Relationship
		expectErr bool
	}{
		{
			name:  "composition left-headed without right arrowhead",
			input: "Car *-- Engine",
			want: ast.Relationship{
				LHS:       identRef("Car"),
				LArrow:    '*',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationComposition,
				RHS:       identRef("Engine"),
			},
		},
		{
			name:  "aggregation left-headed without right arrowhead",
			input: "Car o-- Wheel",
			want: ast.Relationship{
				LHS:       identRef("Car"),
				LArrow:    'o',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationAggregation,
				RHS:       identRef("Wheel"),
			},
		},
		{
			name:  "inheritance left-headed without right arrowhead",
			input: "Vehicle <|-- Car",
			want: ast.Relationship{
				LHS:       identRef("Vehicle"),
				LArrow:    '|',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationInheritance,
				RHS:       identRef("Car"),
			},
		},
		{
			name:  "composition with multiplicity on target",
			input: "Car *-- \"1..*\" Engine",
			want: ast.Relationship{
				LHS:       identRef("Car"),
				LArrow:    '*',
				Body:      '-',
				BodyCount: 2,
				TypeLHS:   ast.RelationComposition,
				MultRHS:   ast.Cardinality{Raw: "1..*", Min: 1, Max: -1},
				RHS:       identRef("Engine"),
			},
		},
		{
			name:  "association with label",
			input: "User --> Service : uses",
			want: ast.Relationship{
				LHS:       identRef("User"),
				Body:      '-',
				RArrow:    '>',
				BodyCount: 2,
				TypeRHS:   ast.RelationAssociation,
				RHS:       identRef("Service"),
				Label:     "uses",
			},
		},
		{
			name:  "realization with right arrowhead",
			input: "User ..|> Greeter",
			want: ast.Relationship{
				LHS:       identRef("User"),
				Body:      '.',
				RArrow:    '|',
				BodyCount: 2,
				TypeRHS:   ast.RelationRealization,
				RHS:       identRef("Greeter"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream: tokenizer.NewTokenStream(tc.input),
			}
			firstTok := p.stream.Emit()

			rel, err := p.parseRelationship(firstTok)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assertASTEqual(t, tc.want, rel)
			}
		})
	}
}

func TestParseTargetRef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  ast.TargetRef
	}{
		{
			name:  "simple entity",
			input: "Client",
			want:  identRef("Client"),
		},
		{
			name:  "package path and entity",
			input: "net.http.Client",
			want:  identRef("Client", "net", "http"),
		},
		{
			name:  "entity and simple member",
			input: "Client::Do",
			want:  identRefMember("Client", "Do"),
		},
		{
			name:  "entity and method parens",
			input: "Client::Do()",
			want: identRefMethod("Client", "Do", []tokenizer.Token{
				{Type: tokenizer.LPAREN, Literal: "("},
				{Type: tokenizer.RPAREN, Literal: ")"},
			}),
		},
		{
			name:  "entity and method with parameters",
			input: `Client::"Do(req Request)"`,
			want:  identRefStringMember("Client", "Do(req Request)"),
		},
		{
			name:  "package path entity and method in quotes",
			input: `net.http.Client::"Do(Context)"`,
			want:  identRefStringMember("Client", "Do(Context)", "net", "http"),
		},
		{
			name:  "package separator follows the entity but not a part of it",
			input: "Foo ..|> Bar",
			want:  identRef("Foo"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := Parser{
				stream:  tokenizer.NewTokenStream(tc.input),
				Dialect: dialect.NewGoDialect(),
				ast:     &ast.Diagram{},
			}
			firstTok := p.stream.Emit()
			got, err := p.parseTargetRef(firstTok)
			require.NoError(t, err)
			assertASTEqual(t, tc.want, got)
		})
	}
}
