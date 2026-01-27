package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMediaURLs(t *testing.T) {
	fileUUID := "abcdef01-2345-6789-abcd-ef0123456789"

	// Mock Drupal server: returns JSON with link headers and source field
	drupalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<http://drupal.test/node/1?_format=jsonld>; rel="alternate"; type="application/ld+json"`)
		w.Header().Add("Link", `<http://drupal.test/sites/default/files/test.pdf>; rel="describes"`)
		mediaJSON := map[string]interface{}{
			"field_media_file": []map[string]interface{}{
				{"target_uuid": fileUUID},
			},
		}
		body, _ := json.Marshal(mediaJSON)
		if _, err := w.Write(body); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer drupalServer.Close()

	// Mock Fedora server: HEAD returns describedby link
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<http://fedora.test/rest/ab/cd/ef/01/`+fileUUID+`/fcr:metadata>; rel="describedby"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	urls, err := getMediaURLs(client, "field_media_file", drupalServer.URL+"/media/1?_format=json", fedoraServer.URL, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, "http://drupal.test/node/1?_format=jsonld", urls.Jsonld)
	assert.Equal(t, "http://drupal.test/sites/default/files/test.pdf", urls.Drupal)
	assert.Contains(t, urls.Fedora, "fcr:metadata")
}

func TestGetMediaURLs_MissingAlternateLink(t *testing.T) {
	drupalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No link headers
		if _, err := w.Write([]byte(`{}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer drupalServer.Close()

	client := NewFedoraClient(true)
	_, err := getMediaURLs(client, "field_media_file", drupalServer.URL+"/media/1", "http://fedora.test", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "alternate")
}
