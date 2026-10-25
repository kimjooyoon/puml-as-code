package ast

import (
	"strings"

	"yur4uwe/pac/pkg/tokenizer"
)

type Node interface {
	Span() tokenizer.SourceSpan
	GetLeadingTrivia() []Comment
	GetTrailingTrivia() []Comment
	SetMetadata(span tokenizer.SourceSpan, leading, trailing []Comment)
}

type BaseNode struct {
	NodeSpan       tokenizer.SourceSpan
	LeadingTrivia  []Comment `json:",omitempty"`
	TrailingTrivia []Comment `json:",omitempty"`
}

func (bn *BaseNode) GetLeadingTrivia() []Comment  { return bn.LeadingTrivia }
func (bn *BaseNode) GetTrailingTrivia() []Comment { return bn.TrailingTrivia }
func (bn *BaseNode) Span() tokenizer.SourceSpan   { return bn.NodeSpan }

func (bn *BaseNode) SetMetadata(span tokenizer.SourceSpan, leading, trailing []Comment) {
	bn.NodeSpan = span
	if len(leading) > 0 {
		if len(bn.LeadingTrivia) > 0 {
			bn.LeadingTrivia = append(bn.LeadingTrivia, leading...)
		} else {
			bn.LeadingTrivia = leading
		}
	}
	if len(trailing) > 0 {
		if len(bn.TrailingTrivia) > 0 {
			bn.TrailingTrivia = append(bn.TrailingTrivia, trailing...)
		} else {
			bn.TrailingTrivia = trailing
		}
	}
}

type Block interface {
	Node
	HasBody() bool
	IsClosed() bool
	BlockStart() tokenizer.Pos
	BlockEnd() tokenizer.Pos
}

type Statement interface {
	Node
	StatementNode() Statement
}

type Ident struct {
	Literal string
	Span    tokenizer.SourceSpan
}

func (i Ident) String() string { return i.Literal }
func (i Ident) IsZero() bool {
	return i.Literal == "" &&
		i.Span.Start.Offset == 0 &&
		i.Span.End.Offset == 0
}

func IdentFromToken(tok tokenizer.Token) Ident {
	return Ident{
		Literal: tok.Literal,
		Span:    tok.Span,
	}
}

func IdentFromTokens(tokens ...tokenizer.Token) Ident {
	if len(tokens) == 0 {
		return Ident{}
	}
	if len(tokens) == 1 {
		return Ident{
			Literal: tokens[0].Literal,
			Span:    tokens[0].Span,
		}
	}
	var sb strings.Builder
	for _, tok := range tokens {
		sb.WriteString(tok.Literal)
	}
	return Ident{
		Literal: sb.String(),
		Span: tokenizer.SourceSpan{
			Start: tokens[0].Span.Start,
			End:   tokens[len(tokens)-1].Span.End,
		},
	}
}

type Comment struct {
	Text string
	Span tokenizer.SourceSpan
}

func (c Comment) IsZero() bool {
	return c.Text == "" &&
		c.Span.Start.Offset == 0 &&
		c.Span.End.Offset == 0
}

func (c Comment) String() string {
	return c.Text
}

// Content returns the comment text stripped of PlantUML comment delimiters.
func (c Comment) Content() string {
	text := c.Text
	if strings.HasPrefix(text, "/'") && strings.HasSuffix(text, "'/") {
		text = text[2 : len(text)-2]
	} else {
		text = strings.TrimPrefix(text, "'")
	}
	return strings.TrimSpace(text)
}

type MemberRef struct {
	Name      Ident
	Signature []tokenizer.Token `json:",omitempty"` // "(" ... ")" inclusive, nil if absent
}

func (m MemberRef) Span() tokenizer.SourceSpan {
	if len(m.Signature) > 0 {
		return tokenizer.SourceSpan{
			Start: m.Name.Span.Start,
			End:   m.Signature[len(m.Signature)-1].Span.End,
		}
	}
	return m.Name.Span
}

type TargetRef struct {
	PackagePath []Ident `json:",omitempty"`
	Name        Ident
	Member      *MemberRef `json:",omitempty"`
}

func (t TargetRef) Span() tokenizer.SourceSpan {
	start := t.Name.Span.Start
	if len(t.PackagePath) > 0 {
		start = t.PackagePath[0].Span.Start
	}
	end := t.Name.Span.End
	if t.Member != nil {
		end = t.Member.Span().End
	}
	return tokenizer.SourceSpan{Start: start, End: end}
}

func (t TargetRef) EntityName() string {
	return t.Name.Literal
}

func (t TargetRef) MemberName() string {
	if t.Member != nil {
		return t.Member.Name.Literal
	}
	return ""
}

func (t TargetRef) PackagePathStrings() []string {
	if len(t.PackagePath) == 0 {
		return nil
	}
	res := make([]string, len(t.PackagePath))
	for i, tok := range t.PackagePath {
		res[i] = tok.Literal
	}
	return res
}

type UnhandledStatement struct {
	Raw string
	BaseNode
}

// StatementNode implements [Statement].
func (t *UnhandledStatement) StatementNode() Statement {
	return t
}

var _ Statement = (*UnhandledStatement)(nil)

type BadStatement struct {
	Diagnostic *Diagnostic       `json:",omitempty"`
	Tokens     []tokenizer.Token `json:",omitempty"`
	BaseNode
}

func (b *BadStatement) StatementNode() Statement { return b }

var _ Statement = (*BadStatement)(nil)
