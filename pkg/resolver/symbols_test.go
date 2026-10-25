package resolver

import (
	"testing"

	"yur4uwe/pac/pkg/parser/ast"

	"github.com/stretchr/testify/require"
)

func TestResolveSymbols_Basic(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
class MainService
MainService --> User
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	require.Len(t, tbl.Entities, 2)
	require.Len(t, tbl.Relationships, 1)

	mainService := tbl.Entities[0]
	user := tbl.Entities[1]
	require.Equal(t, mainService.FQN, "MainService")
	require.Equal(t, user.FQN, "User")

	require.Equal(t, mainService, tbl.Relationships[0].Source, "expected main service to be source of relationship")
	require.Equal(t, user, tbl.Relationships[0].Target, "expected user to be target of relationship")
}

func TestResolveSymbols_EntityWithAlias(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
class "User Entity" as User
User --> Order
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	user := tbl.Lookup("User")
	require.NotNil(t, user)
	require.Equal(t, "User", user.FQN)
	require.NotNil(t, user.AST)
	require.Equal(t, "\"User Entity\"", user.AST.Alias.Literal)
	require.Equal(t, "User", user.AST.Identifier.EntityName())

	require.Len(t, tbl.Relationships, 1)
	require.Equal(t, user, tbl.Relationships[0].Source)
	require.Equal(t, "Order", tbl.Relationships[0].Target.FQN)
}

func TestResolveSymbols_ImplicitBeforeExplicit(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
User --> Order
class User {
  +id string
}
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	user := tbl.Lookup("User")
	require.NotNil(t, user)
	require.NotNil(t, user.AST, "expected AST to be attached to previously implicit entity")
	require.Len(t, user.AST.Members, 1)
}

func TestResolveSymbols_ContainerScoping(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
package net.http {
  class Client
}
class Outside
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	client := tbl.Lookup("net.http.Client")
	require.NotNil(t, client)
	require.Equal(t, []string{"net", "http"}, client.PackagePath)

	outside := tbl.Lookup("Outside")
	require.NotNil(t, outside)
	require.Empty(t, outside.PackagePath)
}

func TestResolveSymbols_RelationshipValidationErrors(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
A <|--|> B
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	_, err := ResolveSymbols(diag)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bidirectional inheritance")
}

func TestResolveSymbols_MergeEntityDeclarations(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
class Service <<API>> #red

class Service {
  +id string
  +DoSomething() error
}

class Service {
  +extra int
  +AnotherMethod() bool
}
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	require.Len(t, tbl.Entities, 1, "expected Service to be merged into a single entity")

	svc := tbl.Lookup("Service")
	require.NotNil(t, svc)
	require.NotNil(t, svc.AST)
	require.NotNil(t, svc.AST.Stereotype)
	require.Equal(t, "API", svc.AST.Stereotype.Value.Literal)
	require.Equal(t, "#red", svc.AST.Color)
	require.Len(t, svc.AST.Members, 4, "expected members from both class bodies to be merged")
}

func TestResolveSymbols_MergeEntityKind(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
class Service
interface Service {
  +Run() error
}
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	svc := tbl.Lookup("Service")
	require.NotNil(t, svc)
	require.Equal(t, ast.EntityInterface, svc.AST.Kind)
	require.Len(t, svc.AST.Members, 1)
}

func TestResolveSymbols_PackageForest_MultiRoot(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
package auth {
  class Token
}
package storage {
  package db {
    class Client
  }
}
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	token := tbl.Lookup("auth.Token")
	require.NotNil(t, token)

	client := tbl.Lookup("storage.db.Client")
	require.NotNil(t, client)
}

func TestResolveSymbols_NoteRelationships(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
class Service {
  +DoWork() error
}
note "Service level note" as N1
Service -- N1

note "Method level note" as N2
N2 -- Service::DoWork
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	svc := tbl.Lookup("Service")
	require.NotNil(t, svc)
	require.Len(t, svc.Notes, 1)
	require.Equal(t, "Service level note", svc.Notes[0].Text)

	require.NotNil(t, svc.MemberNotes)
	require.Len(t, svc.MemberNotes["DoWork"], 1)
	require.Equal(t, "Method level note", svc.MemberNotes["DoWork"][0].Text)

	require.Empty(t, tbl.Relationships, "expected note links to not be registered as code relationships")
}

func TestResolveSymbols_UnlinkedNamedNotes(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
note "Floating unlinked note" as N1
note "Linked note" as N2
class Service
Service -- N2
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	svc := tbl.Lookup("Service")
	require.NotNil(t, svc)
	require.Len(t, svc.Notes, 1)
	require.Equal(t, "Linked note", svc.Notes[0].Text)

	require.Len(t, tbl.Notes, 1)
	require.Equal(t, "Floating unlinked note", tbl.Notes[0].Text)
}

func TestResolveSymbols_PackageScopedRelationships(t *testing.T) {
	fs := MapFS{
		"/project/main.puml": []byte(`
@startuml
package domain {
  class UserService
  class UserRepository
  UserService --> UserRepository : repo
  UserService --> ImplicitStore
}
@enduml
`),
	}

	diag := parseAndResolve(t, "/project/main.puml", fs)
	tbl, err := ResolveSymbols(diag)
	require.NoError(t, err)

	require.Len(t, tbl.Entities, 3, "expected only 3 entities in domain package, no root ghost entities")
	for _, ent := range tbl.Entities {
		require.Equal(t, []string{"domain"}, ent.PackagePath)
	}

	require.Len(t, tbl.Relationships, 2)
	require.Equal(t, "domain.UserService", tbl.Relationships[0].Source.FQN)
	require.Equal(t, "domain.UserRepository", tbl.Relationships[0].Target.FQN)
	require.Equal(t, "domain.UserService", tbl.Relationships[1].Source.FQN)
	require.Equal(t, "domain.ImplicitStore", tbl.Relationships[1].Target.FQN)
}
