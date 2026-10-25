package formatter

import (
	"yur4uwe/pac/pkg/parser/ast"
)

func (s *formatterState) emitTrivia(trivia ast.Comment) {
	if len(s.source) == 0 || trivia.Span.End.Offset == 0 {
		s.buf.WriteString(trivia.Text)
		return
	}
	s.emitRawSpan(trivia.Span)
}

func (s *formatterState) emitBlockStartTrivia(block ast.Node) {
	for _, trivia := range block.GetTrailingTrivia() {
		if trivia.Span.Start.Line == block.Span().Start.Line {
			s.buf.WriteByte(' ')
			s.emitTrivia(trivia)
		}
	}
}

func (s *formatterState) emitInnerBlockTrivia(block ast.Node) {
	for _, trivia := range block.GetTrailingTrivia() {
		if trivia.Span.Start.Line > block.Span().Start.Line && trivia.Span.Start.Line < block.Span().End.Line {
			s.writeIndent()
			s.emitTrivia(trivia)
			s.buf.WriteByte('\n')
		}
	}
}

func (s *formatterState) emitBlockEndTrivia(block ast.Node) {
	isBlock := block.Span().Start.Line != block.Span().End.Line
	for _, trivia := range block.GetTrailingTrivia() {
		sameLine := trivia.Span.Start.Line == block.Span().Start.Line
		if isBlock {
			if trivia.Span.Start.Line >= block.Span().End.Line {
				s.buf.WriteByte(' ')
				s.emitTrivia(trivia)
			}
		} else if sameLine {
			s.buf.WriteByte(' ')
			s.emitTrivia(trivia)
		}
	}
}
