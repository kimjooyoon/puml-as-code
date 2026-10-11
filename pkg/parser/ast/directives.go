package ast

//go:generate enumer -type=IncludeKind -transform=lower -trimprefix=Include -json
type IncludeKind int

const (
	IncludeOnce IncludeKind = iota
	IncludeMany
)

type IncludeDirective struct {
	Path string      `json:",omitempty"`
	Tag  string      `json:",omitempty"`
	Kind IncludeKind `json:",omitempty"`
	BaseNode
}

var _ Statement = (*IncludeDirective)(nil)

func (d *IncludeDirective) StatementNode() Statement { return d }

