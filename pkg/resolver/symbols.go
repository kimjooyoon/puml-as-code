package resolver

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
)

type HeadKind int

// ORDER MATTERS
const (
	HeadNone           HeadKind = iota
	HeadNavigation              // Arrow: '<' or '>' (Association / Dependency)
	HeadContainment             // Diamond: '*' or 'o' (Composition / Aggregation)
	HeadGeneralization          // Triangle: '|' (Inheritance / Realization)
)

func classifyHead(arrow rune) HeadKind {
	switch arrow {
	case '|':
		return HeadGeneralization
	case '*', 'o':
		return HeadContainment
	case '<', '>':
		return HeadNavigation
	default:
		return HeadNone
	}
}

type SymbolTable struct {
	Entities      []*EntitySymbol
	Relationships []*RelationshipSymbol
	Notes         []*ast.Note // Floating or unlinked notes

	// Lookup by identifier, alias, or qualified name
	lookup map[string]*EntitySymbol
}

func (tbl *SymbolTable) Lookup(name string) *EntitySymbol {
	return tbl.lookup[name]
}

func (tbl *SymbolTable) LookupByRef(ref ast.TargetRef, pkgPath []string) *EntitySymbol {
	pkgPathStrings := ref.PackagePathStrings()
	entityName := ref.EntityName()

	if len(pkgPathStrings) > 0 {
		if ent, ok := tbl.lookup[FQN(entityName, pkgPathStrings)]; ok {
			return ent
		}
		if len(pkgPath) > 0 {
			relPkg := append(slices.Clone(pkgPath), pkgPathStrings...)
			if ent, ok := tbl.lookup[FQN(entityName, relPkg)]; ok {
				return ent
			}
		}
		return nil
	}

	// 1. Current package scope
	if len(pkgPath) > 0 {
		if ent, ok := tbl.lookup[FQN(entityName, pkgPath)]; ok {
			return ent
		}
	}

	// 2. Parent package scopes up to root
	for i := len(pkgPath) - 1; i >= 1; i-- {
		if ent, ok := tbl.lookup[FQN(entityName, pkgPath[:i])]; ok {
			return ent
		}
	}

	// 3. Root scope
	if ent, ok := tbl.lookup[entityName]; ok {
		return ent
	}

	// 4. Global search across all known entities for unique simple name match
	var candidate *EntitySymbol
	for _, ent := range tbl.Entities {
		simpleName := ent.FQN
		if idx := strings.LastIndex(ent.FQN, "."); idx != -1 {
			simpleName = ent.FQN[idx+1:]
		}
		if simpleName == entityName || (ent.AST != nil && ent.AST.Identifier.EntityName() == entityName) {
			if candidate == nil {
				candidate = ent
			} else if candidate.FQN != ent.FQN {
				// Ambiguous match across multiple packages
				return nil
			}
		}
	}

	return candidate
}

func (tbl *SymbolTable) FindOrCreateByRef(ref ast.TargetRef, pkgPath []string) *EntitySymbol {
	if ent := tbl.LookupByRef(ref, pkgPath); ent != nil {
		return ent
	}

	targetPkg := ref.PackagePathStrings()
	if len(targetPkg) == 0 && len(pkgPath) > 0 {
		targetPkg = slices.Clone(pkgPath)
	}

	newEnt := &EntitySymbol{
		FQN:         FQN(ref.EntityName(), targetPkg),
		PackagePath: slices.Clone(targetPkg),
	}
	tbl.Entities = append(tbl.Entities, newEnt)
	tbl.lookup[newEnt.FQN] = newEnt
	return newEnt
}

type EntitySymbol struct {
	FQN         string
	PackagePath []string
	AST         *ast.Entity
	Notes       []*ast.Note
	MemberNotes map[string][]*ast.Note
}

func FQN(name string, pkgPath []string) string {
	if len(pkgPath) > 0 {
		return strings.Join(append(pkgPath, name), ".")
	}
	return name
}

func newEntitySymbol(ent *ast.Entity, pkgPath []string) *EntitySymbol {
	combinedPkgPath := append(slices.Clone(pkgPath), ent.Identifier.PackagePathStrings()...)
	return &EntitySymbol{
		FQN:         FQN(ent.Identifier.EntityName(), combinedPkgPath),
		AST:         ent,
		PackagePath: combinedPkgPath,
	}
}

type RelationshipSymbol struct {
	AST        *ast.Relationship
	Source     *EntitySymbol // Subclass / Implementer / Owner
	Target     *EntitySymbol // Superclass / Interface / Contained
	SourceMult ast.Cardinality
	TargetMult ast.Cardinality
	Type       ast.RelationType
	Notes      []*ast.Note // Notes on the relationship/link
}

func newRelationshipSymbol(tbl *SymbolTable, rel *ast.Relationship, pkgPath []string) (*RelationshipSymbol, error) {
	headL := classifyHead(rel.LArrow)
	headR := classifyHead(rel.RArrow)

	if headL >= HeadContainment && headR >= HeadContainment {
		if headL == HeadGeneralization && headR == HeadGeneralization {
			return nil, fmt.Errorf("cannot have bidirectional inheritance: %s and %s", rel.LHS.EntityName(), rel.RHS.EntityName())
		}
		if headL == HeadContainment && headR == HeadContainment {
			return nil, fmt.Errorf("cannot have bidirectional composition/aggregation: %s and %s", rel.LHS.EntityName(), rel.RHS.EntityName())
		}
		return nil, fmt.Errorf("conflicting relationship heads: cannot combine %s and %s on the same line", rel.TypeLHS, rel.TypeRHS)
	}

	symb := &RelationshipSymbol{
		AST: rel,
	}

	switch {
	case headL == HeadGeneralization:
		if headR == HeadNavigation {
			log.Printf("warning: ignoring redundant association arrow on inheritance %s <|%c%c %s", rel.LHS.EntityName(), rel.Body, rel.Body, rel.RHS.EntityName())
		}
		symb.Type = rel.TypeLHS
		symb.Source = tbl.FindOrCreateByRef(rel.RHS, pkgPath) // Subclass / Implementer
		symb.SourceMult = rel.MultRHS
		symb.Target = tbl.FindOrCreateByRef(rel.LHS, pkgPath) // Superclass / Interface
		symb.TargetMult = rel.MultLHS

	case headR == HeadGeneralization:
		if headL == HeadNavigation {
			log.Printf("warning: ignoring redundant association arrow on inheritance %s %c%c|> %s", rel.LHS.EntityName(), rel.Body, rel.Body, rel.RHS.EntityName())
		}
		symb.Type = rel.TypeRHS
		symb.Source = tbl.FindOrCreateByRef(rel.LHS, pkgPath) // Subclass / Implementer
		symb.SourceMult = rel.MultLHS
		symb.Target = tbl.FindOrCreateByRef(rel.RHS, pkgPath) // Superclass / Interface
		symb.TargetMult = rel.MultRHS

	case headL == HeadContainment:
		if headR == HeadNavigation {
			log.Printf("warning: ignoring redundant association arrow on containment %s %c-- %s", rel.LHS.EntityName(), rel.LArrow, rel.RHS.EntityName())
		}
		symb.Type = rel.TypeLHS
		symb.Source = tbl.FindOrCreateByRef(rel.LHS, pkgPath) // Owner
		symb.SourceMult = rel.MultLHS
		symb.Target = tbl.FindOrCreateByRef(rel.RHS, pkgPath) // Contained
		symb.TargetMult = rel.MultRHS

	case headR == HeadContainment:
		if headL == HeadNavigation {
			log.Printf("warning: ignoring redundant association arrow on containment %s --%c %s", rel.LHS.EntityName(), rel.RArrow, rel.RHS.EntityName())
		}
		symb.Type = rel.TypeRHS
		symb.Source = tbl.FindOrCreateByRef(rel.RHS, pkgPath) // Owner
		symb.SourceMult = rel.MultRHS
		symb.Target = tbl.FindOrCreateByRef(rel.LHS, pkgPath) // Contained
		symb.TargetMult = rel.MultLHS

	case headL == HeadNavigation && headR == HeadNavigation:
		symb.Type = rel.TypeLHS
		symb.Source = tbl.FindOrCreateByRef(rel.LHS, pkgPath)
		symb.SourceMult = rel.MultLHS
		symb.Target = tbl.FindOrCreateByRef(rel.RHS, pkgPath)
		symb.TargetMult = rel.MultRHS

	case headR == HeadNavigation: // A --> B
		symb.Type = rel.TypeRHS
		symb.Source = tbl.FindOrCreateByRef(rel.LHS, pkgPath)
		symb.SourceMult = rel.MultLHS
		symb.Target = tbl.FindOrCreateByRef(rel.RHS, pkgPath)
		symb.TargetMult = rel.MultRHS

	case headL == HeadNavigation: // A <-- B
		symb.Type = rel.TypeLHS
		symb.Source = tbl.FindOrCreateByRef(rel.RHS, pkgPath)
		symb.SourceMult = rel.MultRHS
		symb.Target = tbl.FindOrCreateByRef(rel.LHS, pkgPath)
		symb.TargetMult = rel.MultLHS

	default: // Undirected A -- B
		symb.Source = tbl.FindOrCreateByRef(rel.LHS, pkgPath)
		symb.SourceMult = rel.MultLHS
		symb.Target = tbl.FindOrCreateByRef(rel.RHS, pkgPath)
		symb.TargetMult = rel.MultRHS
	}

	return symb, nil
}

func createImplicitEntity(ref ast.TargetRef, pkgPath []string) *EntitySymbol {
	ent := &ast.Entity{
		Identifier: ref,
	}
	implicitEntity := newEntitySymbol(ent, pkgPath)
	return implicitEntity
}

func attachNote(tbl *SymbolTable, noteRef, noteTargetRef ast.TargetRef, note *ast.Note, noteLookup map[string]*ast.Note, pkgPath []string) {
	noteTarget := tbl.FindOrCreateByRef(noteTargetRef, pkgPath)
	if noteTargetRef.MemberName() != "" {
		if noteTarget.MemberNotes == nil {
			noteTarget.MemberNotes = make(map[string][]*ast.Note)
		}
		noteTarget.MemberNotes[noteTargetRef.MemberName()] = append(noteTarget.MemberNotes[noteTargetRef.MemberName()], note)
	} else {
		noteTarget.Notes = append(noteTarget.Notes, note)
	}
	delete(noteLookup, noteRef.EntityName())
}

func mergeEntity(target *ast.Entity, incoming *ast.Entity) error {
	target.Members = append(target.Members, incoming.Members...)

	if target.Kind != incoming.Kind {
		// Yeah ... plantuml just does this
		target.Kind = incoming.Kind
	}

	// Properties backfilling
	// Unfortunately, puml overwrites the properties of the earlier declarations
	if incoming.Stereotype != nil {
		target.Stereotype = incoming.Stereotype
	}
	if incoming.Generic != "" {
		target.Generic = incoming.Generic
	}
	if incoming.Color != "" {
		target.Color = incoming.Color
	}
	if !incoming.Alias.IsZero() {
		target.Alias = incoming.Alias
	}
	if len(incoming.Tags) > 0 {
		target.Tags = append(target.Tags, incoming.Tags...)
	}

	return nil
}

func resolveStatements(tbl *SymbolTable, stmts []ast.Statement, pkgPath []string, noteLookup map[string]*ast.Note, namedNotes *[]*ast.Note) error {
	var prevRelationship *RelationshipSymbol
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.Entity:
			combinedPkgPath := append(slices.Clone(pkgPath), s.Identifier.PackagePathStrings()...)
			fqn := FQN(s.Identifier.EntityName(), combinedPkgPath)
			if ent, ok := tbl.lookup[fqn]; ok {
				if ent.AST == nil {
					ent.AST = s
					continue
				}
				if err := mergeEntity(ent.AST, s); err != nil {
					return err
				}
				continue
			}
			ent := newEntitySymbol(s, pkgPath)
			tbl.Entities = append(tbl.Entities, ent)
			tbl.lookup[ent.FQN] = ent
			if !s.Alias.IsZero() {
				tbl.lookup[s.Alias.Literal] = ent
			}
		case *ast.InlineMember:
			ent := tbl.FindOrCreateByRef(s.Target, pkgPath)
			if ent.AST == nil {
				ent.AST = &ast.Entity{
					Identifier: s.Target,
				}
			}
			ent.AST.Members = append(ent.AST.Members, s.Member)
		case *ast.Container:
			containerPath := append(s.Identifier.PackagePathStrings(), s.Identifier.EntityName())
			childPkgPath := append(slices.Clone(pkgPath), containerPath...)
			if err := resolveStatements(tbl, s.Statements, childPkgPath, noteLookup, namedNotes); err != nil {
				return err
			}
		case *ast.Note:
			// get rid of the link notes
			if s.Target != nil && s.Target.EntityName() == "link" {
				if prevRelationship == nil {
					return fmt.Errorf("no relationship found for link note %s", s.Text)
				}
				prevRelationship.Notes = append(prevRelationship.Notes, s)
				continue
			}

			// get rid of the referenced notes
			if !s.Identifier.Name.IsZero() {
				noteLookup[s.Identifier.EntityName()] = s
				*namedNotes = append(*namedNotes, s)
				continue
			}

			// get rid of targeted notes
			if s.Target != nil {
				target := tbl.LookupByRef(*s.Target, pkgPath)
				if target == nil {
					return fmt.Errorf("no entity found for targeted note %v", s)
				}
				if s.Target.MemberName() != "" {
					if target.MemberNotes == nil {
						target.MemberNotes = make(map[string][]*ast.Note)
					}
					target.MemberNotes[s.Target.MemberName()] = append(target.MemberNotes[s.Target.MemberName()], s)
					continue
				}
				target.Notes = append(target.Notes, s)
				continue
			}

			tbl.Notes = append(tbl.Notes, s)
		case *ast.Relationship:
			rel, err := newRelationshipSymbol(tbl, s, pkgPath)
			if err != nil {
				return err
			}
			prevRelationship = rel

			if note, ok := noteLookup[rel.AST.LHS.EntityName()]; ok {
				attachNote(tbl, rel.AST.LHS, rel.AST.RHS, note, noteLookup, pkgPath)
				continue
			} else if note, ok := noteLookup[rel.AST.RHS.EntityName()]; ok {
				attachNote(tbl, rel.AST.RHS, rel.AST.LHS, note, noteLookup, pkgPath)
				continue
			}

			tbl.Relationships = append(tbl.Relationships, rel)
		}
	}
	return nil
}

// ResolveSymbols resolves all symbols in the AST and return flat symbol table.
func ResolveSymbols(diagram *ast.Diagram) (*SymbolTable, error) {
	tbl := &SymbolTable{
		lookup: map[string]*EntitySymbol{},
	}
	noteLookup := map[string]*ast.Note{}
	var namedNotes []*ast.Note
	err := resolveStatements(tbl, diagram.Statements, nil, noteLookup, &namedNotes)
	if err != nil {
		return nil, err
	}
	for _, n := range namedNotes {
		if _, ok := noteLookup[n.Identifier.EntityName()]; ok {
			tbl.Notes = append(tbl.Notes, n)
		}
	}
	return tbl, nil
}
