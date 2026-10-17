// Package formatter implements PlantUML formatter.
package formatter

import (
	"strings"
	"unicode"

	"yur4uwe/pac/pkg/parser"
	"yur4uwe/pac/pkg/parser/ast"
	"yur4uwe/pac/pkg/parser/dialect"
	"yur4uwe/pac/pkg/tokenizer"
)

type formatterState struct {
	source   []rune
	buf      strings.Builder
	disabled bool
	tabDepth int
	indent   int
	useTabs  bool

	lastNode ast.Node

	// recollected from the AST
	packageSeparator string
}

func Format(src string, opts ...FormatOptions) (string, error) {
	var opt FormatOptions
	if len(opts) == 0 {
		opt = DefaultFormatOptions()
	} else {
		opt = opts[0]
	}
	tree, err := parser.NewParser(dialect.LaxDialect{}).Parse(src)
	if err != nil {
		return "", err
	}

	state := &formatterState{
		source:  []rune(src),
		buf:     strings.Builder{},
		indent:  opt.IndentSize,
		useTabs: opt.UseTabs,

		packageSeparator: ".",
	}

	for _, node := range tree.Statements {
		formatNode(state, node)
	}

	return state.buf.String(), nil
}

func formatNode(s *formatterState, node ast.Node) {
	if s.lastNode != nil && needsBlankLine(s.lastNode, node) {
		s.buf.WriteByte('\n')
	}

	leadingTrivia := node.GetLeadingTrivia()
	for _, comment := range leadingTrivia {
		// Check leading trivia for pragmas
		if strings.Contains(comment.Literal, "pac:fmt:off") {
			s.disabled = true
		} else if strings.Contains(comment.Literal, "pac:fmt:on") {
			s.disabled = false
		}
		s.emitTrivia(comment)
		s.buf.WriteByte('\n')
	}

	if s.disabled {
		s.emitRawSpan(node.Span())
	} else {
		switch node := node.(type) {
		case ast.Statement:
			s.formatStatement(node)
		case ast.Member:
			s.formatMember(node)
		}
	}

	s.emitBlockEndTrivia(node)
	s.buf.WriteString("\n")

	s.lastNode = node
}

func (s *formatterState) formatMember(member ast.Member) {
	switch mem := member.(type) {
	case *ast.ClassSeparator:
		printClassSeparator(s, mem)
	case *ast.FieldDeclaration:
		printField(s, mem)
	case *ast.MethodDeclaration:
		printMethod(s, mem)
	default:
		panic("unreachable")
	}
}

func (s *formatterState) formatStatement(stmt ast.Statement) {
	switch st := stmt.(type) {
	case *ast.DiagramBound:
		printDiagramBound(s, st)
	case *ast.TextBlock:
		printTextBlock(s, st)
	case *ast.UnhandledStatement:
		s.emitRawSpan(st.NodeSpan)
	case *ast.Entity:
		printEntity(s, st)
	case *ast.InlineMember:
		printInlineMember(s, st)
	case *ast.Container:
		printContainer(s, st)
	case *ast.Relationship:
		printRelationship(s, st)
	case *ast.Note:
		printNote(s, st)
	case *ast.StyleDeclaration:
		s.emitWithIndent(st.Property)
		s.buf.WriteString(": ")
		s.buf.WriteString(st.Value)
	case *ast.StyleRule:
		printStyleRule(s, st)
	case *ast.StyleBlock:
		printStyleBlock(s, st)
	case *ast.SkinparamSetting:
		printSkinparamSetting(s, st)
	case *ast.SkinparamBlock:
		printSkinparamBlock(s, st)
	case *ast.GenericCommand:
		printGenericCommand(s, st)
	case *ast.IncludeDirective:
		printIncludeDirective(s, st)
	case *ast.ScaleCommand:
		printScaleCommand(s, st)
	case *ast.VisibilityCommand:
		printVisibilityCommand(s, st)
	case *ast.SetCommand:
		printSetCommand(s, st)
	case *ast.DirectionCommand:
		printDirectionCommand(s, st)
	default:
		panic("unimplemented")
	}
}

func (s *formatterState) emitRawSpan(span tokenizer.SourceSpan) {
	// Find the previous newline to calculate original indent for relative indentation
	var origIndent uint
	if span.Start.Offset > 0 {
		isLeading := true
		for i := int(span.Start.Offset) - 1; i >= 0; i-- {
			if s.source[i] == '\n' {
				break
			}
			if !unicode.IsSpace(s.source[i]) {
				isLeading = false
				break
			}
			origIndent++
		}
		if !isLeading {
			origIndent = 0
		}
	}

	inputSpan := s.source[span.Start.Offset : span.End.Offset+1]
	var inputLines [][]rune
	var localLineIndents []int
	var currLine []rune
	currLineIndent := 0

	for i, r := range inputSpan {
		if len(currLine) == 0 && unicode.IsSpace(r) {
			currLineIndent++
			continue
		}

		if inputSpan[i] == '\n' || i == len(inputSpan)-1 {
			inputLines = append(inputLines, currLine)
			localLineIndents = append(localLineIndents, currLineIndent-int(origIndent))
			currLineIndent = 0
			currLine = nil
		} else {
			currLine = append(currLine, inputSpan[i])
		}
	}

	for i, line := range inputLines {
		if i == 0 {
			s.writeIndent()
			s.buf.WriteString(string(line))
			if len(inputLines) > 1 {
				s.buf.WriteRune('\n')
			}
			continue
		}

		var currTab string
		if s.useTabs {
			currTab = strings.Repeat("\t", s.tabDepth)
		} else {
			sourceIndent := max(s.getIndent()+localLineIndents[i], 0)
			currTab = strings.Repeat(" ", sourceIndent)
		}
		s.buf.WriteString(currTab)
		s.buf.WriteString(string(line))
		if i < len(inputLines)-1 {
			s.buf.WriteRune('\n')
		}
	}
}
