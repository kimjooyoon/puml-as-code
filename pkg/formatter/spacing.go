package formatter

import (
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
)

type statementCategory int

const (
	catUnknown statementCategory = iota
	catDiagramBoundStart
	catDiagramBoundEnd
	catTextBlock
	catStyling
	catDeclaration
	catInlineMemberDecl
	catRelationship
	catCommand
	catDirective
	catNote
)

func isBlockNode(node ast.Node) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *ast.Container:
		return true
	case *ast.Entity:
		return n.Kind != ast.EntityUnknown
	case *ast.StyleBlock, *ast.StyleRule, *ast.SkinparamBlock:
		return true
	case *ast.Note:
		return strings.Contains(n.Text, "\n") || len(n.Text) > 60
	case *ast.TextBlock:
		return strings.Contains(n.Text, "\n") || len(n.Text) > 60
	default:
		return false
	}
}

func categorizeNode(node ast.Node) statementCategory {
	switch n := node.(type) {
	case *ast.DiagramBound:
		if n.IsStart {
			return catDiagramBoundStart
		}
		return catDiagramBoundEnd
	case *ast.TextBlock:
		return catTextBlock
	case *ast.StyleBlock, *ast.StyleRule, *ast.SkinparamBlock, *ast.SkinparamSetting:
		return catStyling
	case *ast.Container:
		return catDeclaration
	case *ast.Entity:
		return catDeclaration
	case *ast.InlineMember:
		return catInlineMemberDecl
	case *ast.Relationship:
		return catRelationship
	case *ast.IncludeDirective:
		return catDirective
	case *ast.Note:
		return catNote
	case *ast.ScaleCommand, *ast.VisibilityCommand, *ast.SetCommand, *ast.DirectionCommand, *ast.GenericCommand:
		return catCommand
	default:
		return catUnknown
	}
}

func nodeStartLine(node ast.Node) int {
	if node == nil {
		return 0
	}
	start := int(node.Span().Start.Line)
	leading := node.GetLeadingTrivia()
	if len(leading) > 0 {
		start = int(leading[0].Span.Start.Line)
	}
	return start
}

func nodeEndLine(node ast.Node) int {
	if node == nil {
		return 0
	}
	end := int(node.Span().End.Line)
	trailing := node.GetTrailingTrivia()
	if len(trailing) <= 0 {
		return end
	}

	lastTrailing := trailing[len(trailing)-1]
	if int(lastTrailing.Span.End.Line) > end {
		return int(lastTrailing.Span.End.Line)
	} else {
		return end
	}
}

func isFieldMember(m ast.Member) bool {
	_, ok := m.(*ast.FieldDeclaration)
	return ok
}

func isMethodMember(m ast.Member) bool {
	_, ok := m.(*ast.MethodDeclaration)
	return ok
}

func isSeparatorMember(m ast.Member) bool {
	_, ok := m.(*ast.ClassSeparator)
	return ok
}

func needsBlankLine(prev, curr ast.Node) bool {
	if prev == nil || curr == nil {
		return false
	}

	prevEnd := nodeEndLine(prev)
	currStart := nodeStartLine(curr)

	// 1. Universal source intent rule:
	// If author explicitly placed blank line(s) in source, preserve 1 blank line
	if prevEnd > 0 && currStart > 0 && currStart > prevEnd+1 {
		return true
	}

	// 2. Member-specific structural spacing
	prevMember, prevIsMem := prev.(ast.Member)
	currMember, currIsMem := curr.(ast.Member)
	if prevIsMem && currIsMem {
		// Around ClassSeparator (-- Section --, == Methods ==, etc.)
		if isSeparatorMember(prevMember) || isSeparatorMember(currMember) {
			return true
		}
		// Field <-> Method transition
		if isFieldMember(prevMember) && isMethodMember(currMember) {
			return true
		}
		if isMethodMember(prevMember) && isFieldMember(currMember) {
			return true
		}
		return false
	}

	catPrev := categorizeNode(prev)
	catCurr := categorizeNode(curr)

	// Diagram bounds: only have blank lines if author put them in source
	if catCurr == catDiagramBoundEnd || catPrev == catDiagramBoundStart {
		return false
	}

	// 3. Statement-specific structural spacing
	if isBlockNode(prev) || isBlockNode(curr) {
		return true
	}

	// Category transitions (e.g. Entity declarations -> Relationships)
	if catPrev != catCurr {
		if (catPrev == catDeclaration || catPrev == catInlineMemberDecl) && catCurr == catRelationship {
			return true
		}
		if catPrev == catRelationship && (catCurr == catDeclaration || catCurr == catInlineMemberDecl) {
			return true
		}
		if catPrev == catTextBlock || catCurr == catTextBlock {
			return true
		}
		if catPrev == catStyling && catCurr != catStyling {
			return true
		}
	}

	return false
}
