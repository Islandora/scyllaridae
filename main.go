package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type Data struct {
	Actor      Actor      `json:"actor"`
	Object     Object     `json:"object"`
	Attachment Attachment `json:"attachment"`
	Type       string     `json:"type"`
	Summary    string     `json:"summary"`
}

type Actor struct {
	Id string `json:"id"`
}

type Object struct {
	Id         string `json:"id"`
	URL        []URL  `json:"url"`
	NewVersion bool   `json:"isNewVersion"`
}

type URL struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Href      string `json:"href"`
	MediaType string `json:"mediaType"`
	Rel       string `json:"rel"`
}

type Attachment struct {
	Type      string  `json:"type"`
	Content   Content `json:"content"`
	MediaType string  `json:"mediaType"`
}

type Content struct {
	MimeType       string `json:"mimetype"`
	Args           string `json:"args"`
	SourceUri      string `json:"source_uri"`
	DestinationUri string `json:"destination_uri"`
	FileUploadUri  string `json:"file_upload_uri"`
	WebServiceUri  string
}

type Cmd struct {
	Command string   `yaml:"cmd,omitempty"`
	Args    []string `yaml:"args,omitempty"`
}

type Config struct {
	Label            string         `yaml:"label"`
	Method           string         `yaml:"destination-http-method"`
	FileHeader       string         `yaml:"file-header"`
	ArgHeader        string         `yaml:"arg-header"`
	ForwardAuth      bool           `yaml:"forward-auth"`
	AllowedMimeTypes []string       `yaml:"allowed-mimetypes"`
	Mimetypes        map[string]Cmd `yaml:"mimetypes"`
}

var (
	config *Config
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
	message, err := DecodeAlpacaMessage(r)
	if err != nil {
		slog.Error("Error decoding Pub/Sub message", "err", err)
		http.Error(w, "Error decoding Pub/Sub message", http.StatusInternalServerError)
		return
	}

	// Fetch the file contents from the URL
	sourceResp, err := http.Get(message.Attachment.Content.SourceUri)
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
	req, err := http.NewRequest(config.Method, message.Attachment.Content.DestinationUri, &outBuf)
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
	req.Header.Set("Content-Location", message.Attachment.Content.FileUploadUri)

	// Execute the PUT request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending request", "method", config.Method, "err", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		slog.Error("Request failed on destination server", "code", resp.StatusCode)
		http.Error(w, fmt.Sprintf("%s request failed with status code %d", config.Method, resp.StatusCode), resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte(""))
	if err != nil {
		slog.Error("Error writing response", "err", err)
	}
}

func DecodeAlpacaMessage(r *http.Request) (Data, error) {
	var d Data

	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		return Data{}, err
	}

	return d, nil
}

func ReadConfig(yp string) (*Config, error) {
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

	var c Config
	err = yaml.Unmarshal(y, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func buildExecCommand(mimetype, addtlArgs string, c *Config) (*exec.Cmd, error) {
	var cmdConfig Cmd
	var exists bool
	slog.Info("Allowed formats", "formats", c.AllowedMimeTypes)
	if isAllowedMIMEType(mimetype, c.AllowedMimeTypes) {
		cmdConfig, exists = c.Mimetypes[mimetype]
		if !exists || (len(cmdConfig.Command) == 0) {
			// Fallback to default if specific MIME type not configured or if command is empty
			cmdConfig = c.Mimetypes["default"]
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

	cmd := exec.Command(cmdConfig.Command, args...)

	return cmd, nil
}

func isAllowedMIMEType(mimetype string, allowedFormats []string) bool {
	for _, format := range allowedFormats {
		if format == mimetype {
			return true
		}
		if strings.HasSuffix(format, "/*") {
			// Check wildcard MIME type
			prefix := strings.TrimSuffix(format, "*")
			if strings.HasPrefix(mimetype, prefix) {
				return true
			}
		}
	}
	return false
}
