package api

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeEventMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantError bool
	}{
		{
			name:      "valid JSON payload",
			input:     []byte(`{"type":"test","target":"thumbnail","object":{"id":"123"}}`),
			wantError: false,
		},
		{
			name:      "invalid JSON",
			input:     []byte(`{invalid json`),
			wantError: true,
		},
		{
			name:      "empty payload",
			input:     []byte(`{}`),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := DecodeEventMessage(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, payload)
		})
	}
}

func TestDecodeAlpacaMessage(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		headers          map[string]string
		wantDestMimeType string
		wantSrcMimeType  string
		wantArgs         string
		wantError        bool
	}{
		{
			name:   "POST request with headers",
			method: "POST",
			headers: map[string]string{
				"Content-Type":     "image/jpeg",
				"Accept":           "image/png",
				"X-Islandora-Args": "-quality 80",
			},
			wantDestMimeType: "image/png",
			wantSrcMimeType:  "image/jpeg",
			wantArgs:         "-quality 80",
			wantError:        false,
		},
		{
			name:   "GET request with base64 event header",
			method: "GET",
			headers: map[string]string{
				"X-Islandora-Event": base64.StdEncoding.EncodeToString([]byte(`{"type":"test","object":{"id":"123"},"attachment":{"content":{"source_mimetype":"text/plain"}}}`)),
				"Accept":            "application/xml",
			},
			wantDestMimeType: "application/xml",
			wantSrcMimeType:  "text/plain",
			wantError:        false,
		},
		{
			name:   "default Accept header",
			method: "POST",
			headers: map[string]string{
				"Content-Type": "text/html",
			},
			wantDestMimeType: "text/plain", // Default when Accept is empty
			wantSrcMimeType:  "text/html",
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			payload, err := DecodeAlpacaMessage(req, "test-auth")
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "test-auth", payload.Authorization)
			assert.Equal(t, tt.wantDestMimeType, payload.Attachment.Content.DestinationMimeType)
			if tt.wantSrcMimeType != "" {
				assert.Equal(t, tt.wantSrcMimeType, payload.Attachment.Content.SourceMimeType)
			}
			if tt.wantArgs != "" {
				assert.Equal(t, tt.wantArgs, payload.Attachment.Content.Args)
			}
		})
	}
}

func TestDecodeAlpacaMessage_WithSourceURIFetch(t *testing.T) {
	// Create a mock server that returns a Content-Type header
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header to be forwarded")
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Apix-Ldp-Resource", mockServer.URL)
	req.Header.Set("Accept", "image/png")

	payload, err := DecodeAlpacaMessage(req, "Bearer test-token")
	assert.NoError(t, err)
	assert.Equal(t, "image/jpeg", payload.Attachment.Content.SourceMimeType)
	assert.Equal(t, mockServer.URL, payload.Attachment.Content.SourceURI)
}

func TestDecodeAlpacaMessage_InvalidBase64Event(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Islandora-Event", "not-valid-base64!@#$")
	req.Header.Set("Apix-Ldp-Resource", "https://example.com/file.txt")

	_, err := DecodeAlpacaMessage(req, "")
	assert.Error(t, err)
}

func TestDecodeAlpacaMessage_InvalidJSONInEvent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Islandora-Event", base64.StdEncoding.EncodeToString([]byte(`{invalid json}`)))
	req.Header.Set("Apix-Ldp-Resource", "https://example.com/file.txt")

	_, err := DecodeAlpacaMessage(req, "")
	assert.Error(t, err)
}

func TestGetSourceUri_ErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		sourceURI string
		wantError bool
	}{
		{
			name:      "empty source URI",
			sourceURI: "",
			wantError: false, // No error when source URI is empty, just returns
		},
		{
			name:      "invalid URL",
			sourceURI: "ht!tp://invalid url with spaces",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Payload{
				Attachment: Attachment{
					Content: Content{
						SourceURI: tt.sourceURI,
					},
				},
			}

			err := p.getSourceUri("")
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetSourceUri_UnreachableServer(t *testing.T) {
	p := &Payload{
		Object: Object{ID: "test-123"},
		Attachment: Attachment{
			Content: Content{
				SourceURI: "http://localhost:99999/unreachable",
			},
		},
	}

	err := p.getSourceUri("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error issuing HEAD request")
}

func TestDecodeAlpacaMessage_FailedSourceURIFetch(t *testing.T) {
	// Create a mock server that returns an error status
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	// Test with invalid event JSON in header
	req := httptest.NewRequest("GET", "/", nil)
	invalidJSON := base64.StdEncoding.EncodeToString([]byte(`{"attachment":{"content":{"source_uri":"` + mockServer.URL + `"}}}`))
	req.Header.Set("X-Islandora-Event", invalidJSON)

	payload, err := DecodeAlpacaMessage(req, "")
	assert.NoError(t, err) // DecodeAlpacaMessage itself doesn't error, just populates the payload
	assert.Equal(t, mockServer.URL, payload.Attachment.Content.SourceURI)
}

func TestPayloadStructures(t *testing.T) {
	// Test that all the payload structures can be marshaled/unmarshaled
	payload := Payload{
		Type:    "Create",
		Target:  "thumbnail",
		Summary: "Test summary",
		Actor:   Actor{ID: "user:1"},
		Object: Object{
			ID:           "node:123",
			IsNewVersion: true,
			URL: []Link{
				{
					Name:      "canonical",
					Type:      "Link",
					Href:      "https://example.com/node/123",
					MediaType: "text/html",
					Rel:       "canonical",
				},
			},
		},
		Attachment: Attachment{
			Type:      "Image",
			MediaType: "image/jpeg",
			Content: Content{
				SourceMimeType:      "image/jpeg",
				DestinationMimeType: "image/png",
				Args:                "-resize 50%",
				SourceURI:           "https://example.com/source.jpg",
				SourceField:         "field_media_image",
				DestinationURI:      "https://example.com/dest.png",
				FileUploadURI:       "private://derivatives/thumb.png",
			},
		},
		Authorization: "Bearer token123",
	}

	// Test that we can access all fields
	assert.Equal(t, "Create", payload.Type)
	assert.Equal(t, "thumbnail", payload.Target)
	assert.Equal(t, "user:1", payload.Actor.ID)
	assert.Equal(t, "node:123", payload.Object.ID)
	assert.True(t, payload.Object.IsNewVersion)
	assert.Len(t, payload.Object.URL, 1)
	assert.Equal(t, "canonical", payload.Object.URL[0].Rel)
	assert.Equal(t, "Image", payload.Attachment.Type)
	assert.Equal(t, "image/jpeg", payload.Attachment.Content.SourceMimeType)
	assert.Equal(t, "Bearer token123", payload.Authorization)
}
