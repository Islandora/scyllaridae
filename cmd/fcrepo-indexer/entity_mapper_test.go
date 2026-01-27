package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFedoraPath(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard UUID",
			uuid:     "9541c0c1-5bee-4973-a93a-69b3c1a1f906",
			expected: "95/41/c0/c1/9541c0c1-5bee-4973-a93a-69b3c1a1f906",
		},
		{
			name:     "another UUID",
			uuid:     "abcdef01-2345-6789-abcd-ef0123456789",
			expected: "ab/cd/ef/01/abcdef01-2345-6789-abcd-ef0123456789",
		},
		{
			name:    "UUID too short",
			uuid:    "abc",
			wantErr: true,
		},
		{
			name:    "empty UUID",
			uuid:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetFedoraPath(tt.uuid)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetDrupalUUID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "standard path",
			path:     "95/41/c0/c1/9541c0c1-5bee-4973-a93a-69b3c1a1f906",
			expected: "9541c0c1-5bee-4973-a93a-69b3c1a1f906",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetDrupalUUID(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
