package ast

import "yur4uwe/pac/pkg/tokenizer"

type Node interface {
	Span() tokenizer.SourceSpan
	GetLeadingTrivia() []tokenizer.Token
	GetTrailingTrivia() []tokenizer.Token
}

type BaseNode struct {
	NodeSpan       tokenizer.SourceSpan
	LeadingTrivia  []tokenizer.Token `json:",omitempty"`
	TrailingTrivia []tokenizer.Token `json:",omitempty"`
}

func (bn BaseNode) GetLeadingTrivia() []tokenizer.Token  { return bn.LeadingTrivia }
func (bn BaseNode) GetTrailingTrivia() []tokenizer.Token { return bn.TrailingTrivia }
func (bn BaseNode) Span() tokenizer.SourceSpan           { return bn.NodeSpan }

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
func (t UnhandledStatement) StatementNode() Statement {
	return t
}

var _ Statement = UnhandledStatement{}

func updateBaseNode(bn *BaseNode, span tokenizer.SourceSpan, trailing []tokenizer.Token) {
	bn.NodeSpan = span
	if len(bn.TrailingTrivia) > 0 && len(trailing) > 0 {
		bn.TrailingTrivia = append(bn.TrailingTrivia, trailing...)
	} else if len(trailing) > 0 {
		bn.TrailingTrivia = trailing
	}
}

// WithMetadata attaches SourceSpan and TrailingTrivia to any Node (whether passed by value or pointer)
// and returns the updated Node.
func WithMetadata[T Node](node T, span tokenizer.SourceSpan, trailing []tokenizer.Token) T {
	var raw any = node
	switch n := raw.(type) {
	case *Entity:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case Entity:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *Container:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case Container:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *Relationship:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case Relationship:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *ClassSeparator:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case ClassSeparator:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *FieldDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case FieldDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *MethodDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case MethodDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *Note:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case Note:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *DiagramBound:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case DiagramBound:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *TextBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case TextBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *UnhandledStatement:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case UnhandledStatement:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *GenericCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case GenericCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *IncludeDirective:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case IncludeDirective:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *ScaleCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case ScaleCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *VisibilityCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case VisibilityCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *SetCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case SetCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *DirectionCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case DirectionCommand:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *StyleDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case StyleDeclaration:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *StyleRule:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case StyleRule:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *StyleBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case StyleBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *SkinparamSetting:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case SkinparamSetting:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	case *SkinparamBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return node
	case SkinparamBlock:
		updateBaseNode(&n.BaseNode, span, trailing)
		return any(n).(T)
	}
	return node
}
