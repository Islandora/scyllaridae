package main

import "os"

// TriplestoreConfig holds configuration for the blazegraph indexer service.
type TriplestoreConfig struct {
	// TriplestoreURL is the SPARQL endpoint URL.
	TriplestoreURL string
	// NamedGraph is an optional named graph URI.
	NamedGraph string
}

// LoadTriplestoreConfig reads configuration from environment variables.
func LoadTriplestoreConfig() *TriplestoreConfig {
	return &TriplestoreConfig{
		TriplestoreURL: envOrDefault("TRIPLESTORE_URL", "http://blazegraph:8080/bigdata/namespace/islandora/sparql"),
		NamedGraph:     os.Getenv("TRIPLESTORE_NAMED_GRAPH"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
