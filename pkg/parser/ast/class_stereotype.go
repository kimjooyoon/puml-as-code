package ast

import "yur4uwe/pac/pkg/tokenizer"

type Stereotype struct {
	// Tokens contains all tokens between "<<" and ">>" that form this single
	// stereotype.
	Tokens []tokenizer.Token

	// FullSpan spans the entire "<< ... >>" construct.
	FullSpan tokenizer.SourceSpan
}
