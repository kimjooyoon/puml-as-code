package resolver

import (
	"testing"

	"yur4uwe/pac/pkg/parser"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"

	"github.com/stretchr/testify/require"
)

func TestResolveSymbols_SeparatorMemberReferences(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		lax          bool
		wantSource   string
		wantTarget   string
		wantLMember  string
		wantRMember  string
		wantEntities int
	}{
		{
			name: "reported_lax_field",
			source: `set separator ::
class User {
  +id: string
}
class Role
User::id --> Role
`,
			lax: true, wantSource: "User", wantTarget: "Role", wantLMember: "id", wantEntities: 2,
		},
		{
			name: "go_field_on_left",
			source: `set separator ::
class User {
  +id string
}
class Role
User::id --> Role
`,
			wantSource: "User", wantTarget: "Role", wantLMember: "id", wantEntities: 2,
		},
		{
			name: "go_field_on_right",
			source: `set separator ::
class User {
  +id string
}
class Role
Role --> User::id
`,
			wantSource: "Role", wantTarget: "User", wantRMember: "id", wantEntities: 2,
		},
		{
			name: "qualified_nested_entity",
			source: `set separator ::
package outer {
  package inner {
    class User {
      +id string
    }
  }
}
class Role
outer::inner::User::id --> Role
`,
			wantSource: "outer.inner.User", wantTarget: "Role", wantLMember: "id", wantEntities: 2,
		},
		{
			name: "parent_package_scope",
			source: `set separator ::
package outer {
  class User {
    +id string
  }
  package inner {
    class Role
    User::id --> Role
  }
}
`,
			wantSource: "outer.User", wantTarget: "outer.inner.Role", wantLMember: "id", wantEntities: 2,
		},
		{
			name: "relative_package_scope",
			source: `set separator ::
package outer {
  package inner {
    class User {
      +id string
    }
  }
  class Role
  inner::User::id --> Role
}
`,
			wantSource: "outer.inner.User", wantTarget: "outer.Role", wantLMember: "id", wantEntities: 2,
		},
		{
			name: "go_method_name",
			source: `set separator ::
class User {
  +Run() error
}
class Role
User::Run --> Role
`,
			wantSource: "User", wantTarget: "Role", wantLMember: "Run", wantEntities: 2,
		},
		{
			name: "lax_method_name",
			source: `set separator ::
class User {
  +Run() string
}
class Role
User::Run --> Role
`,
			lax: true, wantSource: "User", wantTarget: "Role", wantLMember: "Run", wantEntities: 2,
		},
		{
			name: "unknown_member_stays_implicit_entity",
			source: `set separator ::
class User {
  +id string
}
class Role
User::missing --> Role
`,
			wantSource: "User.missing", wantTarget: "Role", wantEntities: 3,
		},
		{
			name: "lax_member_requires_exact_name",
			source: `set separator ::
class User {
  +identity: string
}
class Role
User::id --> Role
`,
			lax: true, wantSource: "User.id", wantTarget: "Role", wantEntities: 3,
		},
		{
			name: "lax_namespace_separator_is_not_field_delimiter",
			source: `set separator ::
class User {
  +std::string value
}
class Role
User::std --> Role
`,
			lax: true, wantSource: "User.std", wantTarget: "Role", wantEntities: 3,
		},
		{
			name: "existing_package_entity",
			source: `set separator ::
package User {
  class id
}
class Role
User::id --> Role
`,
			wantSource: "User.id", wantTarget: "Role", wantEntities: 2,
		},
		{
			name: "existing_entity_wins_over_matching_member",
			source: `set separator ::
class User {
  +id string
}
package User {
  class id
}
class Role
User::id --> Role
`,
			wantSource: "User.id", wantTarget: "Role", wantEntities: 3,
		},
		{
			name: "later_default_separator_keeps_package_reference",
			source: `set separator ::
class User {
  +id string
}
class Role
set separator .
User.id --> Role
`,
			wantSource: "User.id", wantTarget: "Role", wantEntities: 3,
		},
		{
			name: "default_separator_explicit_member",
			source: `class User {
  +id string
}
class Role
User::id --> Role
`,
			wantSource: "User", wantTarget: "Role", wantLMember: "id", wantEntities: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d dialect.Dialect = dialect.NewGoDialect()
			if tt.lax {
				d = dialect.LaxDialect{}
			}
			source := "@startuml\n" + tt.source + "@enduml\n"
			diagram, err := parser.NewParser(d).Parse(source)
			require.NoError(t, err)
			tbl, err := ResolveSymbols(diagram)
			require.NoError(t, err)
			require.Len(t, tbl.Relationships, 1)
			rel := tbl.Relationships[0]
			require.Equal(t, tt.wantSource, rel.Source.FQN)
			require.Equal(t, tt.wantTarget, rel.Target.FQN)
			require.Same(t, tbl.Lookup(tt.wantSource), rel.Source)
			require.Same(t, tbl.Lookup(tt.wantTarget), rel.Target)
			require.Equal(t, tt.wantLMember, rel.AST.LHS.MemberName())
			require.Equal(t, tt.wantRMember, rel.AST.RHS.MemberName())
			require.Len(t, tbl.Entities, tt.wantEntities, "member references must not create extra entities")
			for _, ref := range []ast.TargetRef{rel.AST.LHS, rel.AST.RHS} {
				if ref.Member == nil {
					continue
				}
				require.Equal(t, ref.EntityName(), source[ref.Name.Span.Start.Offset:ref.Name.Span.End.Offset])
				require.Equal(t, ref.MemberName(), source[ref.Member.Name.Span.Start.Offset:ref.Member.Name.Span.End.Offset])
			}
		})
	}
}

func TestResolveSymbols_SeparatorMemberNote(t *testing.T) {
	diagram, err := parser.NewParser(dialect.NewGoDialect()).Parse(`@startuml
set separator ::
class User {
  +id string
}
note right of User::id : Identifier note
@enduml
`)
	require.NoError(t, err)
	tbl, err := ResolveSymbols(diagram)
	require.NoError(t, err)
	require.Len(t, tbl.Entities, 1)
	user := tbl.Lookup("User")
	require.NotNil(t, user)
	require.Empty(t, user.Notes)
	require.Len(t, user.MemberNotes["id"], 1)
	require.Equal(t, "Identifier note", user.MemberNotes["id"][0].Text)
}

func TestResolveSymbols_SeparatorMemberNamedNote(t *testing.T) {
	for _, link := range []string{"N -- User::id", "User::id -- N"} {
		t.Run(link, func(t *testing.T) {
			diagram, err := parser.NewParser(dialect.NewGoDialect()).Parse(`@startuml
set separator ::
class User {
  +id string
}
note "Identifier note" as N
` + link + "\n@enduml\n")
			require.NoError(t, err)
			var relationship *ast.Relationship
			for _, stmt := range diagram.Statements {
				if rel, ok := stmt.(*ast.Relationship); ok {
					relationship = rel
				}
			}
			require.NotNil(t, relationship)
			tbl, err := ResolveSymbols(diagram)
			require.NoError(t, err)
			user := tbl.Lookup("User")
			require.NotNil(t, user)
			require.Nil(t, tbl.Lookup("User.id"))
			require.Empty(t, user.Notes)
			require.Len(t, user.MemberNotes["id"], 1)
			require.Equal(t, "Identifier note", user.MemberNotes["id"][0].Text)
			require.Empty(t, tbl.Relationships)
			require.Empty(t, tbl.Notes)
			memberRef := relationship.LHS
			if memberRef.EntityName() == "N" {
				memberRef = relationship.RHS
			}
			require.Equal(t, "User", memberRef.EntityName())
			require.Empty(t, memberRef.PackagePath)
			require.Equal(t, "id", memberRef.MemberName())
		})
	}
}

func TestResolveSymbols_IncludeSeparatorIsolation(t *testing.T) {
	tests := []struct {
		name, main, included string
	}{
		{
			name: "including_separator_does_not_change_included_reference",
			main: `@startuml
set separator ::
class User {
  +id string
}
!include child.puml
@enduml
`,
			included: "class Role\nUser.id --> Role\n",
		},
		{
			name: "included_separator_does_not_change_following_reference",
			main: `@startuml
class User {
  +id string
}
!include child.puml
User.id --> Role
@enduml
`,
			included: "set separator ::\nclass Role\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagram := parseAndResolve(t, "/project/main.puml", MapFS{
				"/project/main.puml":  []byte(tt.main),
				"/project/child.puml": []byte(tt.included),
			})
			tbl, err := ResolveSymbols(diagram)
			require.NoError(t, err)
			require.Len(t, tbl.Relationships, 1)
			rel := tbl.Relationships[0]
			require.Equal(t, "User.id", rel.Source.FQN)
			require.Empty(t, rel.AST.LHS.MemberName())
			require.Equal(t, "Role", rel.Target.FQN)
			require.Len(t, tbl.Entities, 3)
		})
	}
}

func TestResolveSymbols_SeparatorInlineMemberKeepsEntityTarget(t *testing.T) {
	diagram, err := parser.NewParser(dialect.NewGoDialect()).Parse(`@startuml
set separator ::
class User {
  +id string
}
User::id : +other string
@enduml
`)
	require.NoError(t, err)
	tbl, err := ResolveSymbols(diagram)
	require.NoError(t, err)
	require.Len(t, tbl.Entities, 2)
	user := tbl.Lookup("User")
	require.NotNil(t, user)
	require.Len(t, user.AST.Members, 1)
	require.IsType(t, &ast.FieldDeclaration{}, user.AST.Members[0])
	require.Equal(t, "id", user.AST.Members[0].(*ast.FieldDeclaration).FieldName())
	target := tbl.Lookup("User.id")
	require.NotNil(t, target)
	require.NotNil(t, target.AST)
	require.Len(t, target.AST.Members, 1)
	require.IsType(t, &ast.FieldDeclaration{}, target.AST.Members[0])
	require.Equal(t, "other", target.AST.Members[0].(*ast.FieldDeclaration).FieldName())
	require.Empty(t, tbl.Relationships)
}

func TestResolveSymbols_SeparatorExplicitMethodReference(t *testing.T) {
	const source = `@startuml
set separator ::
class User {
  +Run() string
}
class Role
User::Run() --> Role
@enduml
`
	diagram, err := parser.NewParser(dialect.NewGoDialect()).Parse(source)
	require.NoError(t, err)
	tbl, err := ResolveSymbols(diagram)
	require.NoError(t, err)
	require.Len(t, tbl.Entities, 2)
	require.Len(t, tbl.Relationships, 1)
	rel := tbl.Relationships[0]
	require.Equal(t, "User", rel.Source.FQN)
	require.Equal(t, "Role", rel.Target.FQN)
	ref := rel.AST.LHS
	require.Equal(t, "User", ref.EntityName())
	require.Empty(t, ref.PackagePath)
	require.NotNil(t, ref.Member)
	require.Equal(t, "Run", ref.MemberName())
	require.Len(t, ref.Member.Signature, 2)
	require.Equal(t, "(", ref.Member.Signature[0].Literal)
	require.Equal(t, ")", ref.Member.Signature[1].Literal)
	span := ref.Member.Span()
	require.Equal(t, "Run()", source[span.Start.Offset:span.End.Offset])
	span = ref.Span()
	require.Equal(t, "User::Run()", source[span.Start.Offset:span.End.Offset])
}
