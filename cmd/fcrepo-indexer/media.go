package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// mediaURLs holds the resolved URLs needed for media indexing.
type mediaURLs struct {
	Drupal string
	Fedora string
	Jsonld string
}

// saveMedia indexes a media file by resolving its Drupal/Fedora URLs
// and delegating to updateNode.
func saveMedia(fedoraClient *FedoraClient, cfg *FcrepoIndexerConfig, sourceField, jsonURL, fedoraBaseURL, token string) (int, error) {
	urls, err := getMediaURLs(fedoraClient, sourceField, jsonURL, fedoraBaseURL, token)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error resolving media URLs: %w", err)
	}

	return updateNode(fedoraClient, cfg, urls.Jsonld, urls.Fedora, token)
}

// getMediaURLs resolves the Drupal, Fedora, and JSON-LD URLs for a media resource.
func getMediaURLs(fedoraClient *FedoraClient, sourceField, jsonURL, fedoraBaseURL, token string) (*mediaURLs, error) {
	// GET the media JSON from Drupal
	req, err := http.NewRequest("GET", jsonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request for %s: %w", jsonURL, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	drupalResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", jsonURL, err)
	}
	defer drupalResp.Body.Close()

	if drupalResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", jsonURL, drupalResp.StatusCode)
	}

	// Parse Link header for alternate (application/ld+json) URL
	jsonldURL := ParseLinkHeader(drupalResp, "alternate", "application/ld+json")
	if jsonldURL == "" {
		return nil, fmt.Errorf("cannot parse 'alternate' link header from response to GET %s", jsonURL)
	}

	// Parse Link header for describes URL
	drupalURL := ParseLinkHeader(drupalResp, "describes", "")
	if drupalURL == "" {
		return nil, fmt.Errorf("cannot parse 'describes' link header from response to GET %s", jsonURL)
	}

	// Read body and parse file UUID from source field
	body, err := io.ReadAll(drupalResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var mediaJSON map[string]json.RawMessage
	if err := json.Unmarshal(body, &mediaJSON); err != nil {
		return nil, fmt.Errorf("error parsing media JSON: %w", err)
	}

	fieldRaw, ok := mediaJSON[sourceField]
	if !ok {
		return nil, fmt.Errorf("source field %s not found in media JSON from %s", sourceField, jsonURL)
	}

	var fieldEntries []map[string]interface{}
	if err := json.Unmarshal(fieldRaw, &fieldEntries); err != nil {
		return nil, fmt.Errorf("error parsing source field %s: %w", sourceField, err)
	}

	if len(fieldEntries) == 0 {
		return nil, fmt.Errorf("source field %s is empty in media JSON from %s", sourceField, jsonURL)
	}

	fileUUID, ok := fieldEntries[0]["target_uuid"].(string)
	if !ok || fileUUID == "" {
		return nil, fmt.Errorf("cannot extract target_uuid from source field %s", sourceField)
	}

	// Determine Fedora file path
	fedoraBaseURL = strings.TrimRight(fedoraBaseURL, "/")
	var fedoraFilePath string

	// Check for flysystem paths
	pieces := strings.SplitN(drupalURL, "_flysystem/fedora/", 2)
	if len(pieces) > 1 {
		fedoraFilePath = pieces[1]
	} else {
		fedoraFilePath, err = GetFedoraPath(fileUUID)
		if err != nil {
			return nil, fmt.Errorf("error mapping file UUID %s: %w", fileUUID, err)
		}
	}

	fedoraFileURL := fedoraBaseURL + "/" + fedoraFilePath

	// HEAD the file in Fedora to get describedby link
	slog.Debug("HEAD Fedora file", "url", fedoraFileURL)
	fedoraResp, err := fedoraClient.HeadNoRedirect(fedoraFileURL, token)
	if err != nil {
		return nil, fmt.Errorf("error checking Fedora file %s: %w", fedoraFileURL, err)
	}
	fedoraResp.Body.Close()

	if fedoraResp.StatusCode != http.StatusOK && fedoraResp.StatusCode != http.StatusTemporaryRedirect {
		return nil, fmt.Errorf("HEAD %s returned %d", fedoraFileURL, fedoraResp.StatusCode)
	}

	fedoraURL := ParseLinkHeader(fedoraResp, "describedby", "")
	if fedoraURL == "" {
		return nil, fmt.Errorf("cannot parse 'describedby' link header from response to HEAD %s", fedoraFileURL)
	}

	return &mediaURLs{
		Drupal: drupalURL,
		Fedora: fedoraURL,
		Jsonld: jsonldURL,
	}, nil
}
