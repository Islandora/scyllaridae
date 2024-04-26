package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
)

func TestMessageHandler_MethodNotAllowed(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(MessageHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}

func TestIntegration_cURL(t *testing.T) {
	var err error
	content := "OK"

	destinationServer := createMockDestinationServer(t, content)
	defer destinationServer.Close()

	// read the URL to fetch from the source server
	// this is passed to the cURL command
	sourceServer := createMockSourceServer(t, destinationServer.URL)
	defer sourceServer.Close()

	// Mock the environment variable for the configuration file path
	os.Setenv("SCYLLARIDAE_YML", `
fileHeader: Apix-Ldp-Resource
argHeader: X-Islandora-Args
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`)
	defer os.Unsetenv("SCYLLARIDAE_YML")

	config, err = scyllaridae.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}

	// Configure and start the main server
	setupServer := httptest.NewServer(http.HandlerFunc(MessageHandler))
	defer setupServer.Close()

	// Send the mock message to the main server
	req, err := http.NewRequest("GET", setupServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Islandora-Args", destinationServer.URL)
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Apix-Ldp-Resource", sourceServer.URL)

	// Capture the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != content {
		t.Errorf("Expected response body %s, got %s", content, string(body))
	}
}

func TestIntegration_cat(t *testing.T) {
	var err error
	sourceContent := "foo"

	// read the URL to fetch from the source server
	// this is passed to the cURL command
	sourceServer := createMockSourceServer(t, sourceContent)
	defer sourceServer.Close()

	// Mock the environment variable for the configuration file path
	os.Setenv("SCYLLARIDAE_YML", `
fileHeader: Apix-Ldp-Resource
argHeader: X-Islandora-Args
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: cat
`)
	defer os.Unsetenv("SCYLLARIDAE_YML")

	config, err = scyllaridae.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}

	// Configure and start the main server
	setupServer := httptest.NewServer(http.HandlerFunc(MessageHandler))
	defer setupServer.Close()

	// Send the mock message to the main server
	req, err := http.NewRequest("GET", setupServer.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Islandora-Args", "")
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Apix-Ldp-Resource", sourceServer.URL)

	// Capture the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != sourceContent {
		t.Errorf("Expected response body %s, got %s", sourceContent, string(body))
	}
}

func createMockDestinationServer(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal("Failed to write response in mock server")
		}
	}))
}

func createMockSourceServer(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal("Failed to write response in mock server")
		}
	}))
}
