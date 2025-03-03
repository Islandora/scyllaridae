package server

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gorilla/mux"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Server struct {
	Config  *scyllaridae.ServerConfig
	KeySets *lru.LRU[string, jwk.Set]
}

func RunHTTPServer(server *Server) {
	r := server.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server listening", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		panic(err)
	}
}

func (server *Server) SetupRouter() *mux.Router {
	if os.Getenv("JWKS_URI") == "" && os.Getenv("SKIP_JWT_VERIFY") != "true" {
		slog.Error("Need to provide your JWKS URI in the JWKS_URI e.g. JWKS_URI=https://islandora.dev/oauth/discovery/keys")
		os.Exit(1)
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

func (s *Server) MessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Header.Get("Apix-Ldp-Resource") == "" && r.Header.Get("X-Islandora-Event") == "" && r.Method == http.MethodGet {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	auth := ""
	if s.Config.ForwardAuth {
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

	// Send stdout to the ResponseWriter stream
	cmd.Stdout = w
	if err := cmd.Run(); err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}
