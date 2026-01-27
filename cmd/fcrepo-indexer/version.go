package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// createVersion creates a Memento version for a node in Fedora.
func createVersion(fedoraClient *FedoraClient, uuid, fedoraBaseURL, token string) (int, error) {
	path, err := GetFedoraPath(uuid)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID %s: %w", uuid, err)
	}
	fedoraBaseURL = strings.TrimRight(fedoraBaseURL, "/")
	fedoraURL := fedoraBaseURL + "/" + path

	return postVersion(fedoraClient, fedoraURL, token)
}

// createMediaVersion creates a Memento version for a media resource in Fedora.
func createMediaVersion(fedoraClient *FedoraClient, cfg *FcrepoIndexerConfig, sourceField, jsonURL, fedoraBaseURL, token string) (int, error) {
	urls, err := getMediaURLs(fedoraClient, sourceField, jsonURL, fedoraBaseURL, token)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error resolving media URLs: %w", err)
	}

	return postVersion(fedoraClient, urls.Fedora, token)
}

// postVersion sends a POST to {url}/fcr:versions to create a version.
func postVersion(fedoraClient *FedoraClient, fedoraURL, token string) (int, error) {
	versionURL := fedoraURL + "/fcr:versions"

	timestamp := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")

	req, err := http.NewRequest("POST", versionURL, nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error building POST request for %s: %w", versionURL, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Memento-Datetime", timestamp)

	resp, err := fedoraClient.HTTPClient.Do(req)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error creating version at %s: %w", versionURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("POST %s returned %d: %s", versionURL, resp.StatusCode, string(body))
	}

	slog.Info("Created version in Fedora", "url", versionURL)
	return resp.StatusCode, nil
}
