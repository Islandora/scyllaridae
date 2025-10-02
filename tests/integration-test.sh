#!/usr/bin/env bash

set -eou pipefail

DOCKER_IMAGE="scyllaridae"
DOCKER_CONTAINER="$DOCKER_IMAGE-test"
TEST_DIR="./tests"

echo "Setting up integration test environment..."
rm -f "$TEST_DIR"/*.bin

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
	-v "$(pwd)/$TEST_DIR/cmd.sh:/app/cmd.sh" \
	-v "$(pwd)/$TEST_DIR/scyllaridae.yml:/app/scyllaridae.yml" \
	--name "$DOCKER_CONTAINER" \
	-p "$PORT:8080" \
	"$DOCKER_IMAGE:latest" > /dev/null

echo "Waiting for container to be ready..."
sleep 2

echo "Running integration tests..."
FAILED=0

for bin_file in "$TEST_DIR"/*.bin; do
	filename=$(basename "$bin_file")
	name="${filename%.bin}"

	echo "Testing: $name"
	curl -s --data-binary "@$bin_file" "http://localhost:$PORT" > "$TEST_DIR/$name-result.bin"
	ORIGINAL=$(md5sum "$bin_file" | cut -d' ' -f1)
	RESULT=$(md5sum "$TEST_DIR/$name-result.bin" | cut -d' ' -f1)

	if [ "$ORIGINAL" = "$RESULT" ]; then
		echo "✓ $name test passed (MD5: $ORIGINAL)"
	else
		echo "✗ $name test FAILED (expected: $ORIGINAL, got: $RESULT)"
		FAILED=$((FAILED + 1))
	fi
done

echo "Cleaning up..."
docker stop "$DOCKER_CONTAINER" > /dev/null
docker rm "$DOCKER_CONTAINER" > /dev/null

if [ $FAILED -eq 0 ]; then
	echo ""
	echo "✓ All integration tests passed!"
	exit 0
else
	echo ""
	echo "✗ $FAILED test(s) failed"
	exit 1
fi
