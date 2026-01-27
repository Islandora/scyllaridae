package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"
)

// ProcessJsonld filters a JSON-LD @graph array to the entry matching the
// Drupal URL (scheme-agnostic), rewrites its @id to the Fedora URL,
// and returns the filtered array.
func ProcessJsonld(jsonldBytes []byte, drupalURL, fedoraURL string) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(jsonldBytes, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-LD: %w", err)
	}

	graphRaw, ok := doc["@graph"]
	if !ok {
		return nil, fmt.Errorf("no @graph key in JSON-LD")
	}

	var graph []map[string]json.RawMessage
	if err := json.Unmarshal(graphRaw, &graph); err != nil {
		return nil, fmt.Errorf("failed to parse @graph: %w", err)
	}

	drupalHostPath, err := hostPath(drupalURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse drupal URL %s: %w", drupalURL, err)
	}

	slog.Debug("Processing JSON-LD", "drupalURL", drupalURL, "fedoraURL", fedoraURL, "drupalHostPath", drupalHostPath)

	var filtered []map[string]json.RawMessage
	for _, entry := range graph {
		idRaw, exists := entry["@id"]
		if !exists {
			continue
		}
		var entryID string
		if err := json.Unmarshal(idRaw, &entryID); err != nil {
			continue
		}

		entryHostPath, err := hostPath(entryID)
		if err != nil {
			continue
		}

		if entryHostPath == drupalHostPath {
			// Rewrite @id to Fedora URL
			newID, _ := json.Marshal(fedoraURL)
			entry["@id"] = newID
			filtered = append(filtered, entry)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no @graph entry matched Drupal URL %s", drupalURL)
	}

	return json.Marshal(filtered)
}

// GetModifiedTimestamp extracts the modified timestamp from a JSON-LD
// resource array using the specified predicate.
func GetModifiedTimestamp(jsonldBytes []byte, predicate string) (int64, error) {
	var resources []map[string]json.RawMessage
	if err := json.Unmarshal(jsonldBytes, &resources); err != nil {
		return 0, fmt.Errorf("failed to parse JSON-LD array: %w", err)
	}

	if len(resources) == 0 {
		return 0, fmt.Errorf("empty JSON-LD array")
	}

	predRaw, ok := resources[0][predicate]
	if !ok {
		return 0, fmt.Errorf("predicate %s not found", predicate)
	}

	var values []map[string]string
	if err := json.Unmarshal(predRaw, &values); err != nil {
		return 0, fmt.Errorf("failed to parse predicate values: %w", err)
	}

	if len(values) == 0 {
		return 0, fmt.Errorf("no values for predicate %s", predicate)
	}

	dateStr, ok := values[0]["@value"]
	if !ok {
		return 0, fmt.Errorf("no @value in predicate %s", predicate)
	}

	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
	}

	return t.Unix(), nil
}

// hostPath extracts host+path from a URL, ignoring scheme for comparison.
func hostPath(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Host + u.Path, nil
}
