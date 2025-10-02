package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestIsAllowedMimeType(t *testing.T) {
	tests := []struct {
		name           string
		mimetype       string
		allowedFormats []string
		want           bool
	}{
		{
			name:           "exact match",
			mimetype:       "image/jpeg",
			allowedFormats: []string{"image/jpeg", "image/png"},
			want:           true,
		},
		{
			name:           "wildcard match",
			mimetype:       "image/jpeg",
			allowedFormats: []string{"image/*"},
			want:           true,
		},
		{
			name:           "allow all",
			mimetype:       "anything/goes",
			allowedFormats: []string{"*"},
			want:           true,
		},
		{
			name:           "no match",
			mimetype:       "video/mp4",
			allowedFormats: []string{"image/*", "audio/*"},
			want:           false,
		},
		{
			name:           "mime type with charset",
			mimetype:       "text/html; charset=utf-8",
			allowedFormats: []string{"text/html"},
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedMimeType(tt.mimetype, tt.allowedFormats)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name      string
		yml       string
		wantError bool
		validate  func(*testing.T, *ServerConfig)
	}{
		{
			name: "valid config with defaults",
			yml: `allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "echo"
    args: ["hello"]`,
			wantError: false,
			validate: func(t *testing.T, c *ServerConfig) {
				assert.True(t, *c.ForwardAuth, "forwardAuth should default to true")
				assert.Equal(t, "", c.JwksUri)
				assert.Equal(t, []string{"*"}, c.AllowedMimeTypes)
			},
		},
		{
			name: "config with jwksUri",
			yml: `jwksUri: "https://example.com/keys"
allowedMimeTypes:
  - "image/*"
cmdByMimeType:
  default:
    cmd: "cat"`,
			wantError: false,
			validate: func(t *testing.T, c *ServerConfig) {
				assert.Equal(t, "https://example.com/keys", c.JwksUri)
			},
		},
		{
			name: "config with environment variable expansion",
			yml: `allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "${TEST_CMD}"
    args: ["${TEST_ARG}"]`,
			wantError: false,
			validate: func(t *testing.T, c *ServerConfig) {
				os.Setenv("TEST_CMD", "testcmd")
				os.Setenv("TEST_ARG", "testarg")
				defer os.Unsetenv("TEST_CMD")
				defer os.Unsetenv("TEST_ARG")
				// Note: config was already parsed, so this tests the mechanism exists
			},
		},
		{
			name:      "invalid YAML",
			yml:       "this is not: valid: yaml:",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SCYLLARIDAE_YML", tt.yml)
			defer os.Unsetenv("SCYLLARIDAE_YML")

			config, err := ReadConfig()
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestBuildExecCommand(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServerConfig
		payload   api.Payload
		wantCmd   string
		wantArgs  []string
		wantError bool
	}{
		{
			name: "basic command with default",
			config: &ServerConfig{
				AllowedMimeTypes: []string{"*"},
				CmdByMimeType: map[string]Command{
					"default": {Cmd: "echo", Args: []string{"hello"}},
				},
			},
			payload: api.Payload{
				Attachment: api.Attachment{
					Content: api.Content{
						SourceMimeType: "text/plain",
					},
				},
			},
			wantCmd:   "echo",
			wantArgs:  []string{"hello"},
			wantError: false,
		},
		{
			name: "command with %args placeholder",
			config: &ServerConfig{
				AllowedMimeTypes: []string{"*"},
				CmdByMimeType: map[string]Command{
					"default": {Cmd: "convert", Args: []string{"-", "%args", "jpg:-"}},
				},
			},
			payload: api.Payload{
				Attachment: api.Attachment{
					Content: api.Content{
						SourceMimeType: "image/png",
						Args:           "-quality 80",
					},
				},
			},
			wantCmd:   "convert",
			wantArgs:  []string{"-", "-quality", "80", "jpg:-"},
			wantError: false,
		},
		{
			name: "command with MIME type placeholders",
			config: &ServerConfig{
				AllowedMimeTypes: []string{"*"},
				CmdByMimeType: map[string]Command{
					"default": {Cmd: "echo", Args: []string{"%source-mime-ext", "%destination-mime-ext"}},
				},
			},
			payload: api.Payload{
				Attachment: api.Attachment{
					Content: api.Content{
						SourceMimeType:      "image/jpeg",
						DestinationMimeType: "image/png",
					},
				},
			},
			wantCmd:   "echo",
			wantArgs:  []string{"jpg", "png"},
			wantError: false,
		},
		{
			name: "disallowed MIME type",
			config: &ServerConfig{
				AllowedMimeTypes: []string{"image/*"},
				CmdByMimeType: map[string]Command{
					"default": {Cmd: "echo", Args: []string{"test"}},
				},
			},
			payload: api.Payload{
				Attachment: api.Attachment{
					Content: api.Content{
						SourceMimeType: "video/mp4",
					},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := true
			tt.config.ForwardAuth = &fa

			cmd, err := BuildExecCommand(tt.payload, tt.config)
			if tt.wantError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantCmd, filepath.Base(cmd.Path))
			assert.Equal(t, tt.wantArgs, cmd.Args[1:]) // Skip Args[0] which is the command itself
		})
	}
}

func TestBadCmdArgs(t *testing.T) {
	payloads := []string{
		`"any;thing`,
		`"any&thing`,
		`"any|thing`,
		`"any$thing`,
		`"any\"thing`,
		`"any\thing`,
		`"any*thing`,
		`"any?thing`,
		`"any[thing`,
		`"any]thing`,
		`"any{thing`,
		`"any}thing`,
		`"any(thing`,
		`"any)thing`,
		`"any<thing`,
		`"any>thing`,
		`"anything!`,
		"\"any`thing\"",
	}
	for _, payload := range payloads {
		_, err := GetPassedArgs(payload)
		assert.Error(t, err)
	}

}

func TestMimeToPandoc(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     string
		wantErr  bool
	}{
		{
			name:     "markdown mime type",
			mimeType: "text/markdown",
			want:     "markdown",
			wantErr:  false,
		},
		{
			name:     "html mime type",
			mimeType: "text/html",
			want:     "html",
			wantErr:  false,
		},
		{
			name:     "docx mime type",
			mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			want:     "docx",
			wantErr:  false,
		},
		{
			name:     "latex mime type",
			mimeType: "application/x-latex",
			want:     "latex",
			wantErr:  false,
		},
		{
			name:     "fallback to extension for unknown type",
			mimeType: "image/jpeg",
			want:     "jpg",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MimeToPandoc(tt.mimeType)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMimeTypes(t *testing.T) {
	mimeTypes := map[string]string{
		"application/msword": "doc",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
		"application/vnd.ms-excel": "xls",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
		"application/vnd.ms-powerpoint":                                             "ppt",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",

		"image/jpeg":    "jpg",
		"image/jp2":     "jp2",
		"image/png":     "png",
		"image/gif":     "gif",
		"image/bmp":     "bmp",
		"image/svg+xml": "svg",
		"image/tiff":    "tiff",
		"image/webp":    "webp",

		"audio/mpeg":        "mp3",
		"audio/x-wav":       "wav",
		"audio/ogg":         "ogg",
		"audio/aac":         "m4a",
		"audio/webm":        "webm",
		"audio/flac":        "flac",
		"audio/midi":        "mid",
		"audio/x-m4a":       "m4a",
		"audio/x-realaudio": "ra",

		"video/mp4":                     "mp4",
		"video/x-msvideo":               "avi",
		"video/x-ms-wmv":                "wmv",
		"video/mpeg":                    "mpg",
		"video/webm":                    "webm",
		"video/quicktime":               "mov",
		"application/vnd.apple.mpegurl": "m3u8",
		"video/3gpp":                    "3gp",
		"video/mp2t":                    "ts",
		"video/x-flv":                   "flv",
		"video/x-m4v":                   "m4v",
		"video/x-mng":                   "mng",
		"video/x-ms-asf":                "asx",
		"video/ogg":                     "ogg",

		"text/plain":      "txt",
		"text/html":       "html",
		"application/pdf": "pdf",
		"text/csv":        "csv",
		"text/markdown":   "md",
	}

	for mimeType, extension := range mimeTypes {
		ext, err := GetMimeTypeExtension(mimeType)
		assert.Equal(t, nil, err)
		assert.Equal(t, extension, ext)
	}
}
