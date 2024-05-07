package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	stomp "github.com/go-stomp/stomp/v3"
	ss "github.com/go-stomp/stomp/v3/server"
	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestRecvStompMessages(t *testing.T) {
	addr := "127.0.0.1:61613"
	os.Setenv("STOMP_SERVER_ADDR", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Could not start stomp mock server: %v", err)
	}
	defer func() { l.Close() }()
	go func() {
		err := ss.Serve(l)
		if err != nil {
			slog.Error("Error starting mock stomp server", "err", err)
			os.Exit(1)
		}
	}()

	os.Setenv("SCYLLARIDAE_YML", `
queueName: "test-queue"
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: touch
    args:
      - "%target"
`)
	config, err = scyllaridae.ReadConfig("")
	if err != nil {
		t.Fatalf("Could not read YML: %v", err)
		os.Exit(1)
	}

	subscribed := make(chan bool, 1)
	go RecvStompMessages("test-queue", subscribed)
	<-subscribed

	conn, err := stomp.Dial("tcp", addr,
		stomp.ConnOpt.AcceptVersion(stomp.V11),
		stomp.ConnOpt.AcceptVersion(stomp.V12),
		stomp.ConnOpt.Host("dragon"),
		stomp.ConnOpt.Header("nonce", "B256B26D320A"))

	if err != nil {
		t.Fatalf("Could not connection to stomp mock server: %v", err)
	}

	err = conn.Send(
		"test-queue",
		"text/plain",
		[]byte(`{
	"@context": "https://www.w3.org/ns/activitystreams",
	"actor": {
		"type": "Person",
		"id": "urn:uuid:b3f0a1ba-fd0c-4977-a123-3faf470374f2",
		"url": [
			{
				"name": "Canonical",
				"type": "Link",
				"href": "https://islandora.dev/user/1",
				"mediaType": "text/html",
				"rel": "canonical"
			}
		]
	},
	"object": {
		"id": "urn:uuid:abcdef01-2345-6789-abcd-ef0123456789",
		"url": [
			{
				"name": "Canonical",
				"type": "Link",
				"href": "https://islandora.dev/node/1",
				"mediaType": "text/html",
				"rel": "canonical"
			},
			{
				"name": "JSON",
				"type": "Link",
				"href": "https://islandora.dev/node/1?_format=json",
				"mediaType": "application/json",
				"rel": "alternate"
			},
			{
				"name": "JSONLD",
				"type": "Link",
				"href": "https://islandora.dev/node/1?_format=jsonld",
				"mediaType": "application/ld+json",
				"rel": "alternate"
			}
		],
		"isNewVersion": true
	},
	"target": "/tmp/stomp.success",
	"type": "Update",
	"summary": "Update a Node"
}`))
	if err != nil {
		t.Fatalf("Could not send test stomp message: %v", err)
	}

	// give the command some time to finish
	time.Sleep(time.Second * 5)

	// make sure the command ran
	f := "/tmp/stomp.success"
	assert.FileExists(t, f)
}

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

func TestMimeTypes(t *testing.T) {
	mimeTypes := map[string]string{
		"application/msword": "doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
		"application/vnd.ms-excel": "xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
		"application/vnd.ms-powerpoint":                                             "ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",

		"image/jpeg":    "jpg",
		"image/jp2":     "jp2",
		"image/png":     "png",
		"image/gif":     "gif",
		"image/bmp":     "bmp",
		"image/svg+xml": "svg",
		"image/tiff":    "tiff",
		"image/webp":    "webp",

		"audio/mpeg":        "mp3",
		"audio/x-wav":       "wav",
		"audio/ogg":         "ogg",
		"audio/aac":         "m4a",
		"audio/webm":        "webm",
		"audio/flac":        "flac",
		"audio/midi":        "mid",
		"audio/x-m4a":       "m4a",
		"audio/x-realaudio": "ra",

		"video/mp4":                     "mp4",
		"video/x-msvideo":               "avi",
		"video/x-ms-wmv":                "wmv",
		"video/mpeg":                    "mpg",
		"video/webm":                    "webm",
		"video/quicktime":               "mov",
		"application/vnd.apple.mpegurl": "m3u8",
		"video/3gpp":                    "3gp",
		"video/mp2t":                    "ts",
		"video/x-flv":                   "flv",
		"video/x-m4v":                   "m4v",
		"video/x-mng":                   "mng",
		"video/x-ms-asf":                "asx",
		"video/ogg":                     "ogg",

		"text/plain":      "txt",
		"text/html":       "html",
		"application/pdf": "pdf",
		"text/csv":        "csv",
	}
	test := Test{
		authHeader:          "pass",
		requestAuth:         "pass",
		expectedStatus:      http.StatusOK,
		expectedBody:        "%s txt\n",
		returnedBody:        "",
		expectMismatch:      false,
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
	}
	for mimeType, extension := range mimeTypes {
		test.name = fmt.Sprintf("test %s to %s conversion", mimeType, extension)
		test.mimetype = mimeType
		test.expectedBody = fmt.Sprintf("%s txt\n", extension)
		t.Run(test.name, func(t *testing.T) {
			var err error
			destinationServer := createMockDestinationServer(t, test.returnedBody)
			defer destinationServer.Close()

			sourceServer := createMockSourceServer(t, test.mimetype, test.authHeader, destinationServer.URL)
			defer sourceServer.Close()

			os.Setenv("SCYLLARIDAE_YML", test.yml)
			// set the config based on test.yml
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
			req.Header.Set("Accept", test.destinationMimeType)
			req.Header.Set("Authorization", test.requestAuth)
			req.Header.Set("Apix-Ldp-Resource", sourceServer.URL)

			// Capture the response
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			assert.Equal(t, test.expectedStatus, resp.StatusCode)
			if !test.expectMismatch {
				// if we're setesting up the destination server as the cURL target
				// make sure it returned
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("Unable to read source uri resp body: %v", err)
				}
				assert.Equal(t, test.expectedBody, string(body))
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
