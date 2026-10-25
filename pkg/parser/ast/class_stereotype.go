package ast

import "yur4uwe/pac/pkg/tokenizer"

type Stereotype struct {
	// Tokens contains all tokens between "<<" and ">>" that form this single
	// stereotype.
	Value Ident

	SpotColor string
	// SpotSpan spans the spot syntax in "<< (S,#00FF00) ... >>" construct.
	SpotSpan tokenizer.SourceSpan
	SpotRune rune

	// FullSpan spans the entire "<< ... >>" construct.
	FullSpan tokenizer.SourceSpan
}
