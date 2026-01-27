package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/islandora/scyllaridae/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlazegraphHandler_DeleteEvent(t *testing.T) {
	var capturedBody string
	triplestoreServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
		w.WriteHeader(http.StatusOK)
	}))
	defer triplestoreServer.Close()

	handler := &BlazegraphHandler{
		Config: &TriplestoreConfig{
			TriplestoreURL: triplestoreServer.URL,
		},
	}

	payload := api.Payload{
		Type:    "Delete",
		Summary: "Content deleted",
		Object: api.Object{
			URL: []api.Link{
				{Href: "http://example.com/node/1", Rel: "canonical"},
			},
		},
	}

	status, body, _, err := handler.Handle(payload, "")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "deleted", string(body))
	assert.Contains(t, capturedBody, "update=")
	assert.Contains(t, capturedBody, "DELETE")
}

func TestBlazegraphHandler_IndexEvent(t *testing.T) {
	// Mock Drupal server returning JSON-LD
	drupalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		if _, err := w.Write([]byte(`{
			"@id": "http://example.com/node/1",
			"http://schema.org/name": "Test Node"
		}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer drupalServer.Close()

	var capturedBody string
	triplestoreServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer triplestoreServer.Close()

	handler := &BlazegraphHandler{
		Config: &TriplestoreConfig{
			TriplestoreURL: triplestoreServer.URL,
		},
	}

	payload := api.Payload{
		Type:    "Update",
		Summary: "Node updated",
		Object: api.Object{
			URL: []api.Link{
				{Href: "http://example.com/node/1", Rel: "describes"},
				{Href: drupalServer.URL + "/node/1?_format=jsonld", MediaType: "application/ld+json"},
			},
		},
	}

	status, body, _, err := handler.Handle(payload, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "indexed", string(body))
	assert.Contains(t, capturedBody, "update=")
	// Should contain both DELETE and INSERT
	decoded := strings.ReplaceAll(capturedBody, "%20", " ")
	assert.Contains(t, decoded, "DELETE")
	assert.Contains(t, decoded, "INSERT")
}

func TestBlazegraphHandler_MissingSubjectURL(t *testing.T) {
	handler := &BlazegraphHandler{
		Config: &TriplestoreConfig{
			TriplestoreURL: "http://localhost:9999/sparql",
		},
	}

	payload := api.Payload{
		Type: "Update",
		Object: api.Object{
			URL: []api.Link{}, // No URLs
		},
	}

	status, _, _, err := handler.Handle(payload, "")
	assert.Error(t, err)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, err.Error(), "no subject URL")
}

func TestFindURLByRel(t *testing.T) {
	urls := []api.Link{
		{Href: "http://example.com/node/1?_format=jsonld", MediaType: "application/ld+json", Rel: "alternate"},
		{Href: "http://example.com/node/1", Rel: "canonical"},
		{Href: "http://example.com/files/test.pdf", Rel: "describes"},
	}

	assert.Equal(t, "http://example.com/files/test.pdf", findURLByRel(urls, "describes"))
	assert.Equal(t, "http://example.com/node/1", findURLByRel(urls, "canonical"))
	assert.Equal(t, "", findURLByRel(urls, "nonexistent"))
}

func TestFindURLByMediaType(t *testing.T) {
	urls := []api.Link{
		{Href: "http://example.com/node/1?_format=jsonld", MediaType: "application/ld+json"},
		{Href: "http://example.com/node/1?_format=json", MediaType: "application/json"},
	}

	assert.Equal(t, "http://example.com/node/1?_format=jsonld", findURLByMediaType(urls, "application/ld+json"))
	assert.Equal(t, "", findURLByMediaType(urls, "text/html"))
}
