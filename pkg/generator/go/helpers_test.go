package gogenerator

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/resolver"
	"yur4uwe/pac/pkg/tokenizer"

	"github.com/stretchr/testify/require"
)

func targetRefName(name string) ast.TargetRef {
	return ast.TargetRef{
		Name: ast.Ident{
			Literal: name,
		},
	}
}

func TestTargetTypeName(t *testing.T) {
	tt := []struct {
		name   string
		srcPkg []string
		target *resolver.EntitySymbol
		want   string
	}{
		{
			name:   "same package",
			srcPkg: []string{"models"},
			target: &resolver.EntitySymbol{
				FQN: "models.User",
				AST: &ast.Entity{
					Identifier: targetRefName("User"),
				},
				PackagePath: []string{"models"},
			},
			want: "User",
		},
		{
			name:   "different package",
			srcPkg: []string{"orders"},
			target: &resolver.EntitySymbol{
				FQN: "models.User",
				AST: &ast.Entity{
					Identifier: targetRefName("User"),
				},
				PackagePath: []string{"models"},
			},
			want: "models.User",
		},
		{
			name:   "no package",
			srcPkg: []string{},
			target: &resolver.EntitySymbol{
				FQN: "models.User",
				AST: &ast.Entity{
					Identifier: targetRefName("User"),
				},
				PackagePath: []string{"models"},
			},
			want: "models.User",
		},
		{
			name: "target is in root package",
			srcPkg: []string{
				"orders",
			},
			target: &resolver.EntitySymbol{
				FQN: "User",
				AST: &ast.Entity{
					Identifier: targetRefName("User"),
				},
				PackagePath: []string{},
			},
			want: "root.User",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			resultType := targetTypeName(tc.srcPkg, tc.target)
			require.Equal(t, tc.want, resultType)
		})
	}
}

func TestParseGeneric(t *testing.T) {
	tt := []struct {
		name        string
		generic     string
		expected    []GenericView
		expectedErr string
	}{
		{
			name:    "single generic with only name",
			generic: "T",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "any",
				},
			},
			expectedErr: "",
		},
		{
			name:    "single generic with name and constraint",
			generic: "T int",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "int",
				},
			},
			expectedErr: "",
		},
		{
			name:    "multiple generics without constraints",
			generic: "T, U, V",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "any",
				},
				{
					Name:       "U",
					Constraint: "any",
				},
				{
					Name:       "V",
					Constraint: "any",
				},
			},
			expectedErr: "",
		},
		{
			name:    "multiple generics with constraints",
			generic: "K comparable, V any",
			expected: []GenericView{
				{
					Name:       "K",
					Constraint: "comparable",
				},
				{
					Name:       "V",
					Constraint: "any",
				},
			},
			expectedErr: "",
		},
		{
			name:    "union type constraint",
			generic: "T int | string",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "int | string",
				},
			},
			expectedErr: "",
		},
		{
			name:    "approximation and complex union constraint",
			generic: "T ~int | ~float64 | ~string",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "~int | ~float64 | ~string",
				},
			},
			expectedErr: "",
		},
		{
			name:    "qualified package constraint",
			generic: "T constraints.Ordered",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "constraints.Ordered",
				},
			},
			expectedErr: "",
		},
		{
			name:    "slice and pointer constraints",
			generic: "T []byte, U *os.File",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "[]byte",
				},
				{
					Name:       "U",
					Constraint: "*os.File",
				},
			},
			expectedErr: "",
		},
		{
			name:    "identifiers with digits and underscores",
			generic: "T1 int, _elem any, Item_2 comparable",
			expected: []GenericView{
				{
					Name:       "T1",
					Constraint: "int",
				},
				{
					Name:       "_elem",
					Constraint: "any",
				},
				{
					Name:       "Item_2",
					Constraint: "comparable",
				},
			},
			expectedErr: "",
		},
		{
			name:    "extra whitespace around delimiters and tokens",
			generic: "   T    int    ,    U    string   ",
			expected: []GenericView{
				{
					Name:       "T",
					Constraint: "int",
				},
				{
					Name:       "U",
					Constraint: "string",
				},
			},
			expectedErr: "",
		},
		{
			name:        "empty string error",
			generic:     "",
			expected:    nil,
			expectedErr: "invalid generic declaration: ",
		},
		{
			name:        "comma only error",
			generic:     " , ",
			expected:    nil,
			expectedErr: "invalid generic declaration:  , ",
		},
		{
			name:        "invalid identifier starting with digit",
			generic:     "1T int",
			expected:    nil,
			expectedErr: `invalid generic identifier "1T" in: 1T int`,
		},
		{
			name:        "invalid identifier containing hyphen",
			generic:     "T-1 int",
			expected:    nil,
			expectedErr: `invalid generic identifier "T-1" in: T-1 int`,
		},
		{
			name:        "invalid identifier containing special characters",
			generic:     "T@ int",
			expected:    nil,
			expectedErr: `invalid generic identifier "T@" in: T@ int`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseGeneric(tc.generic)
			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestToTriviaView(t *testing.T) {
	tt := []struct {
		name     string
		trivia   ast.BaseNode
		expected TriviaView
	}{
		{
			name:   "empty trivia",
			trivia: ast.BaseNode{},
			expected: TriviaView{
				LeadingTrivia:  nil,
				TrailingTrivia: nil,
			},
		},
		{
			name: "leading trivia only",
			trivia: ast.BaseNode{
				LeadingTrivia: []ast.Comment{
					{Text: "Doc comment line 1", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 1}}},
					{Text: "Doc comment line 2", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 2}}},
				},
			},
			expected: TriviaView{
				LeadingTrivia:  []string{"Doc comment line 1", "Doc comment line 2"},
				TrailingTrivia: nil,
			},
		},
		{
			name: "single trailing comment on same line",
			trivia: ast.BaseNode{
				TrailingTrivia: []ast.Comment{
					{Text: "inline field comment", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 5}}},
				},
			},
			expected: TriviaView{
				LeadingTrivia:  nil,
				TrailingTrivia: []string{"inline field comment"},
			},
		},
		{
			name: "multiple trailing comments on same line collapsed with semicolon",
			trivia: ast.BaseNode{
				TrailingTrivia: []ast.Comment{
					{Text: "comment 1", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 5}}},
					{Text: "comment 2", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 5}}},
					{Text: "comment 3", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 5}}},
				},
			},
			expected: TriviaView{
				LeadingTrivia:  nil,
				TrailingTrivia: []string{"comment 1; comment 2; comment 3"},
			},
		},
		{
			name: "block open and close trailing trivia on separate lines",
			trivia: ast.BaseNode{
				TrailingTrivia: []ast.Comment{
					{Text: "open comment 1", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 2}}},
					{Text: "open comment 2", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 2}}},
					{Text: "close comment 1", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 8}}},
					{Text: "close comment 2", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 8}}},
				},
			},
			expected: TriviaView{
				LeadingTrivia: nil,
				TrailingTrivia: []string{
					"open comment 1; open comment 2",
					"close comment 1; close comment 2",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			result := toTriviaView(tc.trivia)
			require.Equal(t, tc.expected, result)
		})
	}

	t.Run("panics on more than 2 distinct trailing trivia lines", func(t *testing.T) {
		invalidTrivia := ast.BaseNode{
			TrailingTrivia: []ast.Comment{
				{Text: "line 1", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 1}}},
				{Text: "line 2", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 2}}},
				{Text: "line 3", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 3}}},
			},
		}
		require.Panics(t, func() {
			toTriviaView(invalidTrivia)
		})
	})
}

func TestVisibilityMapping(t *testing.T) {
	owner := &resolver.EntitySymbol{
		FQN: "Account",
		AST: &ast.Entity{Identifier: targetRefName("Account")},
	}

	t.Run("field visibility", func(t *testing.T) {
		fPublic := toFieldView(owner, &ast.FieldDeclaration{
			Visibility: ast.VisibilityPublic,
		}, &dialect.GoField{
			Name: "id",
			Type: &dialect.GoTypeRef{Typ: dialect.KindNamed, Name: "string"},
		}, &FileView{})
		require.Equal(t, "Id", fPublic.Name)
		require.Empty(t, fPublic.TrailingTrivia)

		fPrivate := toFieldView(owner, &ast.FieldDeclaration{
			Visibility: ast.VisibilityPrivate,
		}, &dialect.GoField{
			Name: "password",
			Type: &dialect.GoTypeRef{Typ: dialect.KindNamed, Name: "string"},
		}, &FileView{})
		require.Equal(t, "password", fPrivate.Name)
		require.Equal(t, []string{"private"}, fPrivate.TrailingTrivia)

		fProtected := toFieldView(owner, &ast.FieldDeclaration{
			Visibility: ast.VisibilityProtected,
		}, &dialect.GoField{
			Name: "token",
			Type: &dialect.GoTypeRef{Typ: dialect.KindNamed, Name: "string"},
		}, &FileView{})
		require.Equal(t, "token", fProtected.Name)
		require.Equal(t, []string{"protected"}, fProtected.TrailingTrivia)

		fPackage := toFieldView(owner, &ast.FieldDeclaration{
			Visibility: ast.VisibilityPackage,
		}, &dialect.GoField{
			Name: "internalState",
			Type: &dialect.GoTypeRef{Typ: dialect.KindNamed, Name: "string"},
		}, &FileView{})
		require.Equal(t, "internalState", fPackage.Name)
		require.Empty(t, fPackage.TrailingTrivia)

		fPrivateWithComment := toFieldView(owner, &ast.FieldDeclaration{
			BaseNode: ast.BaseNode{
				TrailingTrivia: []ast.Comment{
					{Text: "must be encrypted", Span: tokenizer.SourceSpan{Start: tokenizer.Pos{Line: 1}}},
				},
			},
			Visibility: ast.VisibilityPrivate,
		}, &dialect.GoField{
			Name: "secret",
			Type: &dialect.GoTypeRef{Typ: dialect.KindNamed, Name: "string"},
		}, &FileView{})
		require.Equal(t, "secret", fPrivateWithComment.Name)
		require.Equal(t, []string{"private; must be encrypted"}, fPrivateWithComment.TrailingTrivia)
	})

	t.Run("method visibility", func(t *testing.T) {
		mPublic := toMethodView(owner, &ast.MethodDeclaration{
			Visibility: ast.VisibilityPublic,
		}, &dialect.GoMethod{
			Name: "login",
		}, &FileView{})
		require.Equal(t, "Login", mPublic.Name)
		require.Empty(t, mPublic.TrailingTrivia)

		mProtected := toMethodView(owner, &ast.MethodDeclaration{
			Visibility: ast.VisibilityProtected,
		}, &dialect.GoMethod{
			Name: "validate",
		}, &FileView{})
		require.Equal(t, "validate", mProtected.Name)
		require.Equal(t, []string{"protected"}, mProtected.TrailingTrivia)

		mPrivate := toMethodView(owner, &ast.MethodDeclaration{
			Visibility: ast.VisibilityPrivate,
		}, &dialect.GoMethod{
			Name: "hashPassword",
		}, &FileView{})
		require.Equal(t, "hashPassword", mPrivate.Name)
		require.Equal(t, []string{"private"}, mPrivate.TrailingTrivia)
	})

	t.Run("reparseLabel visibility", func(t *testing.T) {
		name, vis := reparseLabel("+items")
		require.Equal(t, "Items", name)
		require.Equal(t, ast.VisibilityPublic, vis)

		name, vis = reparseLabel("-engine")
		require.Equal(t, "engine", name)
		require.Equal(t, ast.VisibilityPrivate, vis)

		name, vis = reparseLabel("#token")
		require.Equal(t, "token", name)
		require.Equal(t, ast.VisibilityProtected, vis)

		name, vis = reparseLabel("~config")
		require.Equal(t, "config", name)
		require.Equal(t, ast.VisibilityPackage, vis)
	})
}
