package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	}

	for mimeType, extension := range mimeTypes {
		ext, err := GetMimeTypeExtension(mimeType)
		assert.Equal(t, nil, err)
		assert.Equal(t, extension, ext)
	}
}
