package config

import (
	"fmt"
	"mime"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

// ServerConfig defines server-specific configurations.
//
// swagger:model ServerConfig
type ServerConfig struct {
	// Label of the server configuration used for identification.
	//
	// required: true
	Label string `yaml:"label"`

	// Label of the server configuration used for identification.
	//
	// required: false
	QueueName string `yaml:"queueName"`

	// HTTP method used for sending data to the destination server.
	//
	// required: false
	DestinationHTTPMethod string `yaml:"destinationHttpMethod"`

	// Header name for the file resource.
	//
	// required: false
	FileHeader string `yaml:"fileHeader,omitempty"`

	// Header name for additional arguments passed to the command.
	//
	// required: false
	ArgHeader string `yaml:"argHeader,omitempty"`

	// Indicates whether the authentication header should be forwarded.
	//
	// required: false
	ForwardAuth bool `yaml:"forwardAuth,omitempty"`

	// List of MIME types allowed for processing.
	//
	// required: false
	AllowedMimeTypes []string `yaml:"allowedMimeTypes"`

	// Commands and arguments ran by MIME type.
	//
	// required: false
	CmdByMimeType map[string]Command `yaml:"cmdByMimeType"`
}

// Command describes the command and arguments to execute for a specific MIME type.
//
// swagger:model Command
type Command struct {
	// Command to execute.
	//
	// required: true
	Cmd string `yaml:"cmd"`

	// Arguments for the command.
	//
	// required: false
	Args []string `yaml:"args"`
}

func IsAllowedMimeType(mimetype string, allowedFormats []string) bool {
	for _, format := range allowedFormats {
		if format == mimetype {
			return true
		}
		// if the config specified any mimetype is allowed
		if format == "*" {
			return true
		}
		if strings.HasSuffix(format, "/*") {
			// Check wildcard MIME type
			prefix := strings.TrimSuffix(format, "*")
			if strings.HasPrefix(mimetype, prefix) {
				return true
			}
		}
	}
	return false
}

func ReadConfig(yp string) (*ServerConfig, error) {
	var (
		y   []byte
		err error
	)
	yml := os.Getenv("SCYLLARIDAE_YML")
	if yml != "" {
		y = []byte(yml)
	} else {
		y, err = os.ReadFile(yp)
		if err != nil {
			return nil, err
		}
	}

	var c ServerConfig
	err = yaml.Unmarshal(y, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func BuildExecCommand(replacements map[string]string, c *ServerConfig) (*exec.Cmd, error) {
	if !IsAllowedMimeType(replacements["sourceMimeType"], c.AllowedMimeTypes) {
		return nil, fmt.Errorf("undefined sourceMimeType: %s", replacements["sourceMimeType"])
	}

	cmdConfig, exists := c.CmdByMimeType[replacements["sourceMimeType"]]
	if !exists {
		cmdConfig = c.CmdByMimeType["default"]
	}

	args := []string{}
	for _, a := range cmdConfig.Args {
		// if we have the special value of %args
		// replace it with the args passed by the event
		if a == "%args" && replacements["addtlArgs"] != "" {
			args = append(args, replacements["addtlArgs"])

			// if we have the special value of %source-mime-ext
			// replace it with the source mimetype extension
		} else if a == "%source-mime-ext" {
			a, err := getMimeTypeExtension(replacements["sourceMimeType"])
			if err != nil {
				return nil, fmt.Errorf("unknown mime extension: %s", replacements["sourceMimeType"])
			}

			args = append(args, a)
			// if we have the special value of %destination-mime-ext
			// replace it with the source mimetype extension
		} else if a == "%destination-mime-ext" {
			a, err := getMimeTypeExtension(replacements["destinationMimeType"])
			if err != nil {
				return nil, fmt.Errorf("unknown mime extension: %s", replacements["destinationMimeType"])
			}

			args = append(args, a)

		} else if a == "%target" {
			args = append(args, replacements["target"])
		} else {
			args = append(args, a)
		}
	}

	cmd := exec.Command(cmdConfig.Cmd, args...)
	cmd.Env = os.Environ()

	return cmd, nil
}

func getMimeTypeExtension(mimeType string) (string, error) {
	// since the std mimetype -> extension conversion returns a list
	// we need to override the default extension to use
	// it also is missing some mimetypes
	mimeToExtension := map[string]string{
		"application/msword":            "doc",
		"application/vnd.ms-excel":      "xls",
		"application/vnd.ms-powerpoint": "ppt",

		"image/svg+xml": "svg",
		"image/webp":    "webp",
		"image/jp2":     "jp2",
		"image/bmp":     "bmp",

		"video/mp4":                     "mp4",
		"video/quicktime":               "mov",
		"video/x-ms-asf":                "asx",
		"video/mp2t":                    "ts",
		"video/mpeg":                    "mpg",
		"application/vnd.apple.mpegurl": "m3u8",
		"video/3gpp":                    "3gp",
		"video/x-m4v":                   "m4v",
		"video/x-msvideo":               "avi",
		"video/ogg":                     "ogg",

		"audio/ogg":         "ogg",
		"audio/webm":        "webm",
		"audio/flac":        "flac",
		"audio/aac":         "m4a",
		"audio/mpeg":        "mp3",
		"audio/x-m4a":       "m4a",
		"audio/x-realaudio": "ra",
		"audio/midi":        "mid",
	}
	cleanMimeType := strings.TrimSpace(strings.ToLower(mimeType))
	if ext, ok := mimeToExtension[cleanMimeType]; ok {
		return ext, nil
	}

	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		return "", fmt.Errorf("unknown mime extension: %s", mimeType)
	}

	return strings.TrimPrefix(extensions[len(extensions)-1], "."), nil
}
