package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/islandora/scyllaridae/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyAction(t *testing.T) {
	tests := []struct {
		name     string
		payload  api.Payload
		expected eventAction
	}{
		{
			name: "delete event",
			payload: api.Payload{
				Type:    "Delete",
				Summary: "Content deleted",
			},
			expected: actionDelete,
		},
		{
			name: "external event",
			payload: api.Payload{
				Type:    "Update",
				Summary: "External content",
			},
			expected: actionExternalIndex,
		},
		{
			name: "media event",
			payload: api.Payload{
				Type:    "Update",
				Summary: "Media updated",
				Object: api.Object{
					URL: []api.Link{
						{Href: "http://example.com/media/1?_format=json", MediaType: "application/json"},
					},
				},
				Attachment: api.Attachment{
					Content: api.Content{SourceField: "field_media_file"},
				},
			},
			expected: actionMediaIndex,
		},
		{
			name: "node event",
			payload: api.Payload{
				Type:    "Update",
				Summary: "Node updated",
				Object: api.Object{
					URL: []api.Link{
						{Href: "http://example.com/node/1?_format=jsonld", MediaType: "application/ld+json"},
					},
				},
			},
			expected: actionNodeIndex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := classifyAction(tt.payload.Type, tt.payload.Summary, tt.payload)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestFcrepoHandler_DeleteEvent(t *testing.T) {
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer fedoraServer.Close()

	handler := &FcrepoHandler{
		Config: &FcrepoIndexerConfig{
			FedoraURL:             fedoraServer.URL,
			ModifiedDatePredicate: "http://schema.org/dateModified",
			IsFedora6:             true,
		},
		FedoraClient: NewFedoraClient(true),
	}

	payload := api.Payload{
		Type:    "Delete",
		Summary: "Content deleted",
		Target:  fedoraServer.URL,
		Object: api.Object{
			ID: "urn:uuid:9541c0c1-5bee-4973-a93a-69b3c1a1f906",
		},
	}

	status, body, contentType, err := handler.Handle(payload, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "deleted", string(body))
	assert.Equal(t, "text/plain", contentType)
}

func TestFcrepoHandler_NodeIndexEvent(t *testing.T) {
	// Mock Drupal returns JSON-LD (dynamically using server URL for @id)
	var drupalServer *httptest.Server
	drupalServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		drupalJsonld := map[string]interface{}{
			"@graph": []map[string]interface{}{
				{
					"@id":                            drupalServer.URL + "/node/1",
					"http://schema.org/name":         []map[string]string{{"@value": "Test"}},
					"http://schema.org/dateModified": []map[string]string{{"@value": "2024-01-15T10:30:00+00:00"}},
				},
			},
		}
		w.Header().Set("Content-Type", "application/ld+json")
		if err := json.NewEncoder(w).Encode(drupalJsonld); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer drupalServer.Close()

	// Mock Fedora: HEAD -> 404, PUT -> 201
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "HEAD":
			w.WriteHeader(http.StatusNotFound)
		case "PUT":
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer fedoraServer.Close()

	handler := &FcrepoHandler{
		Config: &FcrepoIndexerConfig{
			FedoraURL:             fedoraServer.URL,
			ModifiedDatePredicate: "http://schema.org/dateModified",
			StripFormatJsonld:     true,
			IsFedora6:             true,
		},
		FedoraClient: NewFedoraClient(true),
	}

	payload := api.Payload{
		Type:    "Update",
		Summary: "Node updated",
		Target:  fedoraServer.URL,
		Object: api.Object{
			ID: "urn:uuid:9541c0c1-5bee-4973-a93a-69b3c1a1f906",
			URL: []api.Link{
				{Href: drupalServer.URL + "/node/1?_format=jsonld", MediaType: "application/ld+json"},
			},
		},
	}

	status, body, _, err := handler.Handle(payload, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "indexed", string(body))
}

func TestFindURL(t *testing.T) {
	urls := []api.Link{
		{Href: "http://example.com/node/1?_format=jsonld", MediaType: "application/ld+json", Rel: "alternate"},
		{Href: "http://example.com/node/1", Rel: "canonical"},
		{Href: "http://example.com/node/1?_format=json", MediaType: "application/json"},
	}

	assert.Equal(t, "http://example.com/node/1?_format=jsonld", findURL(urls, "application/ld+json", ""))
	assert.Equal(t, "http://example.com/node/1", findURL(urls, "", "canonical"))
	assert.Equal(t, "http://example.com/node/1?_format=json", findURL(urls, "application/json", ""))
	assert.Equal(t, "", findURL(urls, "text/html", ""))
}
