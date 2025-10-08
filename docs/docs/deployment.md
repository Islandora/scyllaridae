# Deployment Guide

This guide covers various deployment scenarios for scyllaridae microservices, from local development to production Kubernetes clusters.

## Security Considerations

- Always use HTTPS in production if scyllaridae is accessed across the network
- Have scyllaridae validate JWT tokens when handling sensitive content

## Docker Deployment (recommended)

### Basic Docker Run

The simplest way to deploy a scyllaridae microservice:

```bash
docker run -d \
  --name my-microservice \
  -p 8080:8080 \
  -v $(pwd)/scyllaridae.yml:/app/scyllaridae.yml:ro \
  islandora/scyllaridae:main
```

### Environment Variables

Configure the service using environment variables:

```bash
docker run -d \
  --name my-microservice \
  -p 8080:8080 \
  -e SCYLLARIDAE_PORT=8080 \
  -e SCYLLARIDAE_LOG_LEVEL=INFO \
  -e SCYLLARIDAE_YML_PATH=/app/scyllaridae.yml \
  -v $(pwd)/scyllaridae.yml:/app/scyllaridae.yml:ro \
  islandora/scyllaridae:main
```

### Using Docker Compose

Create a `docker-compose.yml` for your microservice:

```yaml
services:
  microservice:
    image: islandora/scyllaridae:main
    ports:
      - "8080:8080"
    environment:
      SCYLLARIDAE_PORT: 8080
      SCYLLARIDAE_LOG_LEVEL: INFO
    volumes:
      - ./scyllaridae.yml:/app/scyllaridae.yml:ro
      - ./ca.pem:/app/ca.pem:ro # Optional: custom CA certificate
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

Start the service:

```bash
docker-compose up -d
```

## Islandora Integration

### ISLE Docker Compose Integration

Add your microservice to an existing ISLE deployment:

```yaml
services:
  my-microservice-dev: &my-microservice
    <<: [*dev, *common]
    image: islandora/scyllaridae-my-microservice:main
    volumes:
      # Add CA certificate for development
      - ./certs/rootCA.pem:/app/ca.pem:ro
    networks:
      default:
        aliases:
          - my-microservice

  my-microservice-prod:
    <<: [*prod, *my-microservice]
```

### Alpaca Configuration

Configure Alpaca to send events to your microservice by adding to `alpaca.properties.tmpl`:

```properties
# Enable your microservice
derivative.my-microservice.enabled=true
derivative.my-microservice.in.stream=queue:islandora-connector-my-microservice
derivative.my-microservice.service.url=http://my-microservice:8080
derivative.my-microservice.concurrent-consumers=1
derivative.my-microservice.max-concurrent-consumers=3
derivative.my-microservice.async-consumer=true
```

Update the `ALPACA_DERIVATIVE_SYSTEMS` environment variable:

```yaml
environment:
  ALPACA_DERIVATIVE_SYSTEMS: "my-microservice"
```

## Kubernetes Deployment

### Basic Deployment

Create a Kubernetes deployment manifest:

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scyllaridae-microservice
  labels:
    app: scyllaridae-microservice
spec:
  replicas: 3
  selector:
    matchLabels:
      app: scyllaridae-microservice
  template:
    metadata:
      labels:
        app: scyllaridae-microservice
    spec:
      containers:
        - name: scyllaridae
          image: islandora/scyllaridae-my-microservice:latest
          ports:
            - containerPort: 8080
          env:
            - name: SCYLLARIDAE_PORT
              value: "8080"
            - name: SCYLLARIDAE_LOG_LEVEL
              value: "INFO"
          volumeMounts:
            - name: config
              mountPath: /app/scyllaridae.yml
              subPath: scyllaridae.yml
              readOnly: true
          readinessProbe:
            httpGet:
              path: /healthcheck
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
      volumes:
        - name: config
          configMap:
            name: scyllaridae-config

---
apiVersion: v1
kind: Service
metadata:
  name: scyllaridae-microservice
spec:
  selector:
    app: scyllaridae-microservice
  ports:
    - port: 8080
      targetPort: 8080
  type: ClusterIP

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: scyllaridae-config
data:
  scyllaridae.yml: |
    jwksUri: "https://islandora.example.com/oauth/discovery/keys"
    allowedMimeTypes:
      - "*"
    cmdByMimeType:
      default:
        cmd: "echo"
        args:
          - "Hello from Kubernetes"
```

Deploy to Kubernetes:

```bash
kubectl apply -f deployment.yaml
```

### Resource Management

Set appropriate resource requests and limits:

```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Native Binary Deployment

For environments where Docker is not available or preferred, you can run scyllaridae as a native binary.

### Prerequisites

For this example, we are using linux and systemd. For Windows servers an EXE of scyllaridae is available. For any OS you could use another init system than systemd.

- Required processing tools (ImageMagick, FFmpeg, etc.)
- Network access to download releases

### Installation

#### 1. Download the Binary

Download the latest release from GitHub:

**Linux/macOS:**

```bash
# Create application directory
sudo mkdir -p /opt/scyllaridae
cd /opt/scyllaridae

# Download latest release (replace with actual version and CPU architecture)
LATEST_VERSION=$(curl -s https://api.github.com/repos/lehigh-university-libraries/scyllaridae/releases/latest | grep tag_name | cut -d '"' -f 4)

# For Linux (x86_64)
curl -L -o scyllaridae.tar.gz "https://github.com/islandora/scyllaridae/releases/download/${LATEST_VERSION}/scyllaridae_Linux_x86_64.tar.gz"
sudo tar -xzf scyllaridae.tar.gz
sudo rm scyllaridae.tar.gz

# For Linux (arm64)
# curl -L -o scyllaridae.tar.gz "https://github.com/islandora/scyllaridae/releases/download/${LATEST_VERSION}/scyllaridae_Linux_arm64.tar.gz"
# sudo tar -xzf scyllaridae.tar.gz
# sudo rm scyllaridae.tar.gz

# For macOS (x86_64)
# curl -L -o scyllaridae.tar.gz "https://github.com/islandora/scyllaridae/releases/download/${LATEST_VERSION}/scyllaridae_Darwin_x86_64.tar.gz"
# sudo tar -xzf scyllaridae.tar.gz
# sudo rm scyllaridae.tar.gz

# For macOS (arm64)
# curl -L -o scyllaridae.tar.gz "https://github.com/islandora/scyllaridae/releases/download/${LATEST_VERSION}/scyllaridae_Darwin_arm64.tar.gz"
# sudo tar -xzf scyllaridae.tar.gz
# sudo rm scyllaridae.tar.gz

# Make executable
sudo chmod +x scyllaridae

# Create configuration file
sudo touch scyllaridae.yml
```

**Windows (PowerShell):**

```powershell
# Create application directory
New-Item -ItemType Directory -Force -Path C:\scyllaridae
Set-Location C:\scyllaridae

# Download latest release
$latestRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/lehigh-university-libraries/scyllaridae/releases/latest"
$version = $latestRelease.tag_name

# For Windows (x86_64)
$downloadUrl = "https://github.com/islandora/scyllaridae/releases/download/$version/scyllaridae_Windows_x86_64.zip"
Invoke-WebRequest -Uri $downloadUrl -OutFile "scyllaridae.zip"

# For Windows (arm64)
# $downloadUrl = "https://github.com/islandora/scyllaridae/releases/download/$version/scyllaridae_Windows_arm64.zip"
# Invoke-WebRequest -Uri $downloadUrl -OutFile "scyllaridae.zip"

# Extract the archive
Expand-Archive -Path "scyllaridae.zip" -DestinationPath . -Force
Remove-Item "scyllaridae.zip"

# Create configuration file
New-Item -ItemType File -Path "scyllaridae.yml"
```

#### 2. Install Processing Tools

Install the tools your microservice will use:

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y imagemagick ffmpeg poppler-utils tesseract-ocr

# RHEL/CentOS/Fedora
sudo dnf install -y ImageMagick ffmpeg poppler-utils tesseract

# Alpine Linux
sudo apk add --no-cache imagemagick ffmpeg poppler-utils tesseract-ocr
```

#### 3. Create System User

Create a dedicated user for the service:

```bash
sudo useradd --system --no-create-home --shell /bin/false scyllaridae
sudo chown -R scyllaridae:scyllaridae /opt/scyllaridae
```

#### 4. Configure the Service

Create your `scyllaridae.yml` configuration:

```bash
sudo tee /opt/scyllaridae/scyllaridae.yml > /dev/null <<'EOF'
# Example configuration
allowedMimeTypes:
  - "image/*"
  - "application/pdf"

cmdByMimeType:
  "image/*":
    cmd: "convert"
    args:
      - "-"
      - "-thumbnail"
      - "300x300>"
      - "-quality"
      - "80"
      - "jpg:-"

  "application/pdf":
    cmd: "convert"
    args:
      - "-density"
      - "150"
      - "pdf:-[0]"
      - "-thumbnail"
      - "300x300>"
      - "jpg:-"

  default:
    cmd: "echo"
    args:
      - "Unsupported file type"
EOF
```

### Systemd Service Configuration

#### 1. Create Service File

Create the systemd service file:

```bash
sudo tee /etc/systemd/system/scyllaridae.service > /dev/null <<'EOF'
[Unit]
Description=scyllaridae File Processing Service
Documentation=https://lehigh-university-libraries.github.io/scyllaridae/
After=network.target
Wants=network.target

[Service]
Type=simple
User=scyllaridae
Group=scyllaridae
WorkingDirectory=/opt/scyllaridae
ExecStart=/opt/scyllaridae/scyllaridae
ExecReload=/bin/kill -HUP $MAINPID
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=always
RestartSec=5

# Environment variables
Environment=SCYLLARIDAE_PORT=8080
Environment=SCYLLARIDAE_LOG_LEVEL=INFO
Environment=SCYLLARIDAE_YML_PATH=/opt/scyllaridae/scyllaridae.yml

# Add additional security settings and Resource limits as desired

[Install]
WantedBy=multi-user.target
EOF
```

#### 2. Enable and Start the Service

```bash
sudo systemctl enable scyllaridae
sudo systemctl start scyllaridae
sudo systemctl status scyllaridae
```

#### Monitoring and Logs

```bash
sudo journalctl -uf scyllaridae
```

### Configuration Management

#### Environment-Specific Configurations

Create environment-specific configuration files:

```bash
# Production configuration
sudo tee /opt/scyllaridae/scyllaridae-prod.yml > /dev/null <<'EOF'
jwksUri: "https://production.example.com/oauth/discovery/keys"
forwardAuth: true
allowedMimeTypes:
  - "image/*"
  - "application/pdf"
  - "video/*"

cmdByMimeType:
  "image/*":
    cmd: "convert"
    args:
      - "-"
      - "-strip"
      - "-quality"
      - "85"
      - "-thumbnail"
      - "300x300>"
      - "jpg:-"
EOF

# Update systemd service for production
sudo systemctl edit scyllaridae
```

Add environment-specific overrides:

```ini
[Service]
Environment=SCYLLARIDAE_YML_PATH=/opt/scyllaridae/scyllaridae-prod.yml
Environment=SCYLLARIDAE_LOG_LEVEL=WARN
```

#### Update Procedure

**Linux/macOS:**

```bash
#!/bin/bash
# update-scyllaridae.sh

set -e

sudo systemctl stop scyllaridae
sudo cp /opt/scyllaridae/scyllaridae /opt/scyllaridae/scyllaridae.backup

# Download new version
LATEST_VERSION=$(curl -s https://api.github.com/repos/lehigh-university-libraries/scyllaridae/releases/latest | grep tag_name | cut -d '"' -f 4)

# Determine platform and architecture
OS=$(uname -s)  # Darwin or Linux
ARCH=$(uname -m)  # x86_64, arm64, etc.

# Download and extract
curl -L -o /tmp/scyllaridae.tar.gz "https://github.com/islandora/scyllaridae/releases/download/${LATEST_VERSION}/scyllaridae_${OS}_${ARCH}.tar.gz"
cd /tmp
tar -xzf scyllaridae.tar.gz

# Install new binary
sudo mv /tmp/scyllaridae /opt/scyllaridae/scyllaridae
sudo chmod +x /opt/scyllaridae/scyllaridae
sudo chown scyllaridae:scyllaridae /opt/scyllaridae/scyllaridae
sudo rm /tmp/scyllaridae.tar.gz

sudo systemctl start scyllaridae
```

**Windows (PowerShell):**

```powershell
# update-scyllaridae.ps1

# Stop service (adjust for your Windows service name if needed)
# Stop-Service -Name "scyllaridae"

# Backup current binary
Copy-Item -Path "C:\scyllaridae\scyllaridae.exe" -Destination "C:\scyllaridae\scyllaridae.exe.backup" -Force

# Download new version
$latestRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/lehigh-university-libraries/scyllaridae/releases/latest"
$version = $latestRelease.tag_name

# Determine architecture (x86_64, arm64, etc.)
$arch = if ([Environment]::Is64BitOperatingSystem) { "x86_64" } else { "i386" }
$downloadUrl = "https://github.com/islandora/scyllaridae/releases/download/$version/scyllaridae_Windows_$arch.zip"

# Download and extract
Invoke-WebRequest -Uri $downloadUrl -OutFile "$env:TEMP\scyllaridae.zip"
Expand-Archive -Path "$env:TEMP\scyllaridae.zip" -DestinationPath "$env:TEMP\scyllaridae-update" -Force

# Install new binary
Move-Item -Path "$env:TEMP\scyllaridae-update\scyllaridae.exe" -Destination "C:\scyllaridae\scyllaridae.exe" -Force

# Cleanup
Remove-Item "$env:TEMP\scyllaridae.zip"
Remove-Item "$env:TEMP\scyllaridae-update" -Recurse -Force

# Start-Service -Name "scyllaridae"
```

### Monitoring and Health Checks

#### Health Check Script

```bash
#!/bin/bash
# health-check.sh

HEALTH_URL="http://localhost:8080/healthcheck"
TIMEOUT=10

if curl -f --max-time $TIMEOUT "$HEALTH_URL" > /dev/null 2>&1; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy"
    exit 1
fi
```

#### Monitoring with cron

```bash
# Add to crontab for regular health checks
# crontab -e
*/5 * * * * /opt/scyllaridae/health-check.sh || echo "scyllaridae health check failed" | mail -s "Service Alert" admin@example.com
```

### Troubleshooting Native Deployment

#### Common Issues

1. **Service won't start**:

```bash
# Check systemd status
sudo systemctl status scyllaridae

# Check logs
sudo journalctl -u scyllaridae --no-pager

# Test binary directly
sudo -u scyllaridae /opt/scyllaridae/scyllaridae
```

2. **Permission errors**:

```bash
# Fix ownership
sudo chown -R scyllaridae:scyllaridae /opt/scyllaridae

# Check file permissions
ls -la /opt/scyllaridae/
```

3. **Command not found errors**:

```bash
# Check if tools are installed
which convert ffmpeg

# Test command as scyllaridae user
sudo -u scyllaridae convert --version
```

4. **Network connectivity**:

```bash
# Test local connectivity
curl http://localhost:8080/healthcheck

# Check port binding
sudo netstat -tlnp | grep :8080
```
