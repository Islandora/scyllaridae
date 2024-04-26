package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

var (
	config *scyllaridae.ServerConfig
)

func init() {
	var err error

	config, err = scyllaridae.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
}

func main() {
	http.HandleFunc("/", MessageHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server listening", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// Read the Alpaca message payload
	message, err := api.DecodeAlpacaMessage(r)
	if err != nil {
		slog.Error("Error decoding Pub/Sub message", "err", err)
		http.Error(w, "Error decoding Pub/Sub message", http.StatusInternalServerError)
		return
	}

	// Stream the file contents from the source URL
	sourceResp, err := http.Get(message.Attachment.Content.SourceURI)
	if err != nil {
		slog.Error("Error fetching source file contents", "err", err)
		http.Error(w, "Error fetching file contents from URL", http.StatusInternalServerError)
		return
	}
	defer sourceResp.Body.Close()

	// build a command to run that we will pipe the stdin stream into
	cmd, err := scyllaridae.BuildExecCommand(message.Attachment.Content.MimeType, message.Attachment.Content.Args, config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	cmd.Stdin = sourceResp.Body

	// Create a buffer to stream the output of the command
	var stdErr bytes.Buffer
	cmd.Stderr = &stdErr

	// send stdout to the ResponseWriter stream
	cmd.Stdout = w

	slog.Info("Running command", "cmd", cmd.String())
	if err := cmd.Run(); err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
		http.Error(w, "Error running command", http.StatusInternalServerError)
		return
	}
}
