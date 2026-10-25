package ast

import (
	"strings"

	"yur4uwe/pac/pkg/tokenizer"
)

func MergeTokensIntoIdent(tokens ...tokenizer.Token) Ident {
	switch len(tokens) {
	case 0:
		return Ident{}
	case 1:
		return Ident{
			Literal: tokens[0].Literal,
			Span:    tokens[0].Span,
		}
	}

	var sb strings.Builder

	start := tokens[0].Span.Start
	end := tokens[len(tokens)-1].Span.End

	if end.Offset > start.Offset {
		sb.Grow(int(end.Offset - start.Offset))
	}

	sb.WriteString(tokens[0].Literal)
	prevEnd := tokens[0].Span.End

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.Span.Start.Offset > prevEnd.Offset {
			gap := int(tok.Span.Start.Offset - prevEnd.Offset)
			for range gap {
				sb.WriteByte(' ')
			}
		}
		sb.WriteString(tok.Literal)
		prevEnd = tok.Span.End
	}

	return Ident{
		Literal: sb.String(),
		Span: tokenizer.SourceSpan{
			Start: start,
			End:   end,
		},
	}
}
