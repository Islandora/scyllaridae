package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/stretchr/testify/assert"
)

type Test struct {
	name           string
	authHeader     string
	requestAuth    string
	expectedStatus int
	expectedBody   string
	returnedBody   string
	expectMismatch bool
	yml            string
	mimetype       string
}

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

func TestIntegration(t *testing.T) {
	tests := []Test{
		{
			name:           "cURL pass no auth",
			authHeader:     "foo",
			requestAuth:    "bar",
			expectedStatus: http.StatusOK,
			returnedBody:   "foo",
			expectedBody:   "foo",
			expectMismatch: false,
			mimetype:       "text/plain",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`,
		},
		{
			name:           "cURL fail with bad auth",
			authHeader:     "foo",
			requestAuth:    "bar",
			expectedStatus: http.StatusBadRequest,
			returnedBody:   "foo",
			expectedBody:   "foo",
			expectMismatch: false,
			mimetype:       "text/plain",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`,
		},
		{
			name:           "cURL pass with auth",
			authHeader:     "pass",
			requestAuth:    "pass",
			expectedStatus: http.StatusOK,
			returnedBody:   "foo",
			expectedBody:   "foo",
			expectMismatch: false,
			mimetype:       "text/plain",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`,
		},
		{
			name:           "cURL fail no auth bad output",
			requestAuth:    "pass",
			expectedStatus: http.StatusOK,
			returnedBody:   "foo",
			expectedBody:   "bar",
			expectMismatch: true,
			mimetype:       "text/plain",
			yml: `
forwardAuth: false
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`,
		},
		{
			name:           "cURL fail bad output",
			authHeader:     "pass",
			requestAuth:    "pass",
			expectedStatus: http.StatusOK,
			returnedBody:   "foo",
			expectedBody:   "bar",
			expectMismatch: true,
			mimetype:       "text/plain",
			yml: `
forwardAuth: true
allowedMimeTypes:
  - "text/plain"
cmdByMimeType:
  default:
    cmd: curl
    args:
      - "%s"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			destinationServer := createMockDestinationServer(tt.returnedBody)
			defer destinationServer.Close()

			sourceServer := createMockSourceServer(t, tt.mimetype, tt.authHeader, destinationServer.URL)
			defer sourceServer.Close()

			os.Setenv("SCYLLARIDAE_YML", tt.yml)
			// set the config based on tt.yml
			config, err = scyllaridae.ReadConfig("")
			if err != nil {
				t.Fatalf("Could not read YML: %v", err)
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
			// set the mimetype to send to the destination server in the Accept header
			req.Header.Set("Accept", "application/xml")
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
				assert.Equal(t, os.Getenv("NEEDFUL"), tt.expectedBody)
			}
		})
	}
}

func createMockDestinationServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		os.Setenv("NEEDFUL", content)
	}))
}

func createMockSourceServer(t *testing.T, mimetype, auth, content string) *httptest.Server {
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
