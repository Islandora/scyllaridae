package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonldToNTriples(t *testing.T) {
	jsonld := `{
		"@id": "http://example.com/node/1",
		"http://schema.org/name": "Test Node"
	}`

	result, err := JsonldToNTriples([]byte(jsonld))
	require.NoError(t, err)
	assert.Contains(t, result, "<http://example.com/node/1>")
	assert.Contains(t, result, "<http://schema.org/name>")
	assert.Contains(t, result, "Test Node")
}

func TestJsonldToNTriples_WithGraph(t *testing.T) {
	jsonld := `{
		"@context": {"schema": "http://schema.org/"},
		"@id": "http://example.com/node/1",
		"schema:name": "Test"
	}`

	result, err := JsonldToNTriples([]byte(jsonld))
	require.NoError(t, err)

	// Each line should have exactly 3 components + dot (N-Triple format)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		assert.True(t, strings.HasSuffix(line, " ."), "line should end with ' .': %s", line)
	}
}

func TestNquadsToNTriples(t *testing.T) {
	nquads := `<http://example.com/s> <http://example.com/p> "value" <http://example.com/graph> .
<http://example.com/s> <http://example.com/p2> <http://example.com/o> .
`
	result := nquadsToNTriples(nquads)

	lines := strings.Split(strings.TrimSpace(result), "\n")
	assert.Len(t, lines, 2)

	// First line should have graph stripped
	assert.Equal(t, `<http://example.com/s> <http://example.com/p> "value" .`, lines[0])
	// Second line should be preserved as-is
	assert.Equal(t, `<http://example.com/s> <http://example.com/p2> <http://example.com/o> .`, lines[1])
}

func TestSplitNQuadLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		{
			name:     "simple triple",
			line:     `<http://example.com/s> <http://example.com/p> <http://example.com/o>`,
			expected: []string{"<http://example.com/s>", "<http://example.com/p>", "<http://example.com/o>"},
		},
		{
			name:     "literal with spaces",
			line:     `<http://example.com/s> <http://example.com/p> "hello world"`,
			expected: []string{"<http://example.com/s>", "<http://example.com/p>", `"hello world"`},
		},
		{
			name:     "quad with graph",
			line:     `<http://example.com/s> <http://example.com/p> "value" <http://example.com/g>`,
			expected: []string{"<http://example.com/s>", "<http://example.com/p>", `"value"`, "<http://example.com/g>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitNQuadLine(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}
