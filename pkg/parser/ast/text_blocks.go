package ast

//go:generate enumer -type=TextBlockKind -transform=lower -json -trimprefix=Block
type TextBlockKind int

const (
	BlockUnknown TextBlockKind = iota
	BlockLegend
	BlockHeader
	BlockFooter
	BlockTitle
)

// TextBlock implements [Statement].
// if you want to know whether it is multiline, check for \n
type TextBlock struct {
	Kind TextBlockKind
	Text string

	VerticalAlignment   string
	HorizontalAlignment string

	BaseNode
}

// StatementNode implements [Statement].
func (t *TextBlock) StatementNode() Statement {
	return t
}

var _ Statement = (*TextBlock)(nil)

//go:generate enumer -type=NoteKind -transform=lower -trimprefix=Note -json
type NoteKind int

const (
	NoteUnknown NoteKind = iota

	// NoteInlineAlias represents a note defined with a text string and alias.
	//
	// Syntax (single-line):
	//   note "Text" as <alias> [#color]
	//
	// Example:
	//   note "Active connection" as N1
	//
	// Syntax (multiline):
	//   note as <alias> [#color]
	//     <text>
	//   end note
	//
	// Example:
	//   note as N2
	//     This is a floating note
	//   end note
	//
	NoteAlias

	// NoteTargeted represents a note positioned relative to an entity (or previous statement).
	//
	// Syntax (single-line):
	//   note <left|right|top|bottom> [of <target>] [#color] : <text>
	//
	// Syntax (multiline):
	//   note <left|right|top|bottom> [of <target>] [#color]
	//     <text>
	//   end note
	//
	// Example:
	//   note left of User : Authenticated via OAuth
	NoteTargeted

	// NoteLink represents a note attached to the preceding or active relationship link.
	//
	// Syntax (single-line):
	//   note on link [#color] : <text>
	//
	// Syntax (multiline):
	//   note on link [#color]
	//     <text>
	//   end note
	//
	// Example:
	//   note on link : TLS Encrypted
	NoteLink
)

type Note struct {
	Kind       NoteKind      `json:",omitempty"`
	Text       string        `json:",omitempty"`
	Direction  DirectionKind `json:",omitempty"`
	Target     *TargetRef    `json:",omitempty"`
	Color      string        `json:",omitempty"`
	Identifier TargetRef
	BaseNode
}

var _ Statement = (*Note)(nil)

func (n *Note) StatementNode() Statement { return n }
