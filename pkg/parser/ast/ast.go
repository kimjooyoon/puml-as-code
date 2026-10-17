package ast

import "yur4uwe/pac/pkg/tokenizer"

type Node interface {
	Span() tokenizer.SourceSpan
	GetLeadingTrivia() []tokenizer.Token
	GetTrailingTrivia() []tokenizer.Token
	SetMetadata(span tokenizer.SourceSpan, leading, trailing []tokenizer.Token)
}

type BaseNode struct {
	NodeSpan       tokenizer.SourceSpan
	LeadingTrivia  []tokenizer.Token `json:",omitempty"`
	TrailingTrivia []tokenizer.Token `json:",omitempty"`
}

func (bn *BaseNode) GetLeadingTrivia() []tokenizer.Token  { return bn.LeadingTrivia }
func (bn *BaseNode) GetTrailingTrivia() []tokenizer.Token { return bn.TrailingTrivia }
func (bn *BaseNode) Span() tokenizer.SourceSpan           { return bn.NodeSpan }

func (bn *BaseNode) SetMetadata(span tokenizer.SourceSpan, leading, trailing []tokenizer.Token) {
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

type Statement interface {
	Node
	StatementNode() Statement
}

type MemberRef struct {
	Name      tokenizer.Token
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
	PackagePath []tokenizer.Token `json:",omitempty"`
	Name        tokenizer.Token
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
	Diagnostic *Diagnostic       `json:"diagnostic,omitempty"`
	Tokens     []tokenizer.Token `json:"tokens,omitempty"`
	BaseNode
}

func (b *BadStatement) StatementNode() Statement { return b }

var _ Statement = (*BadStatement)(nil)
