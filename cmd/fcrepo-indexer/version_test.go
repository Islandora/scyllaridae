package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateVersion(t *testing.T) {
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/fcr:versions"))
		assert.NotEmpty(t, r.Header.Get("Memento-Datetime"))
		w.WriteHeader(http.StatusCreated)
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	status, err := createVersion(client, "9541c0c1-5bee-4973-a93a-69b3c1a1f906", fedoraServer.URL, "Bearer test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, status)
}

func TestCreateVersion_Failure(t *testing.T) {
	fedoraServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		if _, err := w.Write([]byte("Conflict")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer fedoraServer.Close()

	client := NewFedoraClient(true)
	status, err := createVersion(client, "9541c0c1-5bee-4973-a93a-69b3c1a1f906", fedoraServer.URL, "")
	assert.Error(t, err)
	assert.Equal(t, http.StatusConflict, status)
}
