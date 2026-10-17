package formatter

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
)

func (s *formatterState) targetFQN(target ast.TargetRef) string {
	var sb strings.Builder
	for _, pkg := range target.PackagePath {
		sb.WriteString(pkg.Literal)
		sb.WriteString(s.packageSeparator)
	}
	sb.WriteString(target.Name.Literal)
	if target.Member != nil {
		sb.WriteString("::")
		sb.WriteString(target.Member.Name.Literal)
		for _, sigTok := range target.Member.Signature {
			sb.WriteString(sigTok.Literal)
		}
	}
	return sb.String()
}

func (s *formatterState) formatBlockStatements(parent ast.Node, stmts []ast.Statement) {
	s.onNewLevel(func() {
		for _, stmt := range stmts {
			formatNode(s, stmt)
		}
		if parent != nil {
			s.emitInnerBlockTrivia(parent)
		}
	})
}

func (s *formatterState) formatBlockMembers(parent ast.Node, members []ast.Member) {
	s.onNewLevel(func() {
		for _, member := range members {
			formatNode(s, member)
		}
		if parent != nil {
			s.emitInnerBlockTrivia(parent)
		}
	})
}

func visibilitySymbol(v ast.VisibilityKind) string {
	switch v {
	case ast.VisibilityPublic:
		return "+"
	case ast.VisibilityPrivate:
		return "-"
	case ast.VisibilityProtected:
		return "#"
	case ast.VisibilityPackage:
		return "~"
	default:
		return ""
	}
}

func arrowDirection(d ast.DirectionKind) string {
	switch d {
	case ast.DirectionTop:
		return "up"
	case ast.DirectionBottom:
		return "down"
	case ast.DirectionLeft:
		return "left"
	case ast.DirectionRight:
		return "right"
	default:
		return ""
	}
}
