package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

func TestRecovery(t *testing.T) {
	t.Run("diagram level recovers past bad statement and parses subsequent statements", func(t *testing.T) {
		input := `@startuml
class ValidA
@invalid_command
class ValidB
@enduml`

		p := NewParser(dialect.NewGoDialect())
		diag, err := p.Parse(input)

		require.Error(t, err, "expected Parse to return an error when diagram contains diagnostics")
		require.NotNil(t, diag, "resilient parser should always return non-nil Diagram")
		require.True(t, diag.HasErrors())
		require.Len(t, diag.Diagnostics, 1)

		// Statements: startuml, ValidA, BadStatement, ValidB, enduml
		require.Len(t, diag.Statements, 5)

		entA, ok := diag.Statements[1].(*ast.Entity)
		require.True(t, ok)
		assertASTEqual(t, identRef("ValidA"), entA.Identifier)

		badStmt, ok := diag.Statements[2].(*ast.BadStatement)
		require.True(t, ok, "expected 3rd statement to be BadStatement")
		require.NotNil(t, badStmt.Diagnostic)
		require.NotEmpty(t, badStmt.Tokens, "collector should capture tokens of bad statement")
		require.Equal(t, tokenizer.AT, badStmt.Tokens[0].Type)

		entB, ok := diag.Statements[3].(*ast.Entity)
		require.True(t, ok)
		assertASTEqual(t, identRef("ValidB"), entB.Identifier)

		endBound, ok := diag.Statements[4].(*ast.DiagramBound)
		require.True(t, ok)
		require.False(t, endBound.IsStart)
	})

	t.Run("container level recovers past bad statement in body", func(t *testing.T) {
		input := `@startuml
package mypkg {
  class InnerA
  @bad_token
  class InnerB
}
@enduml`

		p := NewParser(dialect.NewGoDialect())
		diag, err := p.Parse(input)

		require.Error(t, err)
		require.NotNil(t, diag)
		require.True(t, diag.HasErrors())
		require.Len(t, diag.Diagnostics, 1)

		// Statements: startuml, Container, enduml
		require.Len(t, diag.Statements, 3)

		cont, ok := diag.Statements[1].(*ast.Container)
		require.True(t, ok)
		assertASTEqual(t, identRef("mypkg"), cont.Identifier)
		require.Len(t, cont.Statements, 3)

		innerA, ok := cont.Statements[0].(*ast.Entity)
		require.True(t, ok)
		assertASTEqual(t, identRef("InnerA"), innerA.Identifier)

		badStmt, ok := cont.Statements[1].(*ast.BadStatement)
		require.True(t, ok)
		require.NotNil(t, badStmt.Diagnostic)

		innerB, ok := cont.Statements[2].(*ast.Entity)
		require.True(t, ok)
		assertASTEqual(t, identRef("InnerB"), innerB.Identifier)
	})

	t.Run("entity level recovers past bad member in body", func(t *testing.T) {
		input := `@startuml
class Service {
  +id string
  name : invalid
  +Execute() error
}
@enduml`

		p := NewParser(dialect.NewGoDialect())
		diag, err := p.Parse(input)

		require.Error(t, err)
		require.NotNil(t, diag)
		require.True(t, diag.HasErrors())
		require.Len(t, diag.Diagnostics, 1)

		ent, ok := diag.Statements[1].(*ast.Entity)
		require.True(t, ok)
		assertASTEqual(t, identRef("Service"), ent.Identifier)
		require.Len(t, ent.Members, 3)

		field, ok := ent.Members[0].(*ast.FieldDeclaration)
		require.True(t, ok)
		require.Equal(t, "id", field.Field.FieldName())

		badMember, ok := ent.Members[1].(*ast.BadMember)
		require.True(t, ok)
		require.NotNil(t, badMember.Diagnostic)
		require.NotEmpty(t, badMember.Tokens)

		method, ok := ent.Members[2].(*ast.MethodDeclaration)
		require.True(t, ok)
		require.Equal(t, "Execute", method.Method.MethodName())
	})

	t.Run("multiple errors across different levels are all collected", func(t *testing.T) {
		input := `@startuml
@bad_root
class Service {
  invalid_member : foo
  +ValidField int
}
package pkg {
  @bad_container_stmt
  class ValidInner
}
@enduml`

		p := NewParser(dialect.NewGoDialect())
		diag, err := p.Parse(input)

		require.Error(t, err)
		require.NotNil(t, diag)
		require.True(t, diag.HasErrors())
		// 3 diagnostics: root statement, entity member, container statement
		require.Len(t, diag.Diagnostics, 3)
	})
}
