.PHONY: build deps run lint docker test docs integration-test

BINARY_NAME=scyllaridae

DOCKER_IMAGE=scyllaridae

deps:
	go get .
	go mod tidy

build: deps
	go build -o $(BINARY_NAME) .

run: docker
	@docker stop $(DOCKER_IMAGE) 2>/dev/null || true
	@docker rm $(DOCKER_IMAGE) 2>/dev/null || true
	@PORT=8080; \
	VOLUMES="-v ./scyllaridae.yml:/app/scyllaridae.yml"; \
	if [ -f ./cmd.sh ]; then \
		VOLUMES="$$VOLUMES -v ./cmd.sh:/app/cmd.sh"; \
	fi; \
	while lsof -Pi :$$PORT -sTCP:LISTEN -t >/dev/null 2>&1; do \
		PORT=$$((PORT + 1)); \
	done; \
	echo "Starting scyllaridae at http://localhost:$$PORT"; \
	docker run -d $$VOLUMES --name $(DOCKER_IMAGE) -p $$PORT:8080 $(DOCKER_IMAGE):latest > /dev/null

lint:
	go fmt ./...
	golangci-lint run

	@if command -v yq > /dev/null 2>&1; then \
		echo "Running yq validation on YAML files..."; \
		yq . **/*.yml > /dev/null; \
	else \
		echo "yq not found, skipping YAML validation"; \
	fi

	@if command -v shellcheck > /dev/null 2>&1; then \
		echo "Running shellcheck on shell scripts..."; \
		shellcheck **/*.sh; \
	else \
		echo "shellcheck not found, skipping shell script validation"; \
	fi

	@if command -v hadolint > /dev/null 2>&1; then \
		echo "Running hadolint on Dockerfiles..."; \
		find . -name "Dockerfile" | xargs hadolint; \
	else \
		echo "hadolint not found, skipping Dockerfile validation"; \
	fi

docker:
	docker build -t $(DOCKER_IMAGE):latest .

test:
	go test -v -race ./...

integration-test: docker
	./tests/integration-test.sh

docs:
	docker build -t $(DOCKER_IMAGE)-docs:latest docs
	@docker stop $(DOCKER_IMAGE)-docs 2>/dev/null || true
	@PORT=8080; \
	while lsof -Pi :$$PORT -sTCP:LISTEN -t >/dev/null 2>&1; do \
		PORT=$$((PORT + 1)); \
	done; \
	echo "Starting documentation server at http://localhost:$$PORT"; \
	docker run -d --rm --name $(DOCKER_IMAGE)-docs -p $$PORT:80 $(DOCKER_IMAGE)-docs:latest > /dev/null
