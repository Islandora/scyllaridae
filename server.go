package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"
	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Server struct {
	Config *scyllaridae.ServerConfig
}

func (server *Server) SetupRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}).Methods("GET")

	// create the main route with logging and JWT auth middleware
	authRouter := r.PathPrefix("/").Subrouter()
	authRouter.Use(server.LoggingMiddleware, JWTAuthMiddleware)
	authRouter.HandleFunc("/", server.MessageHandler).Methods("GET", "POST")

	// make sure 404s get logged
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		http.Error(w, "404 Not Found", http.StatusNotFound)
	})
	authRouter.NotFoundHandler = server.LoggingMiddleware(notFoundHandler)

	return r
}

func runHTTPServer(server *Server) {
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
