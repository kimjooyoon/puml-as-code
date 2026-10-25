package ast

import (
	"strings"

	"yur4uwe/pac/pkg/tokenizer"
)

type Stereotype struct {
	Value Ident

	SpotColor string `json:",omitempty"`
	// SpotSpan spans the spot syntax in "<< (S,#00FF00) ... >>" construct.
	SpotSpan tokenizer.SourceSpan
	SpotRune rune `json:",omitempty"`

	// FullSpan spans the entire "<< ... >>" construct.
	FullSpan tokenizer.SourceSpan
}

// String returns the formatted representation of the stereotype inside << >>,
// e.g. "(S, #FF7700) MyStereotype" or "Service".
func (s *Stereotype) String() string {
	if s == nil {
		return ""
	}
	if s.SpotRune == 0 {
		return s.Value.Literal
	}
	var sb strings.Builder
	sb.WriteByte('(')
	sb.WriteRune(s.SpotRune)
	if s.SpotColor != "" {
		sb.WriteString(", ")
		sb.WriteString(s.SpotColor)
	}
	sb.WriteByte(')')
	if s.Value.Literal != "" {
		sb.WriteByte(' ')
		sb.WriteString(s.Value.Literal)
	}
	return sb.String()
}

// Name returns the identifier literal for the stereotype.
func (s *Stereotype) Name() string {
	if s == nil {
		return ""
	}
	return s.Value.Literal
}
