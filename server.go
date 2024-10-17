package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Server struct {
	Config *scyllaridae.ServerConfig
}

func runHTTPServer(server *Server) {
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Use the method as the handler
	http.HandleFunc("/", server.MessageHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server listening", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func (s *Server) MessageHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info(r.RequestURI, "method", r.Method, "ip", r.RemoteAddr, "proto", r.Proto)

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if r.Header.Get("Apix-Ldp-Resource") == "" && r.Header.Get("X-Islandora-Event") == "" && r.Method == http.MethodGet {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Read the Alpaca message payload
	auth := ""
	if s.Config.ForwardAuth {
		auth = r.Header.Get("Authorization")
	}
	message, err := api.DecodeAlpacaMessage(r, auth)
	if err != nil {
		slog.Error("Error decoding alpaca message", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	cmd, err := scyllaridae.BuildExecCommand(message, s.Config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

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

	slog.Info("Running command", "cmd", cmd.String())
	if err := cmd.Run(); err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
}
