package ast

import "yur4uwe/pac/pkg/tokenizer"

//go:generate enumer -type=EntityKind -transform=lower -trimprefix=Entity -json
type EntityKind int

const (
	EntityUnknown EntityKind = iota
	EntityClass
	EntityAbstractClass
	EntityInterface
	EntityEnum
	EntityEntityClass
	EntityStruct
	EntityAnnotation
	EntityProtocol
	EntityCircle
	EntityDiamond
	EntityException
	EntityMetaclass
	EntityRecord
	EntityDataclass
	EntityStereotype // For the standalone "stereotype" keyword

	// Robustness (BCE)
	EntityBceEntity
	EntityBoundary
	EntityControl

	// Component / Mixed
	EntityActor
	EntityComponent
	EntityArtifact
)

func (k EntityKind) AllowsBody() bool {
	switch k {
	case EntityCircle, EntityDiamond:
		return false
	default:
		return true
	}
}

type Entity struct {
	BaseNode
	Identifier TargetRef
	Alias      Ident
	Kind       EntityKind    `json:",omitempty"`
	Stereotype *Stereotype   `json:",omitempty"`
	Tags       []string      `json:",omitempty"`
	Generic    string        `json:",omitempty"`
	Color      string        `json:",omitempty"`
	LBrace     tokenizer.Pos // Zero if bodyless declaration
	Members    []Member      `json:",omitempty"`
	RBrace     tokenizer.Pos // Zero if unclosed before EOF
}

var (
	_ Statement = (*Entity)(nil)
	_ Block     = (*Entity)(nil)
)

func (e *Entity) BlockStart() tokenizer.Pos { return e.LBrace }
func (e *Entity) BlockEnd() tokenizer.Pos   { return e.RBrace }

func (e *Entity) HasBody() bool { return e.LBrace != tokenizer.Pos{} }
func (e *Entity) IsClosed() bool {
	return !e.HasBody() || e.RBrace != tokenizer.Pos{}
}

func (e *Entity) StatementNode() Statement {
	return e
}

//go:generate enumer -type=ContainerKind -transform=lower -trimprefix=Container -json
type ContainerKind int

const (
	ContainerUnknown ContainerKind = iota
	ContainerPackage
	ContainerTogether
	ContainerNamespace
	ContainerFolder
	ContainerFrame
	ContainerRectangle
	ContainerDatabase
	ContainerCloud
	ContainerNode
)

type Container struct {
	Identifier TargetRef
	Alias      Ident
	Kind       ContainerKind `json:",omitempty"`
	Stereotype *Stereotype   `json:",omitempty"`
	Tags       []string      `json:",omitempty"`
	Color      string        `json:",omitempty"`
	LBrace     tokenizer.Pos // Zero if bodyless declaration
	Statements []Statement   `json:",omitempty"`
	RBrace     tokenizer.Pos // Zero if unclosed before EOF
	BaseNode
}

func (c *Container) BlockStart() tokenizer.Pos { return c.LBrace }
func (c *Container) BlockEnd() tokenizer.Pos   { return c.RBrace }

func (c *Container) HasBody() bool { return c.LBrace != tokenizer.Pos{} }
func (c *Container) IsClosed() bool {
	return !c.HasBody() || c.RBrace != tokenizer.Pos{}
}

var (
	_ Statement = (*Container)(nil)
	_ Block     = (*Container)(nil)
)

func (c *Container) StatementNode() Statement {
	return c
}

//go:generate enumer -type=VisibilityCommandKind -transform=lower -trimprefix=VisibilityCMD -json
type VisibilityCommandKind int

const (
	VisibilityCMDUnknown VisibilityCommandKind = iota
	VisibilityCMDHide
	VisibilityCMDShow
	VisibilityCMDRemove
	VisibilityCMDRestore
)

type VisibilityCommand struct {
	Kind   VisibilityCommandKind `json:",omitempty"` // Hide, Show, Remove, Restore
	Target string                `json:",omitempty"` // "empty members", "class Name", "circle", etc.
	BaseNode
}

var _ Statement = (*VisibilityCommand)(nil)

func (d *VisibilityCommand) StatementNode() Statement { return d }
