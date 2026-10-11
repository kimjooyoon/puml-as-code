package ast

import "yur4uwe/pac/pkg/tokenizer"

type Node interface {
	Span() tokenizer.SourceSpan
	GetLeadingTrivia() []tokenizer.Token
	GetTrailingTrivia() []tokenizer.Token
	SetMetadata(span tokenizer.SourceSpan, trailing []tokenizer.Token)
}

type BaseNode struct {
	NodeSpan       tokenizer.SourceSpan
	LeadingTrivia  []tokenizer.Token `json:",omitempty"`
	TrailingTrivia []tokenizer.Token `json:",omitempty"`
}

func (bn *BaseNode) GetLeadingTrivia() []tokenizer.Token  { return bn.LeadingTrivia }
func (bn *BaseNode) GetTrailingTrivia() []tokenizer.Token { return bn.TrailingTrivia }
func (bn *BaseNode) Span() tokenizer.SourceSpan           { return bn.NodeSpan }

func (bn *BaseNode) SetMetadata(span tokenizer.SourceSpan, trailing []tokenizer.Token) {
	bn.NodeSpan = span
	if len(bn.TrailingTrivia) > 0 && len(trailing) > 0 {
		bn.TrailingTrivia = append(bn.TrailingTrivia, trailing...)
	} else if len(trailing) > 0 {
		bn.TrailingTrivia = trailing
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

// WithMetadata attaches SourceSpan and TrailingTrivia to any Node and returns the updated Node.
func WithMetadata[T Node](node T, span tokenizer.SourceSpan, trailing []tokenizer.Token) T {
	node.SetMetadata(span, trailing)
	return node
}
