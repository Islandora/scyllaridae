package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
)

type JWK struct {
	Kty      string `json:"kty"`
	Kid      string `json:"kid"`
	Use      string `json:"use"`
	Alg      string `json:"alg"`
	N        string `json:"n"`
	E        string `json:"e"`
	Audience string `json:"aud"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
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
	os.Setenv("SKIP_JWT_VERIFY", "true")
	testConfig := &scyllaridae.ServerConfig{}
	server := &Server{Config: testConfig}

	req, err := http.NewRequest("PUT", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := server.SetupRouter()
	router.ServeHTTP(rr, req)
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
			expectedStatus:      http.StatusFailedDependency,
			returnedBody:        "foo",
			expectedBody:        "Failed Dependency\n",
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

			os.Setenv("SKIP_JWT_VERIFY", "true")
			os.Setenv("SCYLLARIDAE_YML", tt.yml)
			config, err := scyllaridae.ReadConfig("")

			sourceServer := createMockSourceServer(t, config, tt.mimetype, tt.authHeader, destinationServer.URL)
			defer sourceServer.Close()
			if err != nil {
				t.Fatalf("Could not read YML: %v", err)
			}

			// Create a Server instance with the test config
			server := &Server{Config: config}
			router := server.SetupRouter()

			// Configure and start the main server
			setupServer := httptest.NewServer(router)
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

func createMockJwksServer(t *testing.T, jwks []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(jwks)); err != nil {
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

func TestJwtAuth(t *testing.T) {
	os.Setenv("SKIP_JWT_VERIFY", "")
	goodPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal("Error generating RSA key:", err)
	}
	publicKey := &goodPrivateKey.PublicKey
	kid := fmt.Sprintf("%x", sha1.Sum(publicKey.N.Bytes()))

	badPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal("Error generating RSA key:", err)
	}

	jwk := JWK{
		Kty: "RSA",
		Kid: kid,
		Use: "sig",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes()),
	}

	jwks := JWKS{
		Keys: []JWK{jwk},
	}

	token, err := jwt.NewBuilder().
		Subject("1234567890").
		Claim("name", "foo").
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(1 * time.Hour)).
		Build()
	if err != nil {
		t.Fatal("Error building JWT:", err)
	}

	hdr := jws.NewHeaders()
	err = hdr.Set(jws.KeyIDKey, kid)
	if err != nil {
		t.Fatal("Error setting 'kid' header:", err)
	}
	signedToken, err := jwt.Sign(token,
		jwt.WithKey(jwa.RS256(), goodPrivateKey, jws.WithProtectedHeaders(hdr)),
	)
	if err != nil {
		t.Fatal("Error signing JWT:", err)
	}

	badToken, err := jwt.Sign(token,
		jwt.WithKey(jwa.RS256(), badPrivateKey, jws.WithProtectedHeaders(hdr)),
	)
	if err != nil {
		t.Fatal("Error signing JWT:", err)
	}

	iat := time.Now().Add(-24 * time.Hour)
	exp := time.Now().Add(-23*time.Hour + -20*time.Minute) // 86000 seconds ago

	token, err = jwt.NewBuilder().
		Subject("1234567890").
		Claim("name", "foo").
		IssuedAt(iat).
		Expiration(exp).
		Build()
	if err != nil {
		t.Fatal("Error building stale JWT:", err)
	}

	staleToken, err := jwt.Sign(token,
		jwt.WithKey(jwa.RS256(), goodPrivateKey, jws.WithProtectedHeaders(hdr)),
	)
	if err != nil {
		t.Fatal("Error signing stale JWT:", err)
	}

	tests := []Test{
		{
			name:                "cURL pass with auth",
			requestAuth:         fmt.Sprintf("bearer %s", signedToken),
			authHeader:          fmt.Sprintf("bearer %s", signedToken),
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
			name:                "cURL fail with stale token",
			requestAuth:         fmt.Sprintf("bearer %s", staleToken),
			authHeader:          fmt.Sprintf("bearer %s", staleToken),
			expectedStatus:      http.StatusUnauthorized,
			returnedBody:        "foo",
			expectedBody:        "Unauthorized\n",
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
			name:                "cURL fail with no auth",
			authHeader:          "foo",
			requestAuth:         "bar",
			expectedStatus:      http.StatusBadRequest,
			returnedBody:        "foo",
			expectedBody:        "Missing Authorization header\n",
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
			name:                "cURL fail with bad key",
			requestAuth:         fmt.Sprintf("bearer %s", badToken),
			authHeader:          fmt.Sprintf("bearer %s", badToken),
			expectedStatus:      http.StatusUnauthorized,
			returnedBody:        "foo",
			expectedBody:        "Unauthorized\n",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			jwksJSON, err := json.Marshal(jwks)
			if err != nil {
				t.Fatalf("Could not marshal jwks YML: %v", err)
			}

			jwksServer := createMockJwksServer(t, jwksJSON)
			defer jwksServer.Close()
			os.Setenv("JWKS_URI", jwksServer.URL)

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
			router := server.SetupRouter()

			// Configure and start the main server
			setupServer := httptest.NewServer(router)
			defer setupServer.Close()

			// Send the mock message to the main server
			req, err := http.NewRequest("GET", setupServer.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Islandora-Args", destinationServer.URL)
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
