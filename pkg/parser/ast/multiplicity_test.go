package ast

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMultiplicity(t *testing.T) {
	cases := []struct {
		in        string
		want      Cardinality
		shouldErr bool
	}{
		{`"*"`, Cardinality{Raw: "*", Min: 0, Max: -1}, false},
		{`*`, Cardinality{Raw: "*", Min: 0, Max: -1}, false},
		{`"1"`, Cardinality{Raw: "1", Min: 1, Max: 1}, false},
		{`1`, Cardinality{Raw: "1", Min: 1, Max: 1}, false},
		{`"0..*"`, Cardinality{Raw: "0..*", Min: 0, Max: -1}, false},
		{`0..*`, Cardinality{Raw: "0..*", Min: 0, Max: -1}, false},
		{`"1..4"`, Cardinality{Raw: "1..4", Min: 1, Max: 4}, false},
		{"", UnknownCardinality, false},
		{"abc", UnknownCardinality, true},
	}

	for _, c := range cases {
		m, err := ParseCardinality(c.in)
		if c.shouldErr {
			require.Error(t, err, "input=%q", c.in)
			continue
		}
		require.NoError(t, err, "input=%q", c.in)
		require.Equal(t, c.want.Raw, m.Raw, "raw mismatch for %q", c.in)
		require.Equal(t, c.want.Min, m.Min, "min mismatch for %q", c.in)
		require.Equal(t, c.want.Max, m.Max, "max mismatch for %q", c.in)
	}
}

func TestMultiplicityString(t *testing.T) {
	m, _ := ParseCardinality(`"*"`)
	require.Equal(t, "*", m.String())

	m, _ = ParseCardinality(`"1"`)
	require.Equal(t, "1", m.String())

	m, _ = ParseCardinality(`"1..*"`)
	require.Equal(t, "1..*", m.String())

	m, _ = ParseCardinality(`"1..4"`)
	require.Equal(t, "1..4", m.String())

	require.Equal(t, "", UnknownCardinality.String())
}
