package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/internal/server"
)

func main() {
	setupLogger()

	config, err := config.ReadConfig()
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	s := &server.Server{
		Config: config,
	}
	server.RunHTTPServer(s)
}

func setupLogger() {
	logLevel := strings.ToUpper(os.Getenv("LOG_LEVEL"))
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
