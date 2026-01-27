package main

import (
	"fmt"
	"strings"
)

// GetFedoraPath converts a UUID to a Fedora pairtree path.
// The first 8 characters of the UUID are split into 4 pairs,
// forming directory segments, followed by the full UUID.
// Example: "9541c0c1-5bee-4973-a93a-69b3c1a1f906" -> "95/41/c0/c1/9541c0c1-5bee-4973-a93a-69b3c1a1f906"
func GetFedoraPath(uuid string) (string, error) {
	if len(uuid) < 8 {
		return "", fmt.Errorf("UUID must be at least 8 characters, got %d", len(uuid))
	}

	prefix := uuid[:8]
	segments := make([]string, 4)
	for i := 0; i < 4; i++ {
		segments[i] = prefix[i*2 : i*2+2]
	}
	return strings.Join(segments, "/") + "/" + uuid, nil
}

// GetDrupalUUID extracts a UUID from a Fedora pairtree path.
// The UUID is the last path segment.
func GetDrupalUUID(fedoraPath string) (string, error) {
	if fedoraPath == "" {
		return "", fmt.Errorf("empty fedora path")
	}
	segments := strings.Split(fedoraPath, "/")
	return segments[len(segments)-1], nil
}
