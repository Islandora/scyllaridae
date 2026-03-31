package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIWithStdin(t *testing.T) {
	t.Setenv("SCYLLARIDAE_YML", "")
	t.Setenv("SCYLLARIDAE_YML_PATH", "")

	configPath := writeTestConfig(t)
	message := encodeTestMessage(t, `{"type":"test","object":{"id":"123"},"attachment":{"content":{"source_mimetype":"text/plain"}}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"--yml", configPath, "--message", message},
		bytes.NewBufferString("stdin payload"),
		&stdout,
		&stderr,
	)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", exitCode, stderr.String())
	}
	outputPath := strings.TrimSpace(stdout.String())
	if outputPath == "" {
		t.Fatalf("expected stdout to contain temp output path")
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if got := string(output); got != "stdin payload" {
		t.Fatalf("expected temp output to equal stdin payload, got %q", got)
	}
}

func TestRunCLIWithFile(t *testing.T) {
	t.Setenv("SCYLLARIDAE_YML", "")
	t.Setenv("SCYLLARIDAE_YML_PATH", "")

	configPath := writeTestConfig(t)
	message := encodeTestMessage(t, `{"type":"test","object":{"id":"456"},"attachment":{"content":{"source_mimetype":"text/plain"}}}`)

	inputPath := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(inputPath, []byte("file payload"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"--yml", configPath, "--message", message, "--file", inputPath},
		nil,
		&stdout,
		&stderr,
	)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", exitCode, stderr.String())
	}
	outputPath := strings.TrimSpace(stdout.String())
	if outputPath == "" {
		t.Fatalf("expected stdout to contain temp output path")
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if got := string(output); got != "file payload" {
		t.Fatalf("expected destination file to contain file payload, got %q", got)
	}
}

func TestRunCLIRequiresMessage(t *testing.T) {
	t.Setenv("SCYLLARIDAE_YML", "")
	t.Setenv("SCYLLARIDAE_YML_PATH", "")

	configPath := writeTestConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"--yml", configPath, "--file", filepath.Join(t.TempDir(), "input.txt")},
		nil,
		&stdout,
		&stderr,
	)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "scyllaridae.yml")
	config := `allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "cat"
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func encodeTestMessage(t *testing.T, message string) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(message))
}
