# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
BINARY_NAME=apollo

# Component names
CLI_BINARY=apollo-cli
API_BINARY=apollo-api
OPERATOR_BINARY=apollo-operator

# Build directories
BUILD_DIR=build
CLI_DIR=cmd/cli
API_DIR=cmd/api/server
OPERATOR_DIR=cmd/operator

# Docker parameters
DOCKER_CMD=docker
DOCKER_BUILD=$(DOCKER_CMD) build
DOCKER_RUN=$(DOCKER_CMD) run
DOCKER_TAG=$(DOCKER_CMD) tag
DOCKER_PUSH=$(DOCKER_CMD) push

# Version
VERSION=0.1.0

.PHONY: all build test clean run-cli run-api run-operator docker-build docker-push

all: test build

build: build-cli build-api build-operator

build-cli:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(CLI_BINARY) ./$(CLI_DIR)

build-api:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(API_BINARY) ./$(API_DIR)

build-operator:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(OPERATOR_BINARY) ./$(OPERATOR_DIR)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

run-cli:
	$(GOBUILD) -o $(BUILD_DIR)/$(CLI_BINARY) ./$(CLI_DIR)
	./$(BUILD_DIR)/$(CLI_BINARY) $(ARGS)

run-api:
	$(GOBUILD) -o $(BUILD_DIR)/$(API_BINARY) ./$(API_DIR)
	./$(BUILD_DIR)/$(API_BINARY) $(ARGS)

run-operator:
	$(GOBUILD) -o $(BUILD_DIR)/$(OPERATOR_BINARY) ./$(OPERATOR_DIR)
	./$(BUILD_DIR)/$(OPERATOR_BINARY) $(ARGS)

# Docker targets
docker-build: docker-build-cli docker-build-api docker-build-operator

docker-build-cli:
	$(DOCKER_BUILD) -t $(BINARY_NAME)-cli:$(VERSION) -f Dockerfile.cli .

docker-build-api:
	$(DOCKER_BUILD) -t $(BINARY_NAME)-api:$(VERSION) -f Dockerfile.api .

docker-build-operator:
	$(DOCKER_BUILD) -t $(BINARY_NAME)-operator:$(VERSION) -f Dockerfile.operator .

docker-push:
	$(DOCKER_TAG) $(BINARY_NAME)-cli:$(VERSION) $(DOCKER_REGISTRY)/$(BINARY_NAME)-cli:$(VERSION)
	$(DOCKER_TAG) $(BINARY_NAME)-api:$(VERSION) $(DOCKER_REGISTRY)/$(BINARY_NAME)-api:$(VERSION)
	$(DOCKER_TAG) $(BINARY_NAME)-operator:$(VERSION) $(DOCKER_REGISTRY)/$(BINARY_NAME)-operator:$(VERSION)
	$(DOCKER_PUSH) $(DOCKER_REGISTRY)/$(BINARY_NAME)-cli:$(VERSION)
	$(DOCKER_PUSH) $(DOCKER_REGISTRY)/$(BINARY_NAME)-api:$(VERSION)
	$(DOCKER_PUSH) $(DOCKER_REGISTRY)/$(BINARY_NAME)-operator:$(VERSION)

# Development helpers
deps:
	$(GOMOD) download
	$(GOMOD) tidy

lint:
	golangci-lint run

# Help target
help:
	@echo "Available targets:"
	@echo "  build        - Build all components"
	@echo "  build-cli    - Build CLI component"
	@echo "  build-api    - Build API component"
	@echo "  build-operator - Build operator component"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  run-cli      - Run CLI component"
	@echo "  run-api      - Run API component"
	@echo "  run-operator - Run operator component"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-push  - Push Docker images"
	@echo "  deps         - Download dependencies"
	@echo "  lint         - Run linter" 