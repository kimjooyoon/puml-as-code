package ast

//go:generate enumer -type=VisibilityKind -transform=lower -trimprefix=Visibility -json
type VisibilityKind int

const (
	VisibilityUnknown VisibilityKind = iota
	VisibilityPublic
	VisibilityPrivate
	VisibilityProtected
	VisibilityPackage
)

//go:generate enumer -type=DirectionKind -transform=lower -trimprefix=Direction -json
type DirectionKind int

const (
	DirectionUnknown DirectionKind = iota
	DirectionLeft
	DirectionRight
	DirectionTop
	DirectionBottom
)
