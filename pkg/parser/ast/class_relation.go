package ast

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//go:generate enumer -type=RelationType -transform=lower -trimprefix=Relation -json
type RelationType int

const (
	RelationUnknown     RelationType = iota
	RelationAssociation              // <-
	RelationAggregation              // o-
	RelationComposition              // *-
	RelationInheritance              // <|-
	RelationDependency               // <..
	RelationRealization              // <|..
)

type Relationship struct {
	LHS       TargetRef
	RHS       TargetRef
	Direction DirectionKind `json:",omitempty"`

	TypeLHS RelationType `json:",omitempty"`
	TypeRHS RelationType `json:",omitempty"`
	MultLHS Cardinality
	MultRHS Cardinality

	// Arrow itself
	Body           rune // '-', '.'
	BodyCount      int
	LArrow, RArrow rune `json:",omitempty"`
	// Special case for left/righ arrow rune of relationship:
	// if the arrow is like '--|>', the '|' is used to distinguish it from '-->'
	// which would have end = '>'

	Label string   `json:",omitempty"`
	Attrs []string `json:",omitempty"`
	BaseNode
}

var _ Statement = Relationship{}

func (r Relationship) StatementNode() Statement {
	return r
}

type Cardinality struct {
	Raw string
	Min int
	Max int // -1 represents '*' (unbounded)
}

func (m Cardinality) MarshalJSON() ([]byte, error) {
	if m == UnknownCardinality || m.Raw == "" {
		return []byte("null"), nil
	}

	type Alias struct {
		Raw string `json:"Raw,omitempty"`
		Min string
		Max string
	}

	var maxVal string
	if m.Max == -1 {
		maxVal = "*"
	} else {
		maxVal = strconv.Itoa(m.Max)
	}

	return json.Marshal(Alias{
		Raw: m.Raw,
		Min: strconv.Itoa(m.Min),
		Max: maxVal,
	})
}

func (m *Cardinality) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*m = UnknownCardinality
		return nil
	}

	type Alias struct {
		Raw string
		Min string
		Max string
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	m.Raw = aux.Raw
	if aux.Max == "*" || aux.Max == "many" {
		m.Max = -1
	} else {
		n, err := strconv.Atoi(aux.Max)
		if err != nil {
			return err
		}
		m.Max = n
	}

	return nil
}

// UnknownCardinality is an empty/unknown multiplicity.
var UnknownCardinality = Cardinality{Raw: "", Min: -2, Max: -2}

func ParseCardinality(s string) (Cardinality, error) {
	s = strings.TrimSpace(strings.Trim(s, `"`))
	if s == "" {
		return UnknownCardinality, nil
	}
	if s == "*" {
		return Cardinality{Raw: s, Min: 0, Max: -1}, nil
	}
	if strings.Contains(s, "..") {
		parts := strings.SplitN(s, "..", 2)
		if len(parts) != 2 {
			return UnknownCardinality, fmt.Errorf("invalid cardinality: %q", s)
		}
		min, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return UnknownCardinality, fmt.Errorf("invalid min in cardinality %q: %w", s, err)
		}
		maxStr := strings.TrimSpace(parts[1])
		if maxStr == "*" {
			return Cardinality{Raw: s, Min: min, Max: -1}, nil
		}
		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return UnknownCardinality, fmt.Errorf("invalid max in cardinality %q: %w", s, err)
		}
		return Cardinality{Raw: s, Min: min, Max: max}, nil
	}
	// single number
	n, err := strconv.Atoi(s)
	if err != nil {
		return UnknownCardinality, fmt.Errorf("invalid cardinality: %q", s)
	}
	return Cardinality{Raw: s, Min: n, Max: n}, nil
}

func (m Cardinality) String() string {
	if m == UnknownCardinality {
		return ""
	}
	if m.Max == -1 {
		if m.Min == 0 {
			return "*"
		}
		return fmt.Sprintf("%d..*", m.Min)
	}
	if m.Min == m.Max {
		return strconv.Itoa(m.Min)
	}
	return fmt.Sprintf("%d..%d", m.Min, m.Max)
}
