#!/usr/bin/env bash

set -eou pipefail

DOCKER_IMAGE="scyllaridae"
DOCKER_CONTAINER="$DOCKER_IMAGE-test"
TEST_DIR="$(pwd)/tests"
GITHUB_TOKEN=""

# shellcheck disable=SC2317,SC2329
cleanup() {
	echo ""
	echo "Cleaning up..."
	if [ "${1:-}" = "error" ]; then
		echo "Error occurred, showing container logs:"
		docker logs "$DOCKER_CONTAINER" 2>&1 || true
	fi
	docker stop "$DOCKER_CONTAINER" 2>/dev/null || true
	docker rm "$DOCKER_CONTAINER" 2>/dev/null || true
  rm -f "$TEST_DIR"/*.bin
}

trap 'cleanup error' ERR
trap 'cleanup' EXIT

CONFIG_FILE="$TEST_DIR/scyllaridae.yml"

# Fetch GitHub OIDC token if running in GitHub Actions
if [ -n "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}" ] && [ -n "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" ]; then
	echo "Detected GitHub Actions environment, fetching OIDC token..."
	GITHUB_TOKEN=$(curl -s \
		-H "Accept: application/json; api-version=2.0" \
		-H "Content-Type: application/json" -d "{}" \
		-H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
		"$ACTIONS_ID_TOKEN_REQUEST_URL" | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

	if [ -z "$GITHUB_TOKEN" ]; then
		echo "Failed to fetch GitHub OIDC token"
		exit 1
	fi

	echo "Successfully fetched GitHub OIDC token"
	# Add buffer to avoid iat issues
	sleep 5

	# Use GitHub-specific config with JWKS URI
	CONFIG_FILE="$TEST_DIR/scyllaridae.github.yml"
fi

echo "Setting up integration test environment..."

# Create test files of different sizes
echo "Creating test files..."
dd if=/dev/urandom of="$TEST_DIR/small.bin" bs=1024 count=1024 2>/dev/null  # 1MB
dd if=/dev/urandom of="$TEST_DIR/exact.bin" bs=1024 count=2048 2>/dev/null  # 2MB
dd if=/dev/urandom of="$TEST_DIR/large.bin" bs=1024 count=3072 2>/dev/null  # 3MB

# Stop and remove any existing test container
docker stop "$DOCKER_CONTAINER" 2>/dev/null || true
docker rm "$DOCKER_CONTAINER" 2>/dev/null || true

# Find available port
PORT=8080
while lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; do
	PORT=$((PORT + 1))
done

echo "Starting test container on port $PORT..."
docker run -d \
	-v "$TEST_DIR/cmd.sh:/app/cmd.sh" \
	-v "$CONFIG_FILE:/app/scyllaridae.yml" \
	--name "$DOCKER_CONTAINER" \
	-p "$PORT:8080" \
	"$DOCKER_IMAGE:latest" > /dev/null

echo "Waiting for container to be ready..."
sleep 2

echo "Running integration tests..."
FAILED=0

# Test with invalid JWT if we're in GitHub Actions
if [ -n "$GITHUB_TOKEN" ]; then
	echo "Testing: invalid JWT authentication"
	HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
		--data-binary "@$TEST_DIR/small.bin" \
		-H "Authorization: Bearer invalid.jwt.token" \
		"http://localhost:$PORT")

	if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "403" ]; then
		echo "✓ Invalid JWT correctly rejected (HTTP $HTTP_CODE)"
	else
		echo "✗ Invalid JWT test FAILED (expected 401/403, got HTTP $HTTP_CODE)"
		FAILED=$((FAILED + 1))
	fi

	# Test forwardAuth: true - SCYLLARIDAE_AUTH should be present
	echo "Testing: forwardAuth=true (SCYLLARIDAE_AUTH should be present)"
	curl -s --data-binary "@$TEST_DIR/small.bin" \
		-H "Authorization: Bearer $GITHUB_TOKEN" \
		"http://localhost:$PORT" > /dev/null 2>&1
	if docker logs "$DOCKER_CONTAINER" 2>&1 | grep -q "SCYLLARIDAE_AUTH=present"; then
		echo "✓ forwardAuth=true: SCYLLARIDAE_AUTH correctly passed to cmd"
	else
		echo "✗ forwardAuth=true test FAILED: SCYLLARIDAE_AUTH not present"
		FAILED=$((FAILED + 1))
	fi
fi

for bin_file in "$TEST_DIR"/*.bin; do
	filename=$(basename "$bin_file")
	name="${filename%.bin}"

	echo "Testing: $name"
	if [ -n "$GITHUB_TOKEN" ]; then
		curl -s --data-binary "@$bin_file" \
			-H "Authorization: Bearer $GITHUB_TOKEN" \
			"http://localhost:$PORT" > "$TEST_DIR/$name-result.bin"
	else
		curl -s --data-binary "@$bin_file" "http://localhost:$PORT" > "$TEST_DIR/$name-result.bin"
	fi
	ORIGINAL=$(md5sum "$bin_file" | cut -d' ' -f1)
	RESULT=$(md5sum "$TEST_DIR/$name-result.bin" | cut -d' ' -f1)

	if [ "$ORIGINAL" = "$RESULT" ]; then
		echo "✓ $name test passed (MD5: $ORIGINAL)"
	else
		echo "✗ $name test FAILED (expected: $ORIGINAL, got: $RESULT)"
		FAILED=$((FAILED + 1))
	fi
done

# Test forwardAuth: false if in GitHub Actions
if [ -n "$GITHUB_TOKEN" ]; then
	echo ""
	echo "Testing forwardAuth=false configuration..."

	# Stop existing container
	docker stop "$DOCKER_CONTAINER" 2>/dev/null || true
	docker rm "$DOCKER_CONTAINER" 2>/dev/null || true

	# Start container with forwardAuth: false config
	echo "Starting test container with forwardAuth=false on port $PORT..."
	docker run -d \
		-v "$TEST_DIR/cmd.sh:/app/cmd.sh" \
		-v "$TEST_DIR/scyllaridae.github-noforward.yml:/app/scyllaridae.yml" \
		--name "$DOCKER_CONTAINER" \
		-p "$PORT:8080" \
		"$DOCKER_IMAGE:latest" > /dev/null

	echo "Waiting for container to be ready..."
	sleep 2

	# Test that SCYLLARIDAE_AUTH is NOT present when forwardAuth: false
	echo "Testing: forwardAuth=false (SCYLLARIDAE_AUTH should be absent)"
	curl -s --data-binary "@$TEST_DIR/small.bin" \
		-H "Authorization: Bearer $GITHUB_TOKEN" \
		"http://localhost:$PORT" > /dev/null 2>&1
	if docker logs "$DOCKER_CONTAINER" 2>&1 | grep -q "SCYLLARIDAE_AUTH=absent"; then
		echo "✓ forwardAuth=false: SCYLLARIDAE_AUTH correctly NOT passed to cmd"
	else
		echo "✗ forwardAuth=false test FAILED: SCYLLARIDAE_AUTH was present"
		FAILED=$((FAILED + 1))
	fi
fi

if [ $FAILED -eq 0 ]; then
	echo ""
	echo "✓ All integration tests passed!"
	exit 0
else
	echo ""
	echo "✗ $FAILED test(s) failed"
	exit 1
fi
