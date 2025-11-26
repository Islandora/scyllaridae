# Configuration Reference

Scyllaridae is configured using a YAML file (`scyllaridae.yml`) that defines how the service processes different file types and handles authentication.

## Configuration File Location

The configuration file location is specified by the `SCYLLARIDAE_YML_PATH` environment variable:

```bash
export SCYLLARIDAE_YML_PATH="/app/scyllaridae.yml"
```

Alternatively, you can provide the configuration directly via the `SCYLLARIDAE_YML` environment variable:

```bash
export SCYLLARIDAE_YML="allowedMimeTypes: ['*']
cmdByMimeType:
  default:
    cmd: echo
    args: ['Hello World']"
```

## Configuration Schema

### Top-Level Options

| Option                    | Type             | Default | Description                                                            |
| ------------------------- | ---------------- | ------- | ---------------------------------------------------------------------- |
| `forwardAuth`             | boolean          | `true`  | Whether to forward the Authorization header when fetching source files |
| `jwksUri`                 | string           | `""`    | URI for JWT verification. If empty, JWT verification is skipped        |
| `allowedMimeTypes`        | array of strings | `[]`    | MIME types allowed for processing                                      |
| `cmdByMimeType`           | map              | `{}`    | Commands to execute for different MIME types                           |
| `mimeTypeFromDestination` | boolean          | `false` | Use destination MIME type instead of source for command selection      |

### Authentication Configuration

#### JWT Verification

JWT verification is controlled by the `jwksUri` configuration option:

```yaml
# Enable JWT verification
jwksUri: "https://your-domain.com/oauth/discovery/keys"
# Disable JWT verification (default)
# jwksUri: ""
```

When JWT verification is enabled:

- The service validates incoming JWT tokens against the provided JWKS endpoint
- Invalid or missing tokens result in HTTP 401/400 responses

#### Authorization Header Forwarding

Control whether the Authorization header is forwarded when fetching source files:

```yaml
# Forward Authorization header (default)
forwardAuth: true

# Don't forward Authorization header
forwardAuth: false
```

### MIME Type Configuration

#### Allowed MIME Types

Specify which MIME types the service will process:

```yaml
# Allow all MIME types
allowedMimeTypes:
  - "*"

# Allow specific types
allowedMimeTypes:
  - "image/jpeg"
  - "image/png"
  - "application/pdf"

# Allow type families
allowedMimeTypes:
  - "image/*"
  - "video/*"
```

#### MIME Type Source

By default, commands are selected based on the source file's MIME type. You can change this to use the destination MIME type:

```yaml
# Use destination MIME type for command selection
mimeTypeFromDestination: true
```

### Command Configuration

Commands are defined in the `cmdByMimeType` section, which maps MIME types to executable commands.

#### Basic Command Structure

```yaml
cmdByMimeType:
  "image/jpeg":
    cmd: "convert"
    args:
      - "-"
      - "-quality"
      - "80"
      - "jpg:-"
  default:
    cmd: "cat"
    args: []
```

#### Command Security Options

Each command can optionally specify `allowInsecureArgs` to control how arguments from the `X-Islandora-Args` header are validated:

```yaml
cmdByMimeType:
  default:
    cmd: "bash"
    args:
      - "-c"
      - "%args"
    allowInsecureArgs: true  # DANGEROUS: Disables argument validation
```

**⚠️ Security Warning**: When `allowInsecureArgs: false` (the default), arguments from `X-Islandora-Args` are validated against a whitelist regex (`^[a-zA-Z0-9._\-:\/@ =]+$`) to prevent command injection attacks. Setting `allowInsecureArgs: true` disables this validation and allows any characters, which can be dangerous if the header comes from untrusted sources.

Only enable `allowInsecureArgs: true` if:
- You have strict authentication/authorization controls in place
- You trust all sources that can set the `X-Islandora-Args` header
- You need to pass special shell characters (`;`, `|`, `$`, `*`, etc.) to your commands

#### Command Selection

Commands are selected using this priority:

1. Exact MIME type match (e.g., `"image/jpeg"`)
2. MIME type family match (e.g., `"image/*"`)
3. Default command (`"default"`)

#### Special Argument Variables

Scyllaridae provides special variables that can be used in command arguments:

| Variable                   | Description                              | Example Value                     |
| -------------------------- | ---------------------------------------- | --------------------------------- |
| `%args`                    | Arguments from `X-Islandora-Args` header | `-ss 00:00:03.000 -frames 1`      |
| `%source-mime-ext`         | Source file extension                    | `pdf`                             |
| `%destination-mime-ext`    | Destination file extension               | `jpg`                             |
| `%destination-mime-ext:-`  | Destination extension with `:-` suffix   | `jpg:-`                           |
| `%source-mime-pandoc`      | Source MIME type in Pandoc format        | `markdown`                        |
| `%destination-mime-pandoc` | Destination MIME type in Pandoc format   | `html`                            |
| `%target`                  | Target value from event                  | `thumbnail`                       |
| `%source-uri`              | Source file URI                          | `https://example.com/file.pdf`    |
| `%file-upload-uri`         | File upload URI                          | `private://derivatives/thumb.jpg` |
| `%destination-uri`         | Destination URI                          | `https://example.com/media/1`     |
| `%canonical`               | Canonical URL from event                 | `https://example.com/node/1`      |

### Environment Variable Expansion

Configuration values support environment variable expansion using `${VAR}` syntax:

```yaml
jwksUri: "${JWKS_ENDPOINT}/keys"
cmdByMimeType:
  default:
    cmd: "${PROCESSING_COMMAND}"
    args:
      - "--output-dir"
      - "${OUTPUT_DIRECTORY}"
```

## Configuration Examples

### Simple Pass-Through Service

```yaml
allowedMimeTypes:
  - "*"
cmdByMimeType:
  default:
    cmd: "cat"
    args: []
```

### Image Processing Service

```yaml
forwardAuth: false
jwksUri: "https://islandora.dev/oauth/discovery/keys"
allowedMimeTypes:
  - "image/*"
cmdByMimeType:
  "image/tiff":
    cmd: "convert"
    args:
      - "tiff:-[0]"
      - "%args"
      - "jpg:-"
  default:
    cmd: "convert"
    args:
      - "-"
      - "%args"
      - "jpg:-"
```

### Document Conversion Service

```yaml
allowedMimeTypes:
  - "application/pdf"
  - "application/msword"
  - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
mimeTypeFromDestination: true
cmdByMimeType:
  "application/pdf":
    cmd: "libreoffice"
    args:
      - "--headless"
      - "--convert-to"
      - "pdf"
      - "--outdir"
      - "/tmp"
      - "-"
  default:
    cmd: "pandoc"
    args:
      - "-f"
      - "%source-mime-pandoc"
      - "-t"
      - "%destination-mime-pandoc"
      - "-o"
      - "-"
```

### Using Bash Wrapper Scripts

If your command doesn't support reading from stdin and writing to stdout, you can use a bash wrapper script:

```yaml
allowedMimeTypes:
  - "application/msword"
  - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
cmdByMimeType:
  default:
    cmd: "/app/convert.sh"
    args: []
```

Example `/app/convert.sh`:

```bash
#!/usr/bin/env bash
set -eou pipefail

# Read stdin to a temp file
input_temp=$(mktemp /tmp/input-XXXXXX)
cat > "$input_temp"

# Process the file
# Redirect stderr/stdout to /dev/null to avoid polluting the output
libreoffice --headless --convert-to pdf "$input_temp" > /dev/null 2>&1

# Write output to stdout
output="/app/$(basename "$input_temp").pdf"
cat "$output"

# Cleanup
rm "$input_temp" "$output"
```

## Configuration Validation

Scyllaridae validates the configuration file on startup. Common validation errors include:

- **Missing required fields**: `cmdByMimeType` is required
- **Invalid MIME types**: MIME type strings must follow the `type/subtype` format
- **Command not found**: Specified commands must be available in the container
- **Invalid JWKS URI**: Must be a valid HTTP/HTTPS URL when provided

## Configuration Best Practices

1. **Start simple**: Begin with a basic configuration and add complexity as needed
2. **Use specific MIME types**: Prefer specific types over wildcards for better control
3. **Test commands locally**: Verify commands work with your test files before deployment
4. **Validate JWKS connectivity**: Ensure JWKS URI is accessible from your deployment environment
5. **Use environment variables**: Parameterize environment-specific values
6. **Document custom arguments**: Comment your `%args` usage for future reference

## Troubleshooting Configuration

See the [Troubleshooting Guide](troubleshooting.md) for help with configuration issues.
