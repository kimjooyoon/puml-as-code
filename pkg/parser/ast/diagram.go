package ast

type Diagram struct {
	Name        string         `json:",omitempty"`
	Title       string         `json:",omitempty"`
	Statements  []Statement    `json:",omitempty"`
	Diagnostics DiagnosticList `json:"diagnostics,omitempty"`
}

func (d *Diagram) HasErrors() bool {
	return len(d.Diagnostics) > 0
}

type BoundOption struct {
	Key   string `json:",omitempty"`
	Value string `json:",omitempty"`
}

type BoundToolOptions struct {
	File    string        `json:",omitempty"`
	Caption string        `json:",omitempty"`
	Options []BoundOption `json:",omitempty"`
}

func (t BoundToolOptions) Get(key string) (string, bool) {
	for _, opt := range t.Options {
		if opt.Key == key {
			return opt.Value, true
		}
	}
	return "", false
}

type DiagramBound struct {
	IsStart      bool
	Type         string            `json:",omitempty"`
	ID           string            `json:",omitempty"`
	Params       []BoundOption     `json:",omitempty"`
	Tools        *BoundToolOptions `json:",omitempty"`
	TrailingName string            `json:",omitempty"`
	BaseNode
}

var _ Statement = (*DiagramBound)(nil)

func (d *DiagramBound) StatementNode() Statement {
	return d
}

func (d DiagramBound) DiagramName() string {
	if d.Tools != nil && d.Tools.File != "" {
		return d.Tools.File
	}
	return d.TrailingName
}

func (d DiagramBound) Name() string {
	return d.DiagramName()
}

func (d DiagramBound) GetParam(key string) (string, bool) {
	for _, p := range d.Params {
		if p.Key == key {
			return p.Value, true
		}
	}
	return "", false
}
