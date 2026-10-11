package ast

import "fmt"

type Member interface {
	Node
	MemberNode() Member
}

type ClassSeparator struct {
	// Optional label text
	Label string `json:",omitempty"`
	// Separator type. One of "-", "=", ".", "_"
	Type      rune
	TypeCount int
	BaseNode
}

var _ Member = ClassSeparator{}

func (cs ClassSeparator) MemberNode() Member {
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
	Modifiers  []string
	Field      Field
}

var _ Member = FieldDeclaration{}

func (fd FieldDeclaration) MemberNode() Member { return fd }
func (fd FieldDeclaration) FieldName() string  { return fd.Field.FieldName() }

type MethodDeclaration struct {
	BaseNode
	Visibility VisibilityKind
	Modifiers  []string
	Method     Method
}

var _ Member = MethodDeclaration{}

func (md MethodDeclaration) MemberNode() Member { return md }
func (md MethodDeclaration) MethodName() string { return md.Method.MethodName() }
