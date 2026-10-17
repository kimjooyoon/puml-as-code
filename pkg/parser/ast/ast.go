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

type TargetRef struct {
	PackagePath []string `json:",omitempty"`
	Entity      string   `json:",omitempty"`
	Member      string   `json:",omitempty"`
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
