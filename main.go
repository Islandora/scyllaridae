package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/islandora/scyllaridae/internal/config"
	"github.com/islandora/scyllaridae/internal/server"
	"github.com/islandora/scyllaridae/pkg/api"
)

func main() {
	setupLogger()

	stdin := io.Reader(os.Stdin)
	if info, err := os.Stdin.Stat(); err == nil && (info.Mode()&os.ModeCharDevice) != 0 {
		stdin = nil
	}

	os.Exit(run(os.Args[1:], stdin, os.Stdout, os.Stderr))
}

type cliOptions struct {
	ymlPath  string
	message  string
	filePath string
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	opts, err := parseCLIOptions(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	cfg, err := readConfig(opts)
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		return 1
	}

	if !opts.isCLIMode() {
		s := &server.Server{
			Config: cfg,
		}
		server.RunHTTPServer(s)
		return 0
	}

	if err := runCLI(opts, cfg, stdin, stdout, stderr); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		slog.Error("CLI execution failed", "err", err)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}

func parseCLIOptions(args []string, stderr io.Writer) (cliOptions, error) {
	var opts cliOptions

	fs := flag.NewFlagSet("scyllaridae", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.ymlPath, "yml", "", "Path to the scyllaridae YAML config")
	fs.StringVar(&opts.message, "message", "", "Base64-encoded Islandora event message")
	fs.StringVar(&opts.filePath, "file", "", "Path to the input file to stream to the command")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if fs.NArg() != 0 {
		return cliOptions{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if opts.isCLIMode() && opts.message == "" {
		return cliOptions{}, fmt.Errorf("--message is required in CLI mode")
	}

	return opts, nil
}

func (opts cliOptions) isCLIMode() bool {
	return opts.message != "" || opts.filePath != ""
}

func readConfig(opts cliOptions) (*config.ServerConfig, error) {
	if opts.ymlPath != "" {
		return config.ReadConfigFromPath(opts.ymlPath)
	}
	return config.ReadConfig()
}

func runCLI(opts cliOptions, cfg *config.ServerConfig, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	messageBytes, err := base64.StdEncoding.DecodeString(opts.message)
	if err != nil {
		return fmt.Errorf("could not decode --message: %w", err)
	}

	message, err := api.DecodeEventMessage(messageBytes)
	if err != nil {
		return fmt.Errorf("could not parse --message payload: %w", err)
	}
	if err := validateCLIMessage(message); err != nil {
		return err
	}

	cmd, err := config.BuildExecCommand(message, cfg)
	if err != nil {
		return fmt.Errorf("could not build command: %w", err)
	}

	input, cleanupInput, err := openCLIInput(opts.filePath, stdin)
	if err != nil {
		return err
	}
	defer cleanupInput()
	if input != nil {
		cmd.Stdin = input
	}

	output, outputPath, cleanupOutput, err := openCLIOutput()
	if err != nil {
		return err
	}
	defer cleanupOutput()
	cmd.Stdout = output
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(stdout, outputPath); err != nil {
		return fmt.Errorf("could not write output path: %w", err)
	}

	return nil
}

func validateCLIMessage(message api.Payload) error {
	if strings.TrimSpace(message.Attachment.Content.SourceMimeType) == "" {
		return fmt.Errorf("attachment.content.source_mimetype is required in CLI mode")
	}

	return nil
}

func openCLIInput(filePath string, stdin io.Reader) (io.Reader, func(), error) {
	if filePath == "" {
		return stdin, func() {}, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, func() {}, fmt.Errorf("could not open --file: %w", err)
	}

	return f, func() { _ = f.Close() }, nil
}

func openCLIOutput() (io.Writer, string, func(), error) {
	f, err := os.CreateTemp("", "scyllaridae-output-*")
	if err != nil {
		return nil, "", func() {}, fmt.Errorf("could not create temporary output file: %w", err)
	}

	return f, f.Name(), func() { _ = f.Close() }, nil
}

func setupLogger() {
	logLevel := strings.ToUpper(os.Getenv("SCYLLARIDAE_LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "INFO"
	}

	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		slog.Info("Unknown log level", "logLevel", logLevel)
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}
