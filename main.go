package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"

	"gopkg.in/yaml.v3"
)

var (
	config *scyllaridae.ServerConfig
)

func init() {
	var err error

	config, err = ReadConfig("scyllaridae.yml")
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
	if r.Method != http.MethodPost {
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

	// Fetch the file contents from the URL
	sourceResp, err := http.Get(message.Attachment.Content.SourceURI)
	if err != nil {
		slog.Error("Error fetching source file contents", "err", err)
		http.Error(w, "Error fetching file contents from URL", http.StatusInternalServerError)
		return
	}
	defer sourceResp.Body.Close()

	arg := r.Header.Get(config.ArgHeader)
	cmd, err := buildExecCommand(message.Attachment.Content.MimeType, arg, config)
	if err != nil {
		slog.Error("Error building command", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	cmd.Stdin = sourceResp.Body

	// Create a buffer to store the output
	var outBuf, stdErr bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &stdErr

	slog.Info("Running command", "cmd", cmd.String())

	if err := cmd.Run(); err != nil {
		slog.Error("Error running command", "cmd", cmd.String(), "err", stdErr.String())
		http.Error(w, "Error running convert command", http.StatusInternalServerError)
		return
	}

	// Create the PUT request
	req, err := http.NewRequest(config.DestinationHTTPMethod, message.Attachment.Content.DestinationURI, &outBuf)
	if err != nil {
		slog.Error("Error creating HTTP request", "err", err)
		http.Error(w, "Error creating HTTP request", http.StatusInternalServerError)
		return
	}
	if config.ForwardAuth {
		auth := r.Header.Get("Authorization")
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Content-Type", message.Attachment.Content.MimeType)
	req.Header.Set("Content-Location", message.Attachment.Content.FileUploadURI)

	// Execute the PUT request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request", "method", config.DestinationHTTPMethod, "err", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		slog.Error("Request failed on destination server", "code", resp.StatusCode)
		http.Error(w, fmt.Sprintf("%s request failed with status code %d", config.DestinationHTTPMethod, resp.StatusCode), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte(""))
	if err != nil {
		slog.Error("Error writing response", "err", err)
	}
}

func ReadConfig(yp string) (*scyllaridae.ServerConfig, error) {
	var (
		y   []byte
		err error
	)
	yml := os.Getenv("SCYLLARIDAE_YML")
	if yml != "" {
		y = []byte(yml)
	} else {
		y, err = os.ReadFile(yp)
		if err != nil {
			return nil, err
		}
	}

	var c scyllaridae.ServerConfig
	err = yaml.Unmarshal(y, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func buildExecCommand(mimetype, addtlArgs string, c *scyllaridae.ServerConfig) (*exec.Cmd, error) {
	var cmdConfig scyllaridae.Command
	var exists bool
	if scyllaridae.IsAllowedMimeType(mimetype, c.AllowedMimeTypes) {
		cmdConfig, exists = c.CmdByMimeType[mimetype]
		if !exists || (len(cmdConfig.Cmd) == 0) {
			// Fallback to default if specific MIME type not configured or if command is empty
			cmdConfig = c.CmdByMimeType["default"]
		}
	} else {
		return nil, fmt.Errorf("undefined mimetype: %s", mimetype)
	}

	args := []string{}
	for _, a := range cmdConfig.Args {
		if a == "%s" && addtlArgs != "" {
			args = append(args, addtlArgs)
		} else {
			args = append(args, a)
		}
	}

	cmd := exec.Command(cmdConfig.Cmd, args...)

	return cmd, nil
}
