.PHONY: build deps lint docker test docs

BINARY_NAME=scyllaridae

DOCKER_IMAGE=scyllaridae

deps:
	go get .
	go mod tidy

build: deps
	go build -o $(BINARY_NAME) .

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
		shellcheck *.sh; \
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

docs:
	docker build -t $(DOCKER_IMAGE)-docs:latest docs
	docker run -p 8080:80 $(DOCKER_IMAGE)-docs:latest
