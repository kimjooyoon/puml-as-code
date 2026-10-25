package stdlib

import (
	"strings"
	"testing"

	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/resolver"

	"github.com/stretchr/testify/assert"
)

func targetRef(name string) ast.TargetRef {
	return ast.TargetRef{
		Name: ast.Ident{
			Literal: name,
		},
	}
}

func TestBuiltinTypes(t *testing.T) {
	assert.True(t, IsBuiltinType("error"))
	assert.True(t, IsBuiltinType("any"))
	assert.True(t, IsBuiltinType("string"))
	assert.True(t, IsBuiltinType("int64"))
	assert.False(t, IsBuiltinType("MyCustomType"))
	assert.False(t, IsBuiltinType("Time"))
}

func TestLookupImportPath(t *testing.T) {
	tests := []struct {
		input        []string
		expectedPath string
		expectedOk   bool
	}{
		{[]string{"time"}, "time", true},
		{[]string{"context"}, "context", true},
		{[]string{"io"}, "io", true},
		{[]string{"http"}, "net/http", true},
		{[]string{"net", "http"}, "net/http", true},
		{[]string{"json"}, "encoding/json", true},
		{[]string{"encoding", "json"}, "encoding/json", true},
		{[]string{"sql"}, "database/sql", true},
		{[]string{"database", "sql"}, "database/sql", true},
		{[]string{"rand"}, "crypto/rand", true},
		{[]string{"crypto", "rand"}, "crypto/rand", true},
		{[]string{"unknown_pkg"}, "", false},
		{[]string{}, "", false},
		{nil, "", false},
	}

	for _, tt := range tests {
		name := strings.Join(tt.input, "/")
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			path, ok := LookupImportPath(tt.input)
			assert.Equal(t, tt.expectedOk, ok)
			if ok {
				assert.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

func TestIsStdlibPackage(t *testing.T) {
	assert.True(t, IsStdlibPackage([]string{"time"}))
	assert.True(t, IsStdlibPackage([]string{"net", "http"}))
	assert.True(t, IsStdlibPackage([]string{"http"}))
	assert.True(t, IsStdlibPackage([]string{"encoding", "json"}))
	assert.True(t, IsStdlibPackage([]string{"json"}))
	assert.False(t, IsStdlibPackage([]string{"models"}))
	assert.False(t, IsStdlibPackage([]string{"auth", "v1"}))
	assert.False(t, IsStdlibPackage([]string{}))
	assert.False(t, IsStdlibPackage(nil))
}

func TestIsStdlibEntity(t *testing.T) {
	tests := []struct {
		name     string
		ent      *resolver.EntitySymbol
		expected bool
	}{
		{
			name: "builtin error",
			ent: &resolver.EntitySymbol{
				FQN: "error",
				AST: &ast.Entity{Identifier: targetRef("error")},
			},
			expected: true,
		},
		{
			name: "time.Time with PackagePath",
			ent: &resolver.EntitySymbol{
				FQN:         "time.Time",
				PackagePath: []string{"time"},
				AST:         &ast.Entity{Identifier: targetRef("Time")},
			},
			expected: true,
		},
		{
			name: "net.http.Request with PackagePath",
			ent: &resolver.EntitySymbol{
				FQN:         "net.http.Request",
				PackagePath: []string{"net", "http"},
				AST:         &ast.Entity{Identifier: targetRef("Request")},
			},
			expected: true,
		},
		{
			name: "time.Time without PackagePath (dot in FQN)",
			ent: &resolver.EntitySymbol{
				FQN: "time.Time",
				AST: &ast.Entity{Identifier: targetRef("Time")},
			},
			expected: true,
		},
		{
			name: "user custom struct",
			ent: &resolver.EntitySymbol{
				FQN:         "models.User",
				PackagePath: []string{"models"},
				AST:         &ast.Entity{Identifier: targetRef("User")},
			},
			expected: false,
		},
		{
			name: "user custom root struct",
			ent: &resolver.EntitySymbol{
				FQN: "Order",
				AST: &ast.Entity{Identifier: targetRef("Order")},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsStdlibEntity(tt.ent))
		})
	}
}
