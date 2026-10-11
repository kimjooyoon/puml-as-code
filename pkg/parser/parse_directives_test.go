package parser

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"

	"github.com/stretchr/testify/require"
)

func TestParseIncludeDirectives(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      ast.IncludeDirective
		expectErr bool
	}{
		{
			name:  "include simple path",
			input: "!include common.puml",
			want: ast.IncludeDirective{
				Kind: ast.IncludeOnce,
				Path: "common.puml",
			},
		},
		{
			name:  "include_once with path",
			input: "!include_once ./types/models.puml",
			want: ast.IncludeDirective{
				Kind: ast.IncludeOnce,
				Path: "./types/models.puml",
			},
		},
		{
			name:  "include_many with path",
			input: "!include_many base.puml",
			want: ast.IncludeDirective{
				Kind: ast.IncludeMany,
				Path: "base.puml",
			},
		},
		{
			name:  "include with diagram tag",
			input: "!include common.puml!diagram1",
			want: ast.IncludeDirective{
				Kind: ast.IncludeOnce,
				Path: "common.puml",
				Tag:  "diagram1",
			},
		},
		{
			name:  "include_many with diagram tag index",
			input: "!include_many common.puml!0",
			want: ast.IncludeDirective{
				Kind: ast.IncludeMany,
				Path: "common.puml",
				Tag:  "0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(dialect.NewGoDialect())
			p.IsBoundless = true
			diagram, err := p.Parse(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, diagram.Statements, 1)

			got, ok := diagram.Statements[0].(*ast.IncludeDirective)
			require.True(t, ok, "statement should be *ast.IncludeDirective")
			require.Equal(t, tt.want.Kind, got.Kind)
			require.Equal(t, tt.want.Path, got.Path)
			require.Equal(t, tt.want.Tag, got.Tag)
		})
	}
}
