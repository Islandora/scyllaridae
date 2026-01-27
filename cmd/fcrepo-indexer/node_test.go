package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveNode_CreateNew(t *testing.T) {
	// Mock Drupal server returning JSON-LD
	// Use a handler that dynamically builds JSON-LD with correct @id
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

	// Mock Fedora server: HEAD returns 404, PUT returns 201
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

	cfg := &FcrepoIndexerConfig{
		FedoraURL:             fedoraServer.URL,
		ModifiedDatePredicate: "http://schema.org/dateModified",
		StripFormatJsonld:     true,
		IsFedora6:             true,
	}
	client := NewFedoraClient(cfg.IsFedora6)

	status, err := saveNode(client, cfg, "9541c0c1-5bee-4973-a93a-69b3c1a1f906",
		drupalServer.URL+"/node/1?_format=jsonld", fedoraServer.URL, "Bearer test-token")
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, status)
}

func TestDeleteNode_Success(t *testing.T) {
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	status, err := deleteNode(client, "9541c0c1-5bee-4973-a93a-69b3c1a1f906", fedoraServer.URL, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, status)
}

func TestDeleteNode_AlreadyGone(t *testing.T) {
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	status, err := deleteNode(client, "9541c0c1-5bee-4973-a93a-69b3c1a1f906", fedoraServer.URL, "")
	require.NoError(t, err)
	assert.Equal(t, http.StatusGone, status)
}
