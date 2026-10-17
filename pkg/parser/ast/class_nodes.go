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
	Identifier TargetRef       `json:",omitempty"`
	Alias      tokenizer.Token `json:",omitempty"`
	Kind       EntityKind      `json:",omitempty"`
	Stereotype string          `json:",omitempty"`
	Tags       []string        `json:",omitempty"`
	Generic    string          `json:",omitempty"`
	Color      string          `json:",omitempty"`
	Members    []Member        `json:",omitempty"`
}

var _ Statement = (*Entity)(nil)

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
	Identifier TargetRef       `json:",omitempty"`
	Alias      tokenizer.Token `json:",omitempty"`
	Kind       ContainerKind   `json:",omitempty"`
	Stereotype string          `json:",omitempty"`
	Tags       []string        `json:",omitempty"`
	Color      string          `json:",omitempty"`
	Statements []Statement     `json:",omitempty"`
	BaseNode
}

var _ Statement = (*Container)(nil)

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
