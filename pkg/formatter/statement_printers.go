package formatter

import (
	"fmt"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
)

func printNote(s *formatterState, note *ast.Note) {
	isMultiline := strings.Contains(note.Text, "\n")
	// Anstract limit where note is too long to be inline
	isTooLong := len(note.Text) > 60

	if isMultiline || isTooLong {
		printBlockNote(s, note)
	} else {
		printInlineNote(s, note)
	}
}

func printNoteHeader(s *formatterState, note *ast.Note, isBlock bool) {
	s.emitWithIndent("note ")
	switch note.Kind {
	case ast.NoteAlias:
		if !isBlock {
			s.buf.WriteRune('"')
			s.buf.WriteString(note.Text)
			s.buf.WriteString("\" ")
		}
		fmt.Fprintf(&s.buf, "as %s", note.Identifier.EntityName())
	case ast.NoteTargeted:
		s.buf.WriteString(note.Direction.String())
		if note.Target != nil {
			fmt.Fprintf(&s.buf, " of %s", s.targetFQN(*note.Target))
		}
	case ast.NoteLink:
		if note.Direction != ast.DirectionUnknown {
			s.buf.WriteString(note.Direction.String())
			s.buf.WriteByte(' ')
		}
		s.buf.WriteString("on link")
	default:
		panic("unreachable")
	}
	if note.Color != "" {
		s.buf.WriteByte(' ')
		s.buf.WriteString(note.Color)
	}
}

func printInlineNote(s *formatterState, note *ast.Note) {
	printNoteHeader(s, note, false)
	if note.Kind == ast.NoteAlias {
		return
	}
	s.buf.WriteString(" : ")
	s.buf.WriteString(note.Text)
}

func printBlockNote(s *formatterState, note *ast.Note) {
	printNoteHeader(s, note, true)
	s.emitBlockStartTrivia(note)
	s.buf.WriteString("\n")
	s.buf.WriteString(note.Text)
	s.buf.WriteString("\n")
	s.emitWithIndent("end note")
}

func printDiagramBound(s *formatterState, bound *ast.DiagramBound) {
	if bound.IsStart {
		s.buf.WriteString("@start")
	} else {
		s.buf.WriteString("@end")
	}
	s.buf.WriteString(bound.Type)

	if len(bound.Params) > 0 {
		s.buf.WriteString("(")
		for i, param := range bound.Params {
			if i > 0 {
				s.buf.WriteString(", ")
			}
			s.buf.WriteString(param.Key)
			s.buf.WriteString("=")
			s.buf.WriteString(param.Value)
		}
		s.buf.WriteString(")")
	} else if bound.ID != "" {
		s.buf.WriteString("(id=")
		s.buf.WriteString(bound.ID)
		s.buf.WriteString(")")
	}

	if bound.Tools != nil {
		s.buf.WriteString("{")
		s.buf.WriteString(bound.Tools.File)
		if bound.Tools.Caption != "" {
			fmt.Fprintf(&s.buf, ", %s", bound.Tools.Caption)
		}
		for _, opt := range bound.Tools.Options {
			fmt.Fprintf(&s.buf, ", %s=%s", opt.Key, opt.Value)
		}
		s.buf.WriteString("}")
	} else if bound.TrailingName != "" {
		s.buf.WriteString(" ")
		s.buf.WriteString(bound.TrailingName)
	}
}

func printScaleCommand(s *formatterState, scaleCmd *ast.ScaleCommand) {
	s.emitWithIndent("scale ")
	if scaleCmd.IsMax {
		s.buf.WriteString("max ")
	}
	s.buf.WriteString(scaleCmd.Lhs)
	if scaleCmd.Sep != "" {
		if scaleCmd.Sep != "." {
			fmt.Fprintf(&s.buf, " %s ", scaleCmd.Sep)
		} else {
			s.buf.WriteString(scaleCmd.Sep)
		}
		s.buf.WriteString(scaleCmd.Rhs)
	}
	if scaleCmd.Unit != "" {
		s.buf.WriteByte(' ')
		s.buf.WriteString(scaleCmd.Unit)
	}
}

func printSkinparamBlock(s *formatterState, block *ast.SkinparamBlock) {
	if block.Keyword != "" {
		s.emitWithIndent(block.Keyword)
		s.buf.WriteByte(' ')
	} else {
		s.writeIndent()
	}
	blockName := block.Name.Literal
	if blockName != "" {
		s.buf.WriteString(blockName)
		s.buf.WriteByte(' ')
	}
	if block.Stereotype != nil {
		str := block.Stereotype.String()
		if str != "" {
			fmt.Fprintf(&s.buf, "<<%s>> ", str)
		}
	}
	s.buf.WriteString("{")
	s.emitBlockStartTrivia(block)
	s.buf.WriteByte('\n')

	s.formatBlockStatements(block, block.Statements)

	s.emitWithIndent("}")
}

func printSkinparamSetting(s *formatterState, setting *ast.SkinparamSetting) {
	s.writeIndent()
	if setting.Keyword != "" {
		s.buf.WriteString(setting.Keyword)
		s.buf.WriteByte(' ')
	}
	s.buf.WriteString(setting.Name.Literal)
	s.buf.WriteByte(' ')
	if setting.Stereotype != nil {
		str := setting.Stereotype.String()
		if str != "" {
			s.buf.WriteString("<<")
			s.buf.WriteString(str)
			s.buf.WriteString(">> ")
		}
	}
	s.buf.WriteString(setting.Value)
}

func printStyleBlock(s *formatterState, block *ast.StyleBlock) {
	s.emitWithIndent("<style>")
	s.emitBlockStartTrivia(block)
	s.buf.WriteByte('\n')

	s.formatBlockStatements(block, block.Statements)

	s.emitWithIndent("</style>")
}

func printStyleRule(s *formatterState, block *ast.StyleRule) {
	s.writeIndent()
	for i, selector := range block.Selectors {
		if i < len(block.Selectors)-1 {
			s.buf.WriteString(", ")
		}
		s.buf.WriteString(selector)
	}
	s.buf.WriteString(" {")
	s.emitBlockStartTrivia(block)
	s.buf.WriteByte('\n')

	s.formatBlockStatements(block, block.Statements)

	s.emitWithIndent("}")
}

func printInlineMember(s *formatterState, im *ast.InlineMember) {
	s.emitWithIndent(s.targetFQN(im.Target))
	s.buf.WriteString(" : ")
	s.formatMember(im.Member)
}

func printEntity(s *formatterState, ent *ast.Entity) {
	// Otherwise, it is a named entity
	actualKindStr := ent.Kind.String()
	switch ent.Kind {
	case ast.EntityAbstractClass:
		actualKindStr = "abstract class"
	case ast.EntityEntityClass:
		actualKindStr = "entity"
	}
	s.emitWithIndent(actualKindStr)
	s.buf.WriteString(" ")
	if !ent.Alias.IsZero() {
		fmt.Fprintf(&s.buf, "%s as ", ent.Alias.Literal)
	}
	s.buf.WriteString(s.targetFQN(ent.Identifier))
	if ent.Generic != "" {
		s.buf.WriteString(" <")
		s.buf.WriteString(ent.Generic)
		s.buf.WriteString(">")
	}
	if ent.Stereotype != nil {
		str := ent.Stereotype.String()
		if str != "" {
			s.buf.WriteString(" <<")
			s.buf.WriteString(str)
			s.buf.WriteString(">>")
		}
	}
	for _, tag := range ent.Tags {
		s.buf.WriteString(" $")
		s.buf.WriteString(tag)
	}
	if ent.Color != "" {
		s.buf.WriteString(" ")
		s.buf.WriteString(ent.Color)
	}
	s.buf.WriteString(" {")
	s.emitBlockStartTrivia(ent)
	s.buf.WriteByte('\n')

	s.formatBlockMembers(ent, ent.Members)

	if ent.Kind != ast.EntityUnknown {
		s.emitWithIndent("}")
	}
}

func printContainer(s *formatterState, cont *ast.Container) {
	s.emitWithIndent(cont.Kind.String())
	if !cont.Alias.IsZero() {
		fmt.Fprintf(&s.buf, " %s as", cont.Alias.Literal)
	}
	if cont.Kind != ast.ContainerTogether {
		s.buf.WriteString(" ")
		s.buf.WriteString(s.targetFQN(cont.Identifier))
	}
	if cont.Stereotype != nil {
		str := cont.Stereotype.String()
		if str != "" {
			s.buf.WriteString(" <<")
			s.buf.WriteString(str)
			s.buf.WriteString(">>")
		}
	}
	// For 0 tags either way nothing is printed
	for _, tag := range cont.Tags {
		s.buf.WriteString(" $")
		s.buf.WriteString(tag)
	}
	if cont.Color != "" {
		s.buf.WriteString(" ")
		s.buf.WriteString(cont.Color)
	}
	s.buf.WriteString(" {")
	s.emitBlockStartTrivia(cont)
	s.buf.WriteByte('\n')

	s.formatBlockStatements(cont, cont.Statements)

	s.emitWithIndent("}")
}

func printRelationship(s *formatterState, rel *ast.Relationship) {
	s.emitWithIndent(s.targetFQN(rel.LHS))
	if rel.MultLHS.Raw != "" {
		fmt.Fprintf(&s.buf, " %q", rel.MultLHS.Raw)
	}
	s.buf.WriteString(" ")
	lbrCount := rel.BodyCount / 2
	rbrCount := rel.BodyCount - lbrCount

	switch rel.LArrow {
	case '<', '}', 'o', 'x', '*', '+', '^', '#':
		s.buf.WriteRune(rel.LArrow)
	case '|':
		s.buf.WriteString("<|")
	}
	s.buf.WriteString(strings.Repeat(string(rel.Body), lbrCount))
	if rel.Direction != ast.DirectionUnknown {
		s.buf.WriteString(arrowDirection(rel.Direction))
	}
	if len(rel.Attrs) > 0 {
		s.buf.WriteByte('[')
		for i, attr := range rel.Attrs {
			if i > 0 {
				s.buf.WriteByte(',')
			}
			s.buf.WriteString(attr)
		}
		s.buf.WriteByte(']')
	}
	s.buf.WriteString(strings.Repeat(string(rel.Body), rbrCount))
	switch rel.RArrow {
	case '>', '{', 'o', 'x', '*', '+', '^', '#':
		s.buf.WriteRune(rel.RArrow)
	case '|':
		s.buf.WriteString("|>")
	}

	if rel.MultRHS.Raw != "" {
		fmt.Fprintf(&s.buf, " %q", rel.MultRHS.Raw)
	}
	s.buf.WriteString(" ")
	s.buf.WriteString(s.targetFQN(rel.RHS))
	if rel.Label != "" {
		s.buf.WriteString(" : ")
		s.buf.WriteString(rel.Label)
	}
}

func printTextBlock(s *formatterState, block *ast.TextBlock) {
	var kind string
	switch block.Kind {
	case ast.BlockLegend:
		kind = "legend"
	case ast.BlockHeader:
		kind = "header"
	case ast.BlockFooter:
		kind = "footer"
	case ast.BlockTitle:
		kind = "title"
	default:
		panic("unreachable")
	}

	isMultiline := strings.Contains(block.Text, "\n")
	isBlock := isMultiline || len(block.Text) > 60

	alignments := strings.TrimSpace(block.VerticalAlignment + " " + block.HorizontalAlignment)

	if !isBlock {
		if alignments != "" {
			s.buf.WriteString(alignments)
			s.buf.WriteByte(' ')
		}
		s.buf.WriteString(kind)
		s.buf.WriteByte(' ')
	} else {
		s.buf.WriteString(kind)
		if alignments != "" {
			s.buf.WriteByte(' ')
			s.buf.WriteString(alignments)
		}
		s.buf.WriteByte('\n')
	}

	s.buf.WriteString(block.Text)

	if isBlock {
		fmt.Fprintf(&s.buf, "\nend %s", kind)
	}
}

func printGenericCommand(s *formatterState, st *ast.GenericCommand) {
	s.emitWithIndent(st.Name)
	for _, arg := range st.Args {
		s.buf.WriteByte(' ')
		s.buf.WriteString(arg)
	}
}

func printIncludeDirective(s *formatterState, st *ast.IncludeDirective) {
	switch st.Kind {
	case ast.IncludeOnce:
		s.emitWithIndent("!include ")
	case ast.IncludeMany:
		s.emitWithIndent("!include_many ")
	}
	s.buf.WriteString(st.Path)
	if st.Tag != "" {
		s.buf.WriteByte('!')
		s.buf.WriteString(st.Tag)
	}
}

func printVisibilityCommand(s *formatterState, st *ast.VisibilityCommand) {
	switch st.Kind {
	case ast.VisibilityCMDHide:
		s.emitWithIndent("hide ")
	case ast.VisibilityCMDShow:
		s.emitWithIndent("show ")
	case ast.VisibilityCMDRemove:
		s.emitWithIndent("remove ")
	case ast.VisibilityCMDRestore:
		s.emitWithIndent("restore ")
	}
	s.buf.WriteString(st.Target)
}

func printSetCommand(s *formatterState, st *ast.SetCommand) {
	if st.Key == "separator" {
		s.packageSeparator = st.Value
	}
	s.emitWithIndent("set ")
	s.buf.WriteString(st.Key)
	s.buf.WriteByte(' ')
	s.buf.WriteString(st.Value)
}

func printDirectionCommand(s *formatterState, st *ast.DirectionCommand) {
	switch st.Direction {
	case ast.LeftToRightDirection:
		s.emitWithIndent("left to right direction")
	case ast.TopToBottomDirection:
		s.emitWithIndent("top to bottom direction")
	}
}
