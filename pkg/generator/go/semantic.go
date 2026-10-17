package gogenerator

import (
	"fmt"
	"log"
	"slices"

	"yur4uwe/pac/pkg/generator/go/stdlib"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/resolver"
	"yur4uwe/pac/pkg/tokenizer"
)

func isInterface(ent *ast.Entity) bool {
	if ent == nil {
		return false
	}
	return ent.Kind == ast.EntityInterface ||
		ent.Kind == ast.EntityAbstractClass ||
		ent.Kind == ast.EntityProtocol
}

func isStruct(ent *ast.Entity) bool {
	if ent == nil {
		return true // default case
	}
	return ent.Kind == ast.EntityClass ||
		ent.Kind == ast.EntityStruct ||
		ent.Kind == ast.EntityRecord ||
		ent.Kind == ast.EntityDataclass ||
		ent.Kind == ast.EntityException ||
		ent.Kind == ast.EntityEntityClass
}

func isEnum(ent *ast.Entity) bool {
	if ent == nil {
		return false
	}
	return ent.Kind == ast.EntityEnum
}

func tryInterfacePromotion(symb *resolver.EntitySymbol) error {
	if symb.AST == nil {
		symb.AST = &ast.Entity{
			Identifier: ast.TargetRef{
				Name: tokenizer.Token{
					Literal: stdlib.SimpleName(symb.FQN),
				},
			},
			Kind: ast.EntityInterface,
		}
		return nil
	}
	for _, member := range symb.AST.Members {
		switch member.(type) {
		case *ast.FieldDeclaration:
			return fmt.Errorf("cannot realize class %s as an interface in Go: it contains fields", symb.FQN)
		}
	}
	symb.AST.Kind = ast.EntityInterface
	return nil
}

func (GoCodeGenerator) SemanticPass(tbl *resolver.SymbolTable) error {
	for _, ent := range tbl.Entities {
		if ent.AST != nil && isStruct(ent.AST) {
			hasAbstractMethod := slices.ContainsFunc(ent.AST.Members, func(member ast.Member) bool {
				if m, ok := member.(*ast.MethodDeclaration); ok {
					return slices.Contains(m.Modifiers, "abstract")
				}
				return false
			})
			if hasAbstractMethod {
				if err := tryInterfacePromotion(ent); err != nil {
					return fmt.Errorf("cannot have abstract methods on class %s: %w", ent.FQN, err)
				}
				log.Printf("warning: conditions met for interface promotion, %s with abstract methods will become an interface", ent.FQN)
			}
		}
	}

	for _, rel := range tbl.Relationships {
		switch rel.Type {
		case ast.RelationInheritance:
			// Promotion check
			if isInterface(rel.Source.AST) && isStruct(rel.Target.AST) {
				if err := tryInterfacePromotion(rel.Target); err != nil {
					return err
				}
				log.Printf("warning: conditions met for interface promotion, %s will become an interface", rel.Target.FQN)
			} else if isStruct(rel.Source.AST) && isInterface(rel.Target.AST) {
				rel.Type = ast.RelationRealization
			}
		case ast.RelationRealization:
			if isStruct(rel.Target.AST) {
				// Promotion check
				if err := tryInterfacePromotion(rel.Target); err != nil {
					return err
				}
				log.Printf("warning: conditions met for interface promotion, %s will become an interface", rel.Target.FQN)
			} else if isInterface(rel.Source.AST) {
				rel.Type = ast.RelationInheritance
			}
		}
	}

	for _, ent := range tbl.Entities {
		switch {
		case isInterface(ent.AST):
			for _, member := range ent.AST.Members {
				switch member.(type) {
				case *ast.FieldDeclaration:
					return fmt.Errorf("interface or abstract class %s cannot declare fields", ent.FQN)
				}
			}
		case isEnum(ent.AST):
			if len(ent.AST.Members) == 0 {
				return fmt.Errorf("enum %s must declare fields", ent.FQN)
			}
		case stdlib.IsStdlibEntity(ent):
			if ent.AST == nil {
				continue
			}
			if len(ent.AST.Members) > 0 {
				log.Printf("warning: custom members declared on stdlib entity %s will be ignored", ent.FQN)
			}
		}
	}

	tbl.Entities = slices.DeleteFunc(tbl.Entities, func(ent *resolver.EntitySymbol) bool {
		return stdlib.IsStdlibEntity(ent)
	})
	return nil
}
