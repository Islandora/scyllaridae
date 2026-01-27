package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessJsonld(t *testing.T) {
	jsonld := `{
		"@graph": [
			{"@id": "http://example.com/node/1", "http://schema.org/name": [{"@value": "Test"}]},
			{"@id": "http://example.com/other/2", "http://schema.org/name": [{"@value": "Other"}]}
		]
	}`

	result, err := ProcessJsonld([]byte(jsonld), "http://example.com/node/1", "http://fedora:8080/rest/95/41/c0/c1/uuid")
	require.NoError(t, err)

	var resources []map[string]json.RawMessage
	err = json.Unmarshal(result, &resources)
	require.NoError(t, err)
	assert.Len(t, resources, 1)

	var id string
	err = json.Unmarshal(resources[0]["@id"], &id)
	require.NoError(t, err)
	assert.Equal(t, "http://fedora:8080/rest/95/41/c0/c1/uuid", id)
}

func TestProcessJsonld_SchemeAgnostic(t *testing.T) {
	jsonld := `{
		"@graph": [
			{"@id": "https://example.com/node/1", "http://schema.org/name": [{"@value": "Test"}]}
		]
	}`

	result, err := ProcessJsonld([]byte(jsonld), "http://example.com/node/1", "http://fedora:8080/rest/uuid")
	require.NoError(t, err)

	var resources []map[string]json.RawMessage
	err = json.Unmarshal(result, &resources)
	require.NoError(t, err)
	assert.Len(t, resources, 1)
}

func TestProcessJsonld_NoMatch(t *testing.T) {
	jsonld := `{
		"@graph": [
			{"@id": "http://other.com/node/1", "http://schema.org/name": [{"@value": "Test"}]}
		]
	}`

	_, err := ProcessJsonld([]byte(jsonld), "http://example.com/node/1", "http://fedora:8080/rest/uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no @graph entry matched")
}

func TestGetModifiedTimestamp(t *testing.T) {
	jsonld := `[{
		"@id": "http://example.com/node/1",
		"http://schema.org/dateModified": [{"@value": "2024-01-15T10:30:00+00:00"}]
	}]`

	ts, err := GetModifiedTimestamp([]byte(jsonld), "http://schema.org/dateModified")
	require.NoError(t, err)
	assert.Equal(t, int64(1705314600), ts)
}

func TestGetModifiedTimestamp_Missing(t *testing.T) {
	jsonld := `[{"@id": "http://example.com/node/1"}]`
	_, err := GetModifiedTimestamp([]byte(jsonld), "http://schema.org/dateModified")
	assert.Error(t, err)
}
