package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// saveNode checks if a node exists in Fedora and creates or updates it.
func saveNode(fedoraClient *FedoraClient, cfg *FcrepoIndexerConfig, uuid, jsonldURL, fedoraBaseURL, token string) (int, error) {
	path, err := GetFedoraPath(uuid)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID %s: %w", uuid, err)
	}
	fedoraBaseURL = strings.TrimRight(fedoraBaseURL, "/")
	fedoraURL := fedoraBaseURL + "/" + path

	resp, err := fedoraClient.Head(fedoraURL, token)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error checking Fedora resource %s: %w", fedoraURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		slog.Debug("Resource not found, creating", "fedoraURL", fedoraURL)
		return createNode(fedoraClient, cfg, jsonldURL, fedoraURL, token)
	}
	slog.Debug("Resource exists, updating", "fedoraURL", fedoraURL)
	return updateNode(fedoraClient, cfg, jsonldURL, fedoraURL, token)
}

// createNode fetches JSON-LD from Drupal, processes it, and PUTs it to Fedora.
func createNode(fedoraClient *FedoraClient, cfg *FcrepoIndexerConfig, jsonldURL, fedoraURL, token string) (int, error) {
	// Fetch JSON-LD from Drupal
	drupalBody, err := fetchFromDrupal(jsonldURL, token)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error fetching JSON-LD from Drupal %s: %w", jsonldURL, err)
	}

	subjectURL := jsonldURL
	if cfg.StripFormatJsonld {
		subjectURL = strings.TrimSuffix(subjectURL, "?_format=jsonld")
	}

	// Process JSON-LD: filter @graph, rewrite @id
	processed, err := ProcessJsonld(drupalBody, subjectURL, fedoraURL)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error processing JSON-LD: %w", err)
	}

	// PUT to Fedora
	headers := map[string]string{
		"Content-Type": "application/ld+json",
		"Prefer":       "return=minimal; handling=lenient",
	}
	resp, err := fedoraClient.Put(fedoraURL, token, bytes.NewReader(processed), headers)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error saving to Fedora %s: %w", fedoraURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("PUT %s returned %d: %s", fedoraURL, resp.StatusCode, string(body))
	}

	slog.Info("Created node in Fedora", "fedoraURL", fedoraURL, "status", resp.StatusCode)
	return resp.StatusCode, nil
}

// updateNode fetches from Fedora and Drupal, compares timestamps, and conditionally updates.
func updateNode(fedoraClient *FedoraClient, cfg *FcrepoIndexerConfig, jsonldURL, fedoraURL, token string) (int, error) {
	// GET from Fedora
	fedoraHeaders := map[string]string{
		"Accept": "application/ld+json",
	}
	if cfg.IsFedora6 {
		fedoraHeaders["Prefer"] = `return=representation; omit="http://fedora.info/definitions/v4/repository#ServerManaged"`
	}
	fedoraResp, err := fedoraClient.Get(fedoraURL, token, fedoraHeaders)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error getting Fedora resource %s: %w", fedoraURL, err)
	}
	defer fedoraResp.Body.Close()

	if fedoraResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(fedoraResp.Body)
		return fedoraResp.StatusCode, fmt.Errorf("GET %s returned %d: %s", fedoraURL, fedoraResp.StatusCode, string(body))
	}

	// Extract state token
	stateTokens := fedoraResp.Header.Values("X-State-Token")
	stateToken := ""
	if len(stateTokens) > 0 {
		stateToken = `"` + strings.TrimLeft(stateTokens[0], "W/") + `"`
	}
	slog.Debug("State token", "stateToken", stateToken)

	fedoraBody, err := io.ReadAll(fedoraResp.Body)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error reading Fedora response: %w", err)
	}

	// Parse modified timestamp from Fedora (0 if not found, e.g. new media)
	fedoraModified, err := GetModifiedTimestamp(fedoraBody, cfg.ModifiedDatePredicate)
	if err != nil {
		slog.Debug("Could not get modified timestamp from Fedora, using 0", "err", err)
		fedoraModified = 0
	}

	// GET from Drupal
	drupalBody, err := fetchFromDrupal(jsonldURL, token)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error fetching JSON-LD from Drupal %s: %w", jsonldURL, err)
	}

	// Check for "describes" link header in Drupal response for subject URL
	subjectURL := jsonldURL
	if cfg.StripFormatJsonld {
		subjectURL = strings.TrimSuffix(subjectURL, "?_format=jsonld")
	}

	// Process Drupal JSON-LD
	drupalProcessed, err := ProcessJsonld(drupalBody, subjectURL, fedoraURL)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error processing Drupal JSON-LD: %w", err)
	}

	// Get modified timestamp from Drupal
	drupalModified, err := GetModifiedTimestamp(drupalProcessed, cfg.ModifiedDatePredicate)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting modified timestamp from Drupal: %w", err)
	}

	// Abort if Drupal version is not newer
	if drupalModified <= fedoraModified {
		return http.StatusPreconditionFailed, fmt.Errorf("not updating %s because RDF at %s is not newer", fedoraURL, jsonldURL)
	}

	// Conditionally PUT to Fedora
	headers := map[string]string{
		"Content-Type": "application/ld+json",
	}
	if cfg.IsFedora6 {
		headers["Prefer"] = "handling=lenient"
	} else {
		headers["Prefer"] = "handling=lenient;received=minimal"
	}
	if stateToken != "" {
		headers["X-If-State-Match"] = stateToken
	}

	resp, err := fedoraClient.Put(fedoraURL, token, bytes.NewReader(drupalProcessed), headers)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error saving to Fedora %s: %w", fedoraURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("PUT %s returned %d: %s", fedoraURL, resp.StatusCode, string(body))
	}

	slog.Info("Updated node in Fedora", "fedoraURL", fedoraURL, "status", resp.StatusCode)
	return resp.StatusCode, nil
}

// deleteNode removes a resource from Fedora.
func deleteNode(fedoraClient *FedoraClient, uuid, fedoraBaseURL, token string) (int, error) {
	path, err := GetFedoraPath(uuid)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID %s: %w", uuid, err)
	}
	fedoraBaseURL = strings.TrimRight(fedoraBaseURL, "/")
	fedoraURL := fedoraBaseURL + "/" + path

	resp, err := fedoraClient.Delete(fedoraURL, token)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error deleting Fedora resource %s: %w", fedoraURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusGone {
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("DELETE %s returned %d: %s", fedoraURL, resp.StatusCode, string(body))
	}

	slog.Info("Deleted node from Fedora", "fedoraURL", fedoraURL, "status", resp.StatusCode)
	return resp.StatusCode, nil
}

// fetchFromDrupal fetches content from a Drupal URL with optional auth.
func fetchFromDrupal(url, token string) ([]byte, error) {
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
