package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
)

func TestCommentTriviaAttachment(t *testing.T) {
	input := `@startuml
/' Leading comment for class '/
class Foo {
}

' Single line comment for relationship
Foo --> Bar
@enduml`

	p := &Parser{
		Dialect: dialect.NewGoDialect(),
	}
	diag, err := p.Parse(input)
	require.NoError(t, err)
	require.NotNil(t, diag)

	var fooEntity *ast.Entity
	var relStmt *ast.Relationship

	for _, stmt := range diag.Statements {
		switch s := stmt.(type) {
		case *ast.Entity:
			if s.Identifier.EntityName() == "Foo" {
				fooEntity = s
			}
		case *ast.Relationship:
			relStmt = s
		}
	}

	require.NotNil(t, fooEntity, "Expected Foo entity to be present")
	require.NotEmpty(t, fooEntity.LeadingTrivia, "Expected leading trivia on Foo entity")
	require.Contains(t, fooEntity.LeadingTrivia[0].Text, "Leading comment for class")

	require.NotNil(t, relStmt, "Expected relationship statement to be present")
	require.NotEmpty(t, relStmt.LeadingTrivia, "Expected leading trivia on relationship statement")
	require.Contains(t, relStmt.LeadingTrivia[0].Text, "Single line comment for relationship")
}

func TestMemberAndTrailingTriviaAttachment(t *testing.T) {
	input := `@startuml
class Example {
  ' Field leading comment
  +id string ' Field trailing comment
  -- Methods --
  ' Method leading comment
  +DoWork() error ' Method trailing comment
} ' Class closing trailing comment
@enduml`

	p := &Parser{
		Dialect: dialect.NewGoDialect(),
	}
	diag, err := p.Parse(input)
	require.NoError(t, err)
	require.NotNil(t, diag)

	var exampleEntity *ast.Entity
	for _, stmt := range diag.Statements {
		if ent, ok := stmt.(*ast.Entity); ok && ent.Identifier.EntityName() == "Example" {
			exampleEntity = ent
			break
		}
	}
	require.NotNil(t, exampleEntity, "Expected Example entity")
	require.NotEmpty(t, exampleEntity.TrailingTrivia, "Expected trailing trivia on Example entity closing")
	require.Contains(t, exampleEntity.TrailingTrivia[0].Text, "Class closing trailing comment")

	require.Len(t, exampleEntity.Members, 3, "Expected 3 members: field, separator, method")

	// 1. Field
	field, ok := exampleEntity.Members[0].(*ast.FieldDeclaration)
	require.True(t, ok, "Expected FieldDeclaration")
	require.NotEmpty(t, field.LeadingTrivia, "Expected leading trivia on field")
	require.Contains(t, field.LeadingTrivia[0].Text, "Field leading comment")
	require.NotEmpty(t, field.TrailingTrivia, "Expected trailing trivia on field")
	require.Contains(t, field.TrailingTrivia[0].Text, "Field trailing comment")

	// 2. Separator
	sep, ok := exampleEntity.Members[1].(*ast.ClassSeparator)
	require.True(t, ok, "Expected ClassSeparator")
	require.Equal(t, "Methods", sep.Label)

	// 3. Method
	method, ok := exampleEntity.Members[2].(*ast.MethodDeclaration)
	require.True(t, ok, "Expected MethodDeclaration")
	require.NotEmpty(t, method.LeadingTrivia, "Expected leading trivia on method")
	require.Contains(t, method.LeadingTrivia[0].Text, "Method leading comment")
	require.NotEmpty(t, method.TrailingTrivia, "Expected trailing trivia on method")
	require.Contains(t, method.TrailingTrivia[0].Text, "Method trailing comment")
}

func TestMultipleTrailingCommentsAttachment(t *testing.T) {
	input := `@startuml
class User { /' open comment 1 '/ /' open comment 2 '/
  +id string /' field comment 1 '/ /' field comment 2 '/
} /' close comment 1 '/ /' close comment 2 '/
@enduml`

	p := &Parser{
		Dialect: dialect.NewGoDialect(),
	}
	diag, err := p.Parse(input)
	require.NoError(t, err)
	require.NotNil(t, diag)

	var userEntity *ast.Entity
	for _, stmt := range diag.Statements {
		if ent, ok := stmt.(*ast.Entity); ok && ent.Identifier.EntityName() == "User" {
			userEntity = ent
			break
		}
	}
	require.NotNil(t, userEntity)

	// Trailing trivia on User entity should contain 4 tokens (2 from opening line, 2 from closing line)
	require.Len(t, userEntity.TrailingTrivia, 4)
	require.Contains(t, userEntity.TrailingTrivia[0].Text, "open comment 1")
	require.Contains(t, userEntity.TrailingTrivia[1].Text, "open comment 2")
	require.Contains(t, userEntity.TrailingTrivia[2].Text, "close comment 1")
	require.Contains(t, userEntity.TrailingTrivia[3].Text, "close comment 2")

	// Field should have 2 trailing comments on the same line
	require.Len(t, userEntity.Members, 1)
	field, ok := userEntity.Members[0].(*ast.FieldDeclaration)
	require.True(t, ok)
	require.Len(t, field.TrailingTrivia, 2)
	require.Contains(t, field.TrailingTrivia[0].Text, "field comment 1")
	require.Contains(t, field.TrailingTrivia[1].Text, "field comment 2")
}

