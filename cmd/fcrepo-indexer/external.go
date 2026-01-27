package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// saveExternal handles external content by HEADing the external URL to get
// its Content-Type, then saving it to Fedora using an ExternalContent link header.
func saveExternal(fedoraClient *FedoraClient, uuid, externalURL, fedoraBaseURL, token string) (int, error) {
	path, err := GetFedoraPath(uuid)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("invalid UUID %s: %w", uuid, err)
	}
	fedoraBaseURL = strings.TrimRight(fedoraBaseURL, "/")
	fedoraURL := fedoraBaseURL + "/" + path

	// HEAD the external URL to get Content-Type
	mimetype, err := headExternalURL(externalURL, token)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error checking external URL %s: %w", externalURL, err)
	}

	// PUT to Fedora with ExternalContent link header
	externalRel := "http://fedora.info/definitions/fcrepo#ExternalContent"
	link := fmt.Sprintf(`<%s>; rel="%s"; handling="redirect"; type="%s"`, externalURL, externalRel, mimetype)

	headers := map[string]string{
		"Link": link,
	}

	resp, err := fedoraClient.Put(fedoraURL, token, nil, headers)
	if err != nil {
		return http.StatusBadGateway, fmt.Errorf("error saving external content to Fedora %s: %w", fedoraURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, fmt.Errorf("PUT %s returned %d: %s", fedoraURL, resp.StatusCode, string(body))
	}

	slog.Info("Saved external content to Fedora", "fedoraURL", fedoraURL, "externalURL", externalURL)
	return resp.StatusCode, nil
}

// headExternalURL performs a HEAD on an external URL. Tries with auth first,
// falls back to unauthenticated if the authenticated request fails.
func headExternalURL(url, token string) (string, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", fmt.Errorf("error building HEAD request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error performing HEAD: %w", err)
	}
	resp.Body.Close()

	// If authenticated request fails, try without auth
	if resp.StatusCode >= 400 && token != "" {
		slog.Debug("Authenticated HEAD failed, retrying without auth", "status", resp.StatusCode, "url", url)
		req, err = http.NewRequest("HEAD", url, nil)
		if err != nil {
			return "", fmt.Errorf("error building HEAD request: %w", err)
		}
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("error performing HEAD without auth: %w", err)
		}
		resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HEAD %s returned %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	// Strip parameters (e.g., charset)
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	return contentType, nil
}
