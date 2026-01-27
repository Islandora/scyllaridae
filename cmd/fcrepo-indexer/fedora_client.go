package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// FedoraClient is a thin HTTP client for Fedora REST API operations.
type FedoraClient struct {
	HTTPClient *http.Client
	IsFedora6  bool
}

// NewFedoraClient creates a new FedoraClient.
func NewFedoraClient(isFedora6 bool) *FedoraClient {
	return &FedoraClient{
		HTTPClient: &http.Client{},
		IsFedora6:  isFedora6,
	}
}

// Head performs a HEAD request to Fedora.
func (fc *FedoraClient) Head(url, token string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building HEAD request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return fc.HTTPClient.Do(req)
}

// HeadNoRedirect performs a HEAD request without following redirects.
func (fc *FedoraClient) HeadNoRedirect(url, token string) (*http.Response, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building HEAD request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return client.Do(req)
}

// Get performs a GET request to Fedora with optional headers.
func (fc *FedoraClient) Get(url, token string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building GET request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return fc.HTTPClient.Do(req)
}

// Put performs a PUT request to Fedora.
func (fc *FedoraClient) Put(url, token string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, fmt.Errorf("error building PUT request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	slog.Debug("PUT request", "url", url, "headers", req.Header)
	return fc.HTTPClient.Do(req)
}

// Delete performs a DELETE request to Fedora.
func (fc *FedoraClient) Delete(url, token string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building DELETE request for %s: %w", url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return fc.HTTPClient.Do(req)
}

// ParseLinkHeader extracts the URI from a Link header matching the given rel.
// Optionally filters by type if provided.
func ParseLinkHeader(resp *http.Response, relName string, linkType string) string {
	for _, linkHeader := range resp.Header.Values("Link") {
		for _, link := range strings.Split(linkHeader, ",") {
			link = strings.TrimSpace(link)
			parts := strings.Split(link, ";")
			if len(parts) < 2 {
				continue
			}

			uri := strings.TrimSpace(parts[0])
			if !strings.HasPrefix(uri, "<") || !strings.HasSuffix(uri, ">") {
				continue
			}
			uri = uri[1 : len(uri)-1]

			hasRel := false
			hasType := linkType == ""
			for _, param := range parts[1:] {
				param = strings.TrimSpace(param)
				kv := strings.SplitN(param, "=", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.TrimSpace(kv[0])
				val := strings.Trim(strings.TrimSpace(kv[1]), "\"")

				if key == "rel" && val == relName {
					hasRel = true
				}
				if key == "type" && val == linkType {
					hasType = true
				}
			}

			if hasRel && hasType {
				return uri
			}
		}
	}
	return ""
}
