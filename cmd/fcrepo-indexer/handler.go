package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/islandora/scyllaridae/pkg/api"
)

// FcrepoHandler implements api.MessageHandler for the fcrepo indexer service.
type FcrepoHandler struct {
	Config       *FcrepoIndexerConfig
	FedoraClient *FedoraClient
}

// Handle processes an incoming event and dispatches to the appropriate operation.
func (h *FcrepoHandler) Handle(payload api.Payload, auth string) (int, []byte, string, error) {
	token := auth
	if token == "" {
		token = payload.Authorization
	}

	eventType := payload.Type
	summary := payload.Summary
	fedoraBaseURL := payload.Target
	if fedoraBaseURL == "" {
		fedoraBaseURL = h.Config.FedoraURL
	}

	// Extract UUID from object ID (strip urn:uuid: prefix)
	uuid := strings.TrimPrefix(payload.Object.ID, "urn:uuid:")

	slog.Info("Processing event",
		"type", eventType,
		"summary", summary,
		"uuid", uuid,
		"target", fedoraBaseURL,
	)

	// Determine the action based on event type/summary
	action := classifyAction(eventType, summary, payload)

	switch action {
	case actionDelete:
		status, err := deleteNode(h.FedoraClient, uuid, fedoraBaseURL, token)
		if err != nil {
			return status, nil, "", err
		}
		return http.StatusOK, []byte("deleted"), "text/plain", nil

	case actionNodeIndex:
		jsonldURL := findURL(payload.Object.URL, "application/ld+json", "")
		if jsonldURL == "" {
			return http.StatusBadRequest, nil, "", fmt.Errorf("no JSON-LD URL found in event")
		}

		status, err := saveNode(h.FedoraClient, h.Config, uuid, jsonldURL, fedoraBaseURL, token)
		if err != nil {
			return status, nil, "", err
		}

		// Create version if requested
		if payload.Object.IsNewVersion {
			vStatus, vErr := createVersion(h.FedoraClient, uuid, fedoraBaseURL, token)
			if vErr != nil {
				slog.Warn("Failed to create version", "err", vErr, "status", vStatus)
			}
		}

		return http.StatusOK, []byte("indexed"), "text/plain", nil

	case actionMediaIndex:
		sourceField := payload.Attachment.Content.SourceField
		if sourceField == "" {
			return http.StatusBadRequest, nil, "", fmt.Errorf("no source_field in event attachment")
		}
		jsonURL := findURL(payload.Object.URL, "application/json", "")
		if jsonURL == "" {
			return http.StatusBadRequest, nil, "", fmt.Errorf("no JSON URL found in event")
		}

		status, err := saveMedia(h.FedoraClient, h.Config, sourceField, jsonURL, fedoraBaseURL, token)
		if err != nil {
			return status, nil, "", err
		}

		// Create media version if requested
		if payload.Object.IsNewVersion {
			vStatus, vErr := createMediaVersion(h.FedoraClient, h.Config, sourceField, jsonURL, fedoraBaseURL, token)
			if vErr != nil {
				slog.Warn("Failed to create media version", "err", vErr, "status", vStatus)
			}
		}

		return http.StatusOK, []byte("media indexed"), "text/plain", nil

	case actionExternalIndex:
		canonicalURL := findURL(payload.Object.URL, "", "canonical")
		if canonicalURL == "" {
			return http.StatusBadRequest, nil, "", fmt.Errorf("no canonical URL found in event")
		}

		status, err := saveExternal(h.FedoraClient, uuid, canonicalURL, fedoraBaseURL, token)
		if err != nil {
			return status, nil, "", err
		}

		return http.StatusOK, []byte("external indexed"), "text/plain", nil

	default:
		slog.Warn("Unknown action for event", "type", eventType, "summary", summary)
		return http.StatusOK, []byte("no action"), "text/plain", nil
	}
}

type eventAction int

const (
	actionUnknown eventAction = iota
	actionDelete
	actionNodeIndex
	actionMediaIndex
	actionExternalIndex
)

// classifyAction determines the action to take based on the event type and summary.
func classifyAction(eventType, summary string, payload api.Payload) eventAction {
	lowerType := strings.ToLower(eventType)
	lowerSummary := strings.ToLower(summary)

	// Delete events
	if strings.Contains(lowerType, "delete") || strings.Contains(lowerSummary, "delete") {
		return actionDelete
	}

	// External content events
	if strings.Contains(lowerSummary, "external") {
		return actionExternalIndex
	}

	// Media events: have a source field in attachment
	if payload.Attachment.Content.SourceField != "" {
		// Check if there's a JSON URL (media index) vs JSON-LD URL (node index)
		jsonURL := findURL(payload.Object.URL, "application/json", "")
		if jsonURL != "" {
			return actionMediaIndex
		}
	}

	// Node index events: have a JSON-LD URL
	jsonldURL := findURL(payload.Object.URL, "application/ld+json", "")
	if jsonldURL != "" {
		return actionNodeIndex
	}

	return actionUnknown
}

// findURL searches the URL array for a matching mediaType or rel.
func findURL(urls []api.Link, mediaType, rel string) string {
	for _, u := range urls {
		if mediaType != "" && u.MediaType == mediaType {
			return u.Href
		}
		if rel != "" && u.Rel == rel {
			return u.Href
		}
	}
	return ""
}
