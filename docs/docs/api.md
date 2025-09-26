# API Reference

Scyllaridae provides a simple HTTP API for processing files through configured commands. The service supports both GET and POST requests for different use cases.

## Base URL

The service runs on port 8080 by default (configurable via `SCYLLARIDAE_PORT`):

```
http://localhost:8080
```

## Endpoints

### Health Check

Check if the service is running and healthy.

**Endpoint:** `GET /healthcheck`

**Response:**

- **200 OK**: Service is healthy
- **Body**: `OK`

**Example:**

```bash
curl -f http://localhost:8080/healthcheck
```

### File Processing

Process files using the configured commands. Supports both GET (file URL) and POST (file upload) methods.

**Endpoint:** `GET|POST /`

#### GET Method - Process Remote File

Process a file from a remote URL. Used primarily by Islandora/Alpaca for derivative generation.

**Headers:**

| Header              | Required    | Description                                      |
| ------------------- | ----------- | ------------------------------------------------ |
| `Authorization`     | Conditional | JWT token (required if JWT verification enabled) |
| `Apix-Ldp-Resource` | Yes         | URL of the source file to process                |
| `Accept`            | Optional    | Desired output MIME type                         |
| `X-Islandora-Args`  | Optional    | Additional arguments for the command             |
| `X-Islandora-Event` | Optional    | Event type identifier                            |

**Example:**

```bash
curl -X GET \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Apix-Ldp-Resource: https://example.com/files/document.pdf" \
  -H "Accept: image/jpeg" \
  -H "X-Islandora-Args: -quality 80 -resize 300x300" \
  http://localhost:8080/
```

#### POST Method - Upload and Process File

Upload a file directly to the service for processing.

**Headers:**

| Header             | Required    | Description                                      |
| ------------------ | ----------- | ------------------------------------------------ |
| `Authorization`    | Conditional | JWT token (required if JWT verification enabled) |
| `Content-Type`     | Yes         | MIME type of the uploaded file                   |
| `Accept`           | Optional    | Desired output MIME type                         |
| `X-Islandora-Args` | Optional    | Additional arguments for the command             |

**Body:** Binary file data

**Example:**

```bash
curl -X POST \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIs..." \
  -H "Content-Type: image/png" \
  -H "Accept: image/jpeg" \
  -H "X-Islandora-Args: -quality 90" \
  --data-binary "@image.png" \
  http://localhost:8080/
```

## Request Flow

### 1. Authentication (if enabled)

When JWT verification is enabled (`jwksUri` configured), the service:

1. Validates the `Authorization` header format (`Bearer <token>`)
2. Verifies the JWT signature against the JWKS endpoint
3. Checks token expiration and validity
4. Rejects requests with missing or invalid tokens

### 2. File Acquisition

**GET requests:**

- Downloads file from `Apix-Ldp-Resource` URL
- Forwards `Authorization` header if `forwardAuth: true`
- Streams file directly to command

**POST requests:**

- Reads file data from request body
- Uses `Content-Type` header for MIME type detection

### 3. MIME Type Validation

- Compares source MIME type against `allowedMimeTypes`
- Supports exact matches, wildcards (`*`), and type families (`image/*`)
- Returns 400 Bad Request for disallowed types

### 4. Command Selection

- Selects command from `cmdByMimeType` configuration
- Priority: exact type → type family → default
- Uses destination MIME type if `mimeTypeFromDestination: true`

### 5. Command Execution

- Pipes file data to command stdin
- Substitutes special variables in command arguments
- Streams command stdout back as HTTP response
- Captures stderr for logging

## Response Codes

| Code | Description           | Common Causes                                             |
| ---- | --------------------- | --------------------------------------------------------- |
| 200  | Success               | Command executed successfully                             |
| 400  | Bad Request           | Invalid headers, unsupported MIME type, malformed request |
| 401  | Unauthorized          | Invalid JWT token                                         |
| 404  | Not Found             | Invalid endpoint                                          |
| 405  | Method Not Allowed    | Unsupported HTTP method                                   |
| 424  | Failed Dependency     | Unable to fetch source file                               |
| 500  | Internal Server Error | Command execution failed, configuration error             |

## Response Headers

The service may set the following response headers:

| Header         | Description                       |
| -------------- | --------------------------------- |
| `Content-Type` | MIME type of the processed output |
| `Connection`   | Connection handling directive     |

## Error Responses

Error responses include a plain text error message:

```
HTTP/1.1 400 Bad Request
Content-Type: text/plain

Missing Authorization header
```

## Authentication Details

### JWT Token Format

When JWT verification is enabled, tokens must:

- Use the `Bearer` scheme: `Authorization: Bearer <token>`
- Be valid RS256-signed JWTs
- Have current timestamps (not expired)
- Be verifiable against the configured JWKS endpoint

### Token Forwarding

When processing files, the JWT token is made available to commands via the `SCYLLARIDAE_AUTH` environment variable (if `forwardAuth: true`). This allows the command ran by scyllaridae to utilize the JWT if needed in its command.

## Special Headers

### X-Islandora-Args

Additional command-line arguments passed through the `%args` variable:

```bash
# Header: X-Islandora-Args: -quality 80 -resize 300x300
# Configuration: args: ["-", "%args", "output.jpg"]
# Actual command: convert - -quality 80 -resize 300x300 output.jpg
```

### Accept Header

Specifies the desired output MIME type, available as `%destination-mime-*` variables:

```bash
# Header: Accept: image/jpeg
# Variable %destination-mime-ext: jpg
# Variable %destination-mime-pandoc: jpeg
```

## Integration Examples

### Islandora/Alpaca Integration

Typical request from Alpaca for derivative generation:

```bash
curl -X GET \
  -H "Authorization: Bearer ${JWT_TOKEN}" \
  -H "Apix-Ldp-Resource: https://islandora.dev/sites/default/files/fedora/2024-03/image.tiff" \
  -H "Accept: image/jpeg" \
  -H "X-Islandora-Args: -quality 80 -thumbnail 300x300>" \
  http://scyllaridae-thumbnails:8080/
```

### Direct File Upload

Process a local file directly:

```bash
curl -X POST \
  -H "Content-Type: application/pdf" \
  -H "Accept: text/plain" \
  --data-binary "@document.pdf" \
  http://localhost:8080/ \
  > extracted-text.txt
```

### Batch Processing Script

```bash
#!/bin/bash
for file in *.pdf; do
  echo "Processing $file..."
  curl -X POST \
    -H "Content-Type: application/pdf" \
    -H "Accept: image/jpeg" \
    --data-binary "@$file" \
    http://localhost:8080/ \
    > "${file%.pdf}.jpg"
done
```

## Rate Limiting and Concurrency

Scyllaridae processes requests concurrently but does not implement built-in rate limiting. For production deployments:

- Use a reverse proxy (nginx, traefik) for rate limiting
- Configure appropriate resource limits in Docker/Kubernetes
- Monitor memory usage for large file processing


## Monitoring and Observability

### Logging

The service logs request details including:

- HTTP method and path
- Response status code
- Request duration
- Client IP and User-Agent
- Command executed
- Message ID (from events)

## Testing the API

### Basic Connectivity Test

```bash
# Health check
curl -f http://localhost:8080/healthcheck || echo "Service unavailable"
```

### File Processing Test

```bash
# Create test file
echo "Hello, World!" > test.txt

# Process with echo command
curl -X POST \
  -H "Content-Type: text/plain" \
  --data-binary "@test.txt" \
  http://localhost:8080/
```

### JWT Authentication Test

```bash
# With valid token
curl -X POST \
  -H "Authorization: Bearer ${VALID_JWT}" \
  -H "Content-Type: text/plain" \
  --data-binary "@test.txt" \
  http://localhost:8080/

# Without token (should fail if JWT enabled)
curl -X POST \
  -H "Content-Type: text/plain" \
  --data-binary "@test.txt" \
  http://localhost:8080/
```
