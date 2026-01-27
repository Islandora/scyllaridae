package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/islandora/scyllaridae/internal/config"
	"github.com/islandora/scyllaridae/internal/server"
)

func main() {
	setupBlazegraphLogger()

	cfg := LoadTriplestoreConfig()

	handler := &BlazegraphHandler{
		Config: cfg,
	}

	// Build a ServerConfig with the custom handler
	fa := true
	serverCfg := &config.ServerConfig{
		ForwardAuth:   &fa,
		JwksUri:       os.Getenv("SCYLLARIDAE_JWKS_URI"),
		CustomHandler: handler,
	}

	s := &server.Server{
		Config: serverCfg,
	}

	slog.Info("Starting blazegraph-indexer",
		"triplestoreURL", cfg.TriplestoreURL,
		"namedGraph", cfg.NamedGraph,
	)

	server.RunHTTPServer(s)
}

func setupBlazegraphLogger() {
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
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}
