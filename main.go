package main

import (
	"log/slog"
	"os"

	"github.com/lehigh-university-libraries/scyllaridae/internal/config"
	"github.com/lehigh-university-libraries/scyllaridae/internal/server"
)

func main() {
	config, err := config.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	s := &server.Server{
		Config: config,
	}
	server.RunHTTPServer(s)
}
