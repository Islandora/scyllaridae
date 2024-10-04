package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
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

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if r.Header.Get("Apix-Ldp-Resource") == "" && r.Header.Get("X-Islandora-Event") == "" {
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

	// Stream the file contents from the source URL
	req, err := http.NewRequest("GET", message.Attachment.Content.SourceURI, nil)
	if err != nil {
		slog.Error("Error creating request to source", "source", message.Attachment.Content.SourceURI, "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	if s.Config.ForwardAuth {
		req.Header.Set("Authorization", auth)
	}
	sourceResp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Error fetching source file contents", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer sourceResp.Body.Close()
	if sourceResp.StatusCode != http.StatusOK {
		slog.Error("SourceURI sent a bad status code", "code", sourceResp.StatusCode, "uri", message.Attachment.Content.SourceURI)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	cmd, err := scyllaridae.BuildExecCommand(message, s.Config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	cmd.Stdin = sourceResp.Body

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
