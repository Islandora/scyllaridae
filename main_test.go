package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/stretchr/testify/assert"
)

type Test struct {
	name                string
	authHeader          string
	requestAuth         string
	expectedStatus      int
	expectedBody        string
	returnedBody        string
	expectMismatch      bool
	yml                 string
	mimetype            string
	destinationMimeType string
}

func TestMessageHandler_MethodNotAllowed(t *testing.T) {
	testConfig := &scyllaridae.ServerConfig{}
	server := &Server{Config: testConfig}

	req, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.MessageHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Handler returned wrong status code: got %v want %v",
			status, http.StatusMethodNotAllowed)
	}
}

func TestIntegration(t *testing.T) {
	tests := []Test{
		{
			name:                "cURL pass no auth",
			authHeader:          "foo",
			requestAuth:         "bar",
			expectedStatus:      http.StatusOK,
			returnedBody:        "foo",
			expectedBody:        "foo",
			expectMismatch:      false,
			mimetype:            "text/plain",
			destinationMimeType: "application/xml",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%args"
`,
		},
		{
			name:                "cURL fail with bad auth",
			authHeader:          "foo",
			requestAuth:         "bar",
			expectedStatus:      http.StatusBadRequest,
			returnedBody:        "foo",
			expectedBody:        "Bad request\n",
			expectMismatch:      false,
			mimetype:            "text/plain",
			destinationMimeType: "application/xml",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%args"
`,
		},
		{
			name:                "cURL pass with auth",
			authHeader:          "pass",
			requestAuth:         "pass",
			expectedStatus:      http.StatusOK,
			returnedBody:        "foo",
			expectedBody:        "foo",
			expectMismatch:      false,
			mimetype:            "text/plain",
			destinationMimeType: "application/xml",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%args"
`,
		},
		{
			name:                "cURL fail no auth bad output",
			requestAuth:         "pass",
			expectedStatus:      http.StatusOK,
			returnedBody:        "foo",
			expectedBody:        "bar",
			expectMismatch:      true,
			mimetype:            "text/plain",
			destinationMimeType: "application/xml",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%args"
`,
		},
		{
			name:                "cURL fail bad output",
			authHeader:          "pass",
			requestAuth:         "pass",
			expectedStatus:      http.StatusOK,
			returnedBody:        "foo",
			expectedBody:        "bar",
			expectMismatch:      true,
			mimetype:            "text/plain",
			destinationMimeType: "application/xml",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%args"
`,
		},
		{
			name:                "test mimetype to ext conversion",
			authHeader:          "pass",
			requestAuth:         "pass",
			expectedStatus:      http.StatusOK,
			expectedBody:        "ppt txt\n",
			expectMismatch:      false,
			mimetype:            "application/vnd.ms-powerpoint",
			destinationMimeType: "text/plain",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "application/vnd.ms-powerpoint"
cmdByMimeType:
  default:
    cmd: echo
    args:
      - "%source-mime-ext"
      - "%destination-mime-ext"
`,
		},
		{
			name:                "test bad mimetype succeeds if not getting extension",
			authHeader:          "pass",
			requestAuth:         "pass",
			expectedStatus:      http.StatusOK,
			expectedBody:        "OK\n",
			expectMismatch:      false,
			mimetype:            "application/this-mime-type-doesn't-exist",
			destinationMimeType: "text/plain",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: echo
    args:
      - "OK"
`,
		},
		{
			name:                "test bad mimetype",
			authHeader:          "pass",
			requestAuth:         "pass",
			expectedStatus:      http.StatusBadRequest,
			expectedBody:        "Bad request\n",
			expectMismatch:      false,
			mimetype:            "application/this-mime-type-doesn't-exist",
			destinationMimeType: "text/plain",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: echo
    args:
      - "%source-mime-ext"
      - "%destination-mime-ext"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			destinationServer := createMockDestinationServer(t, tt.returnedBody)
			defer destinationServer.Close()

			os.Setenv("SCYLLARIDAE_YML", tt.yml)
			config, err := scyllaridae.ReadConfig("")

			sourceServer := createMockSourceServer(t, config, tt.mimetype, tt.authHeader, destinationServer.URL)
			defer sourceServer.Close()
			if err != nil {
				t.Fatalf("Could not read YML: %v", err)
			}

			// Create a Server instance with the test config
			server := &Server{Config: config}

			// Configure and start the main server
			setupServer := httptest.NewServer(http.HandlerFunc(server.MessageHandler))
			defer setupServer.Close()

			// Send the mock message to the main server
			req, err := http.NewRequest("GET", setupServer.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Islandora-Args", destinationServer.URL)
			// set the mimetype to send to the destination server in the Accept header
			req.Header.Set("Accept", tt.destinationMimeType)
			req.Header.Set("Authorization", tt.requestAuth)
			req.Header.Set("Apix-Ldp-Resource", sourceServer.URL)

			// Capture the response
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			if !tt.expectMismatch {
				// if we're setting up the destination server as the cURL target
				// make sure it returned
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Unable to read source uri resp body: %v", err)
				}
				assert.Equal(t, tt.expectedBody, string(body))
			}
		})
	}

}

func createMockDestinationServer(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal("Failed to write response in mock server")
		}
	}))
}

func createMockSourceServer(t *testing.T, config *scyllaridae.ServerConfig, mimetype, auth, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.ForwardAuth && r.Header.Get("Authorization") != auth {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", mimetype)
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal("Failed to write response in mock server")
		}
	}))
}
