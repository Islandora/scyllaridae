package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeleteWhere_NoNamedGraph(t *testing.T) {
	result := DeleteWhere("http://example.com/node/1", "")
	assert.Equal(t, "DELETE WHERE { <http://example.com/node/1> ?p ?o }", result)
}

func TestDeleteWhere_WithNamedGraph(t *testing.T) {
	result := DeleteWhere("http://example.com/node/1", "http://example.com/graph")
	assert.Equal(t, "DELETE WHERE { GRAPH <http://example.com/graph> { <http://example.com/node/1> ?p ?o } }", result)
}

func TestInsertData_NoNamedGraph(t *testing.T) {
	ntriples := `<http://example.com/node/1> <http://schema.org/name> "Test" .` + "\n"
	result := InsertData(ntriples, "")
	expected := `INSERT DATA { <http://example.com/node/1> <http://schema.org/name> "Test" .` + "\n}"
	assert.Equal(t, expected, result)
}

func TestInsertData_WithNamedGraph(t *testing.T) {
	ntriples := `<http://example.com/node/1> <http://schema.org/name> "Test" .` + "\n"
	result := InsertData(ntriples, "http://example.com/graph")
	assert.Contains(t, result, "GRAPH <http://example.com/graph>")
	assert.Contains(t, result, ntriples)
}

func TestBuildUpdateQuery(t *testing.T) {
	ntriples := `<http://example.com/node/1> <http://schema.org/name> "Test" .` + "\n"
	result := BuildUpdateQuery("http://example.com/node/1", ntriples, "")
	assert.Contains(t, result, "DELETE WHERE")
	assert.Contains(t, result, "INSERT DATA")
	assert.Contains(t, result, ntriples)
}

func TestEncodeUpdateBody(t *testing.T) {
	query := `DELETE WHERE { <http://example.com/node/1> ?p ?o }`
	result := EncodeUpdateBody(query)
	assert.Contains(t, result, "update=")
	// URL-encoded content should not contain raw spaces or angle brackets
	assert.NotContains(t, result[7:], " ")
}
