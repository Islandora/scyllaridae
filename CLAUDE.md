# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 📚 Critical Documentation References
- **Project Overview**: `./docs/docs/index.md`
- **Go Conventions**: `./docs/GO_CONVENTIONS.md`
- **Project Architecture**: `./docs/docs/api.md`
- **YAML Specification**: `./docs/docs/configuration.md`
- **Islandora Event Integration**: `./docs/docs/derivatives/events.md`

## Key Commands

### Building and Testing
```bash
# Build binary
make build

# Run tests with race detection
make test

# Run specific test
go test -v -race ./internal/config -run TestSpecificFunction

# Lint code (must pass before committing)
make lint
```

### Docker
```bash
# Build Docker image
make docker

# Build and run documentation site
make docs
```

### Running Locally
```bash
# Set required environment variables
export SCYLLARIDAE_YML_PATH="./scyllaridae.yml"
export SCYLLARIDAE_PORT="8080"
export SCYLLARIDAE_LOG_LEVEL="DEBUG"

# Run the service
./scyllaridae
```

## Integration with Islandora

This service implements the Islandora/Alpaca microservice pattern:
- Accepts events via `X-Islandora-Event` header (base64-encoded JSON)
- Supports `Apix-Ldp-Resource` header for file URLs
- Uses `Accept` header for destination MIME type
- Healthcheck endpoint at `/healthcheck` returns 200 OK
