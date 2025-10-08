package server

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/mux"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	scyllaridae "github.com/islandora/scyllaridae/internal/config"
	"github.com/islandora/scyllaridae/pkg/api"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Server struct {
	Config  *scyllaridae.ServerConfig
	KeySets *lru.LRU[string, jwk.Set]
}

// RunHTTPServer starts the HTTP server and listens on the configured port.
// The port is determined by the SCYLLARIDAE_PORT environment variable, defaulting to 8080.
// This function blocks and will panic if the server fails to start.
func RunHTTPServer(server *Server) {
	r := server.SetupRouter()

	port := os.Getenv("SCYLLARIDAE_PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server listening", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		panic(err)
	}
}

func (server *Server) SetupRouter() *mux.Router {
	if server.Config.JwksUri == "" {
		slog.Info("No JWKS URI configured, skipping JWT verification")
	}

	server.KeySets = lru.NewLRU[string, jwk.Set](25, nil, time.Minute*15)

	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}).Methods("GET")

	// create the main route with logging and JWT auth middleware
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(server.LoggingMiddleware, server.JWTAuthMiddleware)
	authRouter.HandleFunc("/", server.MessageHandler).Methods("GET", "POST")

	// make sure 404s get logged
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		http.Error(w, "404 Not Found", http.StatusNotFound)
	})
	authRouter.NotFoundHandler = server.LoggingMiddleware(notFoundHandler)

	return r
}

// bufferingWriter buffers initial output to detect early command failures.
// Once the buffer threshold is reached, it flushes and streams remaining output.
type bufferingWriter struct {
	w           http.ResponseWriter
	buffer      *bytes.Buffer
	maxBuffer   int
	flushed     bool
	totalWrites int
}

func (bw *bufferingWriter) Write(p []byte) (int, error) {
	bw.totalWrites++

	// If already flushed, stream directly
	if bw.flushed {
		return bw.w.Write(p)
	}

	// Buffer until we reach threshold
	if bw.buffer.Len() < bw.maxBuffer {
		writeSize := min(len(p), bw.maxBuffer-bw.buffer.Len())
		bw.buffer.Write(p[:writeSize])

		// If buffer is full, flush it and write remainder
		if bw.buffer.Len() >= bw.maxBuffer {
			if err := bw.flush(); err != nil {
				return 0, err
			}
			// Write any remaining data
			if writeSize < len(p) {
				n, err := bw.w.Write(p[writeSize:])
				return writeSize + n, err
			}
		}
		return len(p), nil
	}

	// Should not reach here, but handle it
	return bw.w.Write(p)
}

func (bw *bufferingWriter) flush() error {
	if bw.flushed {
		return nil
	}
	bw.flushed = true
	if bw.buffer.Len() > 0 {
		_, err := io.Copy(bw.w, bw.buffer)
		return err
	}
	return nil
}

func (s *Server) MessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Header.Get("Apix-Ldp-Resource") == "" && r.Header.Get("X-Islandora-Event") == "" && r.Method == http.MethodGet {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	auth := ""
	if *s.Config.ForwardAuth {
		auth = r.Header.Get("Authorization")
	}
	cmd := r.Context().Value(cmdKey).(*exec.Cmd)
	message := r.Context().Value(msgKey).(api.Payload)

	// Stream the file contents from the source URL or request body
	fs, errCode, err := s.Config.GetFileStream(r, message, auth)
	if err != nil {
		http.Error(w, cases.Title(language.English).String(fmt.Sprint(err)), errCode)
		return
	}
	if fs != nil {
		defer fs.Close()
		cmd.Stdin = fs
	}

	// Create a buffer to capture stderr
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	// Use buffering writer to detect early failures (buffer first 2MB)
	const bufferSize = 2 * 1024 * 1024 // 2MB
	bw := &bufferingWriter{
		w:         w,
		buffer:    bytes.NewBuffer(make([]byte, 0, bufferSize)),
		maxBuffer: bufferSize,
		flushed:   false,
	}
	cmd.Stdout = bw

	err = cmd.Run()
	if err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "cmdStdErr", stdErr.String())
		// If buffer hasn't been flushed yet, we can still send an error response
		if !bw.flushed {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		// Headers already sent - partial output delivered with 200 status
		// Log the error but can't change response status
		slog.Warn("Command failed after streaming started", "cmd", cmd.String(), "bytesWritten", bw.totalWrites)
		return
	}

	// Command succeeded - flush any remaining buffered data
	if err := bw.flush(); err != nil {
		slog.Error("Error flushing output", "err", err)
		return
	}
	slog.Debug("Command completed", "msgId", message.Object.ID, "cmd", cmd.String(), "cmdStdErr", stdErr.String())
}
