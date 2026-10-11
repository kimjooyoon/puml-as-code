package ast

type GenericCommand struct {
	Name string   `json:",omitempty"`
	Args []string `json:",omitempty"`
	BaseNode
}

var _ Statement = (*GenericCommand)(nil)

type ScaleCommand struct {
	IsMax bool
	Lhs   string `json:",omitempty"` // "1", "2", "200"
	Sep   string `json:",omitempty"` // "", ".", "/", "*", "x"
	Rhs   string `json:",omitempty"` // "5", "3", "100"; empty when Sep is empty
	Unit  string `json:",omitempty"` // "", "width", "height"
	BaseNode
}

var _ Statement = (*ScaleCommand)(nil)

type SetCommand struct {
	Key   string `json:",omitempty"` // e.g. separator
	Value string `json:",omitempty"` // e.g. .
	BaseNode
}

var _ Statement = (*SetCommand)(nil)

type DirectionCommandKind int

const (
	UnknownDirection DirectionCommandKind = iota
	LeftToRightDirection
	TopToBottomDirection
)

type DirectionCommand struct {
	Direction DirectionCommandKind `json:",omitempty"`
	BaseNode
}

var _ Statement = (*DirectionCommand)(nil)

func (d *GenericCommand) StatementNode() Statement   { return d }
func (d *ScaleCommand) StatementNode() Statement     { return d }
func (d *SetCommand) StatementNode() Statement       { return d }
func (d *DirectionCommand) StatementNode() Statement { return d }
