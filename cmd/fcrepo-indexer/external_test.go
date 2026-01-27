package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveExternal(t *testing.T) {
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}))
	defer externalServer.Close()

	var capturedLinkHeader string
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedLinkHeader = r.Header.Get("Link")
		w.WriteHeader(http.StatusCreated)
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	status, err := saveExternal(client, "9541c0c1-5bee-4973-a93a-69b3c1a1f906",
		externalServer.URL+"/file.pdf", fedoraServer.URL, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, status)
	assert.Contains(t, capturedLinkHeader, "ExternalContent")
	assert.Contains(t, capturedLinkHeader, "application/pdf")
}

func TestHeadExternalURL_FallbackToPublic(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Header.Get("Authorization") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mime, err := headExternalURL(server.URL+"/image.jpg", "Bearer bad-token")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", mime)
	assert.Equal(t, 2, callCount)
}

func TestHeadExternalURL_StripParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mime, err := headExternalURL(server.URL+"/page.html", "")
	require.NoError(t, err)
	assert.Equal(t, "text/html", mime)
	assert.False(t, strings.Contains(mime, ";"))
}
