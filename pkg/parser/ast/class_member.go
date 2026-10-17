package ast

import (
	"fmt"

	"yur4uwe/pac/pkg/tokenizer"
)

type Member interface {
	Node
	MemberNode() Member
}

type BadMember struct {
	Diagnostic *Diagnostic       `json:"diagnostic,omitempty"`
	Tokens     []tokenizer.Token `json:"tokens,omitempty"`
	BaseNode
}

func (b *BadMember) MemberNode() Member { return b }

var _ Member = (*BadMember)(nil)

type ClassSeparator struct {
	// Optional label text
	Label string `json:",omitempty"`
	// Separator type. One of "-", "=", ".", "_"
	Type      rune
	TypeCount int
	BaseNode
}

var _ Member = (*ClassSeparator)(nil)

func (cs *ClassSeparator) MemberNode() Member {
	return cs
}

type Field interface {
	fmt.Stringer
	FieldName() string
}

type Method interface {
	fmt.Stringer
	MethodName() string
}

type FieldDeclaration struct {
	BaseNode
	Visibility VisibilityKind
	Modifiers  []string `json:",omitempty"`
	Field      Field
}

var _ Member = (*FieldDeclaration)(nil)

func (fd *FieldDeclaration) MemberNode() Member { return fd }
func (fd *FieldDeclaration) FieldName() string  { return fd.Field.FieldName() }

type MethodDeclaration struct {
	BaseNode
	Visibility VisibilityKind
	Modifiers  []string `json:",omitempty"`
	Method     Method
}

var _ Member = (*MethodDeclaration)(nil)

func (md *MethodDeclaration) MemberNode() Member { return md }
func (md *MethodDeclaration) MethodName() string { return md.Method.MethodName() }

type InlineMember struct {
	Target TargetRef
	Member Member
	BaseNode
}

var _ Statement = (*InlineMember)(nil)

func (im *InlineMember) StatementNode() Statement { return im }
