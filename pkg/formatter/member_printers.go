package formatter

import (
	"fmt"
	"strings"

	"yur4uwe/pac/pkg/parser/ast"
)

func printClassSeparator(s *formatterState, sep *ast.ClassSeparator) {
	s.writeIndent()
	if sep.Label == "" {
		s.buf.WriteString(strings.Repeat(string(sep.Type), sep.TypeCount))
		return
	}

	leftCount := sep.TypeCount / 2
	rightCount := sep.TypeCount - leftCount
	s.buf.WriteString(strings.Repeat(string(sep.Type), leftCount))
	fmt.Fprintf(&s.buf, " %s ", sep.Label)
	s.buf.WriteString(strings.Repeat(string(sep.Type), rightCount))
}

func printField(s *formatterState, field *ast.FieldDeclaration) {
	s.writeIndent()
	for _, modifier := range field.Modifiers {
		fmt.Fprintf(&s.buf, "{%s} ", modifier)
	}
	s.buf.WriteString(visibilitySymbol(field.Visibility))
	s.buf.WriteString(field.Field.String())
}

func printMethod(s *formatterState, method *ast.MethodDeclaration) {
	s.writeIndent()
	for _, modifier := range method.Modifiers {
		fmt.Fprintf(&s.buf, "{%s} ", modifier)
	}
	s.buf.WriteString(visibilitySymbol(method.Visibility))
	s.buf.WriteString(method.Method.String())
}
