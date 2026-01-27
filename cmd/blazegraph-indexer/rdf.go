package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/piprate/json-gold/ld"
)

// JsonldToNTriples converts JSON-LD content to N-Triples format.
// It uses the json-gold library for JSON-LD processing.
func JsonldToNTriples(jsonldBytes []byte) (string, error) {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("")
	options.Format = "application/n-quads"

	// Parse the JSON-LD input
	doc, err := ld.DocumentFromReader(strings.NewReader(string(jsonldBytes)))
	if err != nil {
		return "", fmt.Errorf("error parsing JSON-LD: %w", err)
	}

	// Convert to RDF (N-Quads format)
	triples, err := proc.ToRDF(doc, options)
	if err != nil {
		return "", fmt.Errorf("error converting JSON-LD to RDF: %w", err)
	}

	// When format is set, ToRDF returns a string of N-Quads
	var nquads string
	switch v := triples.(type) {
	case string:
		nquads = v
	case *ld.RDFDataset:
		// Serialize using NQuadRDFSerializer
		serializer := ld.NQuadRDFSerializer{}
		var buf bytes.Buffer
		if err := serializer.SerializeTo(&buf, v); err != nil {
			return "", fmt.Errorf("error serializing RDF dataset: %w", err)
		}
		nquads = buf.String()
	default:
		return "", fmt.Errorf("unexpected RDF result type: %T", triples)
	}

	// N-Quads include graph component; for N-Triples we strip the graph
	// component if present (4th element before the dot in each line).
	return nquadsToNTriples(nquads), nil
}

// nquadsToNTriples strips the graph component from N-Quads to produce N-Triples.
// N-Quads format: <s> <p> <o> <g> .
// N-Triples format: <s> <p> <o> .
func nquadsToNTriples(nquads string) string {
	lines := strings.Split(nquads, "\n")
	var result strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// If the line has a graph component (4 parts before dot), remove it
		// A simple heuristic: count elements before the final "."
		if !strings.HasSuffix(line, ".") {
			result.WriteString(line)
			result.WriteString("\n")
			continue
		}

		// Remove trailing " ."
		content := strings.TrimSuffix(line, " .")
		content = strings.TrimSuffix(content, ".")
		content = strings.TrimSpace(content)

		// Split to find components. N-Quads may have 3 or 4 components.
		// We need to be careful about quoted strings with spaces.
		parts := splitNQuadLine(content)
		if len(parts) >= 4 {
			// Has graph component, strip it (keep first 3 parts)
			result.WriteString(strings.Join(parts[:3], " "))
			result.WriteString(" .\n")
		} else if len(parts) == 3 {
			// Already N-Triple format
			result.WriteString(strings.Join(parts, " "))
			result.WriteString(" .\n")
		} else {
			// Preserve as-is if unexpected format
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	return result.String()
}

// splitNQuadLine splits an N-Quad line into its components,
// respecting quoted strings and IRIs.
func splitNQuadLine(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	inIRI := false
	prevChar := rune(0)

	for _, ch := range line {
		switch {
		case ch == '"' && prevChar != '\\' && !inIRI:
			inQuote = !inQuote
			current.WriteRune(ch)
		case ch == '<' && !inQuote:
			inIRI = true
			current.WriteRune(ch)
		case ch == '>' && !inQuote:
			inIRI = false
			current.WriteRune(ch)
		case ch == ' ' && !inQuote && !inIRI:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
		prevChar = ch
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
