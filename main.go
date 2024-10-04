package main

import (
	"log/slog"
	"os"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
)

func main() {
	config, err := scyllaridae.ReadConfig("scyllaridae.yml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}

	if len(config.QueueMiddlewares) > 0 {
		runStompSubscribers(config)
	} else {
		server := &Server{Config: config}
		runHTTPServer(server)
	}
}
