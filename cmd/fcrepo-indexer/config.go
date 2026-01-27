package main

import (
	"os"
	"strings"
)

// FcrepoIndexerConfig holds configuration for the fcrepo indexer service.
type FcrepoIndexerConfig struct {
	// FedoraURL is the base URL for Fedora REST API.
	FedoraURL string
	// ModifiedDatePredicate is the RDF predicate for timestamp comparison.
	ModifiedDatePredicate string
	// StripFormatJsonld controls whether to strip ?_format=jsonld from URLs.
	StripFormatJsonld bool
	// IsFedora6 enables Fedora 6 specific headers.
	IsFedora6 bool
}

// LoadFcrepoConfig reads configuration from environment variables.
func LoadFcrepoConfig() *FcrepoIndexerConfig {
	cfg := &FcrepoIndexerConfig{
		FedoraURL:             envOrDefault("FCREPO_INDEXER_FEDORA_URL", "http://fcrepo:8080/fcrepo/rest"),
		ModifiedDatePredicate: envOrDefault("FCREPO_INDEXER_MODIFIED_PREDICATE", "http://schema.org/dateModified"),
		StripFormatJsonld:     envBoolOrDefault("FCREPO_INDEXER_STRIP_FORMAT_JSONLD", true),
		IsFedora6:             envBoolOrDefault("FCREPO_INDEXER_IS_FEDORA6", true),
	}
	return cfg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBoolOrDefault(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return strings.EqualFold(v, "true") || v == "1"
}
