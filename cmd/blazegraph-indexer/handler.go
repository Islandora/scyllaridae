package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/islandora/scyllaridae/pkg/api"
)

// BlazegraphHandler implements api.MessageHandler for the triplestore indexer service.
type BlazegraphHandler struct {
	Config *TriplestoreConfig
}

// Handle processes an incoming event and indexes/deletes from the triplestore.
func (h *BlazegraphHandler) Handle(payload api.Payload, auth string) (int, []byte, string, error) {
	token := auth
	if token == "" {
		token = payload.Authorization
	}

	// Extract subject URL: prefer rel="describes", fall back to rel="canonical"
	subjectURL := findURLByRel(payload.Object.URL, "describes")
	if subjectURL == "" {
		subjectURL = findURLByRel(payload.Object.URL, "canonical")
	}
	if subjectURL == "" {
		return http.StatusBadRequest, nil, "", fmt.Errorf("no subject URL (describes or canonical) found in event")
	}

	eventType := strings.ToLower(payload.Type)
	summary := strings.ToLower(payload.Summary)

	slog.Info("Processing triplestore event",
		"type", eventType,
		"summary", summary,
		"subject", subjectURL,
	)

	// Delete events
	if strings.Contains(eventType, "delete") || strings.Contains(summary, "delete") {
		return h.handleDelete(subjectURL)
	}

	// Index events: fetch JSON-LD, convert to N-Triples, update triplestore
	return h.handleIndex(payload, subjectURL, token)
}

func (h *BlazegraphHandler) handleDelete(subjectURL string) (int, []byte, string, error) {
	query := DeleteWhere(subjectURL, h.Config.NamedGraph)
	body := EncodeUpdateBody(query)

	status, err := h.postToTriplestore(body)
	if err != nil {
		return status, nil, "", err
	}

	slog.Info("Deleted from triplestore", "subject", subjectURL)
	return http.StatusOK, []byte("deleted"), "text/plain", nil
}

func (h *BlazegraphHandler) handleIndex(payload api.Payload, subjectURL, token string) (int, []byte, string, error) {
	// Find JSON-LD URL
	jsonldURL := findURLByMediaType(payload.Object.URL, "application/ld+json")
	if jsonldURL == "" {
		return http.StatusBadRequest, nil, "", fmt.Errorf("no JSON-LD URL found in event")
	}

	// Fetch JSON-LD from Drupal
	jsonldBytes, err := fetchContent(jsonldURL, token)
	if err != nil {
		return http.StatusBadGateway, nil, "", fmt.Errorf("error fetching JSON-LD from %s: %w", jsonldURL, err)
	}

	// Convert JSON-LD to N-Triples
	ntriples, err := JsonldToNTriples(jsonldBytes)
	if err != nil {
		return http.StatusInternalServerError, nil, "", fmt.Errorf("error converting JSON-LD to N-Triples: %w", err)
	}

	// Build combined DELETE + INSERT query
	query := BuildUpdateQuery(subjectURL, ntriples, h.Config.NamedGraph)
	body := EncodeUpdateBody(query)

	status, err := h.postToTriplestore(body)
	if err != nil {
		return status, nil, "", err
	}

	slog.Info("Indexed in triplestore", "subject", subjectURL)
	return http.StatusOK, []byte("indexed"), "text/plain", nil
}

func (h *BlazegraphHandler) postToTriplestore(body string) (int, error) {
	req, err := http.NewRequest("POST", h.Config.TriplestoreURL, strings.NewReader(body))
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error building triplestore request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error posting to triplestore: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("triplestore returned %d: %s", resp.StatusCode, string(respBody))
	}

	return resp.StatusCode, nil
}

func fetchContent(url, token string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func findURLByRel(urls []api.Link, rel string) string {
	for _, u := range urls {
		if u.Rel == rel {
			return u.Href
		}
	}
	return ""
}

func findURLByMediaType(urls []api.Link, mediaType string) string {
	for _, u := range urls {
		if u.MediaType == mediaType {
			return u.Href
		}
	}
	return ""
}
