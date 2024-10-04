package main

import (
	"log/slog"
	"os"

	scyllaridae "github.com/lehigh-university-libraries/scyllaridae/internal/config"
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
	if len(config.QueueMiddlewares) > 0 {
		runStompSubscribers(config)
	} else {
		runHTTPServer(config)
	}
}

