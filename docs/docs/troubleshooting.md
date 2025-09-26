# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Scyllaridae microservices.

## General Debugging Steps

### 1. Check Service Health

First, verify the service is running and responding:

```bash
# Check health endpoint
curl -f http://localhost:8080/healthcheck

# Check service logs
docker logs your-microservice-container
kubectl logs deployment/your-microservice
```

### 2. Verify Configuration

Ensure your configuration is valid and properly loaded:

```bash
# Check configuration file syntax
docker exec your-container cat /app/scyllaridae.yml

# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('scyllaridae.yml'))"
```

### 3. Test with Simple Request

Test with a basic request to isolate the issue:

```bash
# Simple POST request
echo "test" | curl -X POST \
  -H "Content-Type: text/plain" \
  --data-binary @- \
  http://localhost:8080/
```

## Logging and Monitoring

### Enable Debug Logging

Set the log level to DEBUG for detailed information:

```bash
# Docker
docker run -e SCYLLARIDAE_LOG_LEVEL=DEBUG your-image

# Kubernetes
kubectl set env deployment/your-microservice SCYLLARIDAE_LOG_LEVEL=DEBUG
```

### Structured Logging

Scyllaridae uses structured logging. Key log fields include:

- `level`: Log level (DEBUG, INFO, ERROR)
- `msg`: Log message
- `path`: Request path
- `status`: HTTP status code
- `duration`: Request duration
- `command`: Executed command
- `msgId`: Message ID from events

## Common Issues and Solutions

### Authentication Issues

#### JWT Token Invalid

**Symptoms:**

- HTTP 401 Unauthorized responses
- Error message: "JWT verification failed"

**Causes:**

- Expired JWT token
- Invalid token signature
- Incorrect JWKS URI
- Network connectivity to JWKS endpoint

**Solutions:**

1. **Verify token validity:**

```bash
  # Decode JWT to check expiration (install jq first)
  echo "your-jwt-token" | cut -d. -f2 | base64 -d | jq .exp
```

2. **Test JWKS connectivity:**

```bash
  curl https://your-domain.com/oauth/discovery/keys
```

3. **Check configuration:**

```yaml
# Ensure correct JWKS URI in scyllaridae.yml
jwksUri: "https://your-domain.com/oauth/discovery/keys"
```

4. **Temporarily disable JWT for testing:**

```yaml
# Remove or comment out jwksUri
# jwksUri: ""
```

#### Missing Authorization Header

**Symptoms:**

- HTTP 400 Bad Request
- Error message: "Missing Authorization header"

**Solutions:**

1. **Add Authorization header:**

```bash
  curl -H "Authorization: Bearer your-jwt-token" ...
```

2. **Check if JWT is required:**

```yaml
# If no authentication needed, ensure jwksUri is empty
# jwksUri: ""
```

### Configuration Issues

#### Invalid YAML Syntax

**Symptoms:**

- Service fails to start
- Error message contains "yaml" or "parsing"

**Solutions:**

1. **Validate YAML syntax:**

```bash
  python -c "import yaml; print(yaml.safe_load(open('scyllaridae.yml')))"
```

2. **Check indentation (use spaces, not tabs):**

```yaml
# Correct indentation
cmdByMimeType:
  default:
    cmd: "echo"
    args:
      - "hello"
```

3. **Quote string values with special characters:**

```yaml
jwksUri: "https://example.com/keys" # Quote URLs
```

#### Command Not Found

**Symptoms:**

- HTTP 500 Internal Server Error
- Log message: "executable file not found"

**Solutions:**

1. **Verify command availability:**

```bash
  docker exec your-container which your-command
```

2. **Install missing dependencies in Dockerfile:**

```dockerfile
  FROM ghcr.io/lehigh-university-libraries/lehighlts/scyllaridae:main
  RUN apk add --no-cache your-required-package
```

3. **Use full command paths:**

```yaml
cmdByMimeType:
  default:
    cmd: "/usr/bin/convert" # Full path instead of "convert"
```

#### Invalid MIME Type Configuration

**Symptoms:**

- HTTP 400 Bad Request
- Error message: "undefined mimeType to build command"

**Solutions:**

1. **Check allowed MIME types:**

```yaml
allowedMimeTypes:
  - "*" # Allow all, or specify exact types
  - "image/*" # Allow type families
```

2. **Add missing MIME types:**

```yaml
allowedMimeTypes:
  - "application/pdf"
  - "image/jpeg"
  - "text/plain"
```

3. **Check Content-Type header:**

```bash
  curl -H "Content-Type: image/jpeg" ...  # Must match allowed types
```

### Network and Connectivity Issues

#### Cannot Fetch Source File

**Symptoms:**

- HTTP 424 Failed Dependency
- Log message: "Error fetching source file contents"

**Causes:**

- Source URL unreachable
- Network connectivity issues
- SSL/TLS certificate problems
- Authorization issues with source

**Solutions:**

1. **Test source URL directly:**

```bash
  curl -I "https://example.com/source-file.pdf"
```

2. **Check SSL certificates:**

```bash
  # Add CA certificate to container
  docker run -v ./ca.pem:/app/ca.pem your-image
```

3. **Verify network connectivity:**

```bash
  docker exec your-container ping example.com
```

4. **Check authorization forwarding:**

```yaml
forwardAuth: true # Forward Authorization header to source
```

### Performance Issues

#### High Memory Usage

**Symptoms:**

- Out of memory errors
- Service becoming unresponsive
- Container restarts

**Solutions:**

1. **Monitor memory usage:**

```bash
  docker stats your-container
  kubectl top pods
```

2. **Optimize command usage:**

```yaml
# Use streaming commands when possible
cmdByMimeType:
  default:
    cmd: "convert"
    args: ["-", "output:-"] # Stream input/output
```

3. **Set memory limits:**

```yaml
# Docker Compose
services:
  microservice:
    deploy:
      resources:
        limits:
          memory: 512M
```

4. **Process smaller files or implement file size limits.**

#### Slow Processing

**Symptoms:**

- Long response times
- Request timeouts

**Solutions:**

1. **Profile command performance:**

```bash
  time your-command < input-file > output-file
```

2. **Optimize command arguments:**

```yaml
# Example: faster image processing
cmdByMimeType:
  "image/*":
    cmd: "convert"
    args: ["-", "-quality", "80", "-resize", "300x300>", "jpg:-"]
```

3. **Scale horizontally:**

```yaml
# Kubernetes
spec:
  replicas: 3 # Run multiple instances
```

### File Processing Issues

#### Empty Output

**Symptoms:**

- HTTP 200 response with empty body
- No error messages in logs

**Causes:**

- Command produces no output
- Command writes to stderr instead of stdout
- Command requires specific arguments

**Solutions:**

1. **Test command manually:**

```bash
  echo "test input" | your-command
```

2. **Check command output redirection:**

```bash
  # Ensure command writes to stdout, not files
  your-command < input > /dev/stdout
```

3. **Debug with verbose logging:**

```yaml
# Set SCYLLARIDAE_LOG_LEVEL=DEBUG
environment:
  SCYLLARIDAE_LOG_LEVEL: DEBUG
```

#### Corrupted Output

**Symptoms:**

- Garbled or invalid output files
- File format errors

**Causes:**

- Binary/text encoding issues
- Command error output mixed with file output
- Incorrect command arguments

**Solutions:**

1. **Ensure binary mode for binary files:**

```bash
  curl --data-binary "@file.pdf" ...  # Use --data-binary for binary files
```

2. **Redirect stderr to prevent contamination:**

```bash
  #!/bin/bash
  your-command 2>/dev/null  # Redirect stderr
```

3. **Use wrapper scripts for complex commands:**

```bash
  #!/bin/bash
  input_file=$(mktemp)
  output_file=$(mktemp)

  cat > "$input_file"
  your-command "$input_file" "$output_file" 2>/dev/null
  cat "$output_file"

  rm "$input_file" "$output_file"
```

### Container and Runtime Issues

#### Service Won't Start

**Symptoms:**

- Container exits immediately
- "Container failed to start" errors

**Solutions:**

1. **Check container logs:**

```bash
  docker logs your-container
```

2. **Verify base image:**

```dockerfile
  FROM ghcr.io/lehigh-university-libraries/lehighlts/scyllaridae:main  # Ensure correct base image
```

3. **Test container interactively:**

```bash
  docker run -it your-image /bin/bash
```

4. **Check file permissions:**

```bash
  # Ensure scyllaridae.yml is readable
  ls -la /app/scyllaridae.yml
```

#### Port Binding Issues

**Symptoms:**

- "Port already in use" errors
- Cannot connect to service

**Solutions:**

1. **Check port availability:**

```bash
  netstat -tlnp | grep :8080
  lsof -i :8080
```

2. **Use different port:**

```bash
  docker run -p 8081:8080 your-image
```

3. **Check firewall settings:**

```bash
  # Ensure port is open
  ufw status
  iptables -L
```
