package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMessageHandler_MethodNotAllowed(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
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

func TestIntegration_PutDestination(t *testing.T) {
	var err error

	method := "PUT"
	content := "This is a test file content"

	destinationServer := createMockDestinationServer(t, method, content)
	defer destinationServer.Close()

	sourceServer := createMockSourceServer(t, content)
	defer sourceServer.Close()

	// Mock the environment variable for the configuration file path
	os.Setenv("SCYLLARIDAE_YML", `
destination-http-method: "PUT"
file-header: Apix-Ldp-Resource
arg-header: X-Islandora-Args
forward-auth: false
allowed-mimetypes: [
  "text/plain"
]
cmd-by-mimetype:
  default:
    cmd: "cat"
`)
	defer os.Unsetenv("SCYLLARIDAE_YML")

	config, err = ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	// Configure and start the main server
	setupServer := httptest.NewServer(http.HandlerFunc(MessageHandler))
	defer setupServer.Close()

	// Prepare a mock message to be sent to the main server
	testData := Data{
		Actor: Actor{
			Id: "actor1",
		},
		Object: Object{
			Id: "object1",
			URL: []URL{
				{
					Name:      "Source",
					Type:      "source",
					Href:      sourceServer.URL,
					MediaType: "text/plain",
					Rel:       "source",
				},
			},
			NewVersion: true,
		},
		Attachment: Attachment{
			Type: "file",
			Content: Content{
				MimeType:       "text/plain",
				Args:           "",
				SourceUri:      sourceServer.URL,
				DestinationUri: destinationServer.URL,
				FileUploadUri:  "",
			},
			MediaType: "text/plain",
		},
		Type:    "TestType",
		Summary: "This is a test",
	}

	jsonBytes, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Send the mock message to the main server
	req, err := http.NewRequest("POST", setupServer.URL, bytes.NewReader(jsonBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Capture the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
}

func TestIntegration_GetDestination(t *testing.T) {
	var err error
	method := "GET"
	content := ""

	sourceServer := createMockSourceServer(t, content)
	defer sourceServer.Close()

	destinationServer := createMockDestinationServer(t, method, content)
	defer destinationServer.Close()

	// Mock the environment variable for the configuration file path
	os.Setenv("SCYLLARIDAE_YML", fmt.Sprintf(`
destination-http-method: "%s"
file-header: Apix-Ldp-Resource
arg-header: X-Islandora-Args
forward-auth: false
allowed-mimetypes: [
  "text/plain"
]
cmd-by-mimetype:
  default:
    cmd: "cat"
`, method))
	defer os.Unsetenv("SCYLLARIDAE_YML")

	config, err = ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	// Configure and start the main server
	setupServer := httptest.NewServer(http.HandlerFunc(MessageHandler))
	defer setupServer.Close()

	// Prepare a mock message to be sent to the main server
	testData := Data{
		Actor: Actor{
			Id: "actor1",
		},
		Object: Object{
			Id: "object1",
			URL: []URL{
				{
					Name:      "Source",
					Type:      "source",
					Href:      sourceServer.URL,
					MediaType: "text/plain",
					Rel:       "source",
				},
			},
			NewVersion: true,
		},
		Attachment: Attachment{
			Type: "file",
			Content: Content{
				MimeType:       "text/plain",
				Args:           "",
				SourceUri:      sourceServer.URL,
				DestinationUri: destinationServer.URL,
				FileUploadUri:  "",
			},
			MediaType: "text/plain",
		},
		Type:    "TestType",
		Summary: "This is a test",
	}

	jsonBytes, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Send the mock message to the main server
	req, err := http.NewRequest("POST", setupServer.URL, bytes.NewReader(jsonBytes))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Capture the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
}

func createMockDestinationServer(t *testing.T, method, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("Expected %s method, got %s", method, r.Method)
		}
		if r.Method != "GET" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal("Failed to read request body")
			}
			defer r.Body.Close()

			if string(body) != content {
				t.Errorf("Unexpected body content: %s", string(body))
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func createMockSourceServer(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal("Failed to write response in mock server")
		}
	}))
}
