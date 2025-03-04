APP_NAME := tv-shows-bot
DOCKER_REPO := ghcr.io/deniskhalizov/$(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty)
MAIN_PKG := ./cmd/main.go
BUILD_DIR := ./bin
CONFIG_FILE := config.yaml
DOCKER_COMPOSE_FILE := docker-compose.yml
TEST_COVERAGE_PROFILE := coverage.out

GO_TOOLS := golangci-lint mockgen godoc gopls goimports staticcheck dlv govulncheck

LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"
GOFLAGS := -trimpath

.PHONY: default
default: help

.PHONY: help
help:
	@echo "TV Shows Notification Bot Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build the binary"
	@echo "  run           Run the application locally"
	@echo "  test          Run unit tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  lint          Run linter"
	@echo "  docker-build  Build Docker image"
	@echo "  clean         Clean build artifacts"
	@echo "  mock-apis     Generate mock API clients for testing"
	@echo "  setup-env     Setup local development environment"
	@echo "  fmt           Format Go code"
	@echo "  vet           Run Go vet"
	@echo "  help          Show this help"


.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PKG)
	@echo "Binary built successfully: $(BUILD_DIR)/$(APP_NAME)"


.PHONY: run
run:
	@echo "Running $(APP_NAME)..."
	@go run $(MAIN_PKG)


.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...


.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=$(TEST_COVERAGE_PROFILE) -covermode=atomic ./...
	@go tool cover -html=$(TEST_COVERAGE_PROFILE)


.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@go tool golangci-lint run --fix --fast ./...
	@echo "Running staticcheck..."
	@go tool staticcheck ./...


.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...
	@echo "Running goimports..."
	@go tool goimports -w -local "shows" ./


.PHONY: vet
vet:
	@echo "Running Go vet..."
	@go vet ./...


.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(DOCKER_REPO):$(VERSION)..."
	@docker build $(DOCKER_BUILD_ARGS) -t $(DOCKER_REPO):$(VERSION) -t $(DOCKER_REPO):latest .

.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(TEST_COVERAGE_PROFILE)
	@go clean -cache -testcache


.PHONY: setup-env
setup-env:
	@echo "Setting up local development environment..."
	@go mod download
	@echo "Adding tool dependencies..."
	@go get -tool github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go get -tool golang.org/x/tools/cmd/godoc@latest
	@go get -tool golang.org/x/tools/gopls@latest
	@go get -tool golang.org/x/tools/cmd/goimports@latest
	@go get -tool honnef.co/go/tools/cmd/staticcheck@latest
	@go get -tool github.com/go-delve/delve/cmd/dlv@latest
	@go get -tool golang.org/x/vuln/cmd/govulncheck@latest
	@go mod tidy
	@cp -n config.example.yaml $(CONFIG_FILE) || true
	@echo "Development environment setup complete!"

.PHONY: build-all
build-all:
	@echo "Building for all supported platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_PKG)
	@GOOS=linux GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_PKG)
	@echo "Binaries built in $(BUILD_DIR)/"


.PHONY: vuln
vuln:
	@echo "Checking for vulnerabilities in dependencies..."
	@go tool govulncheck ./...


.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...


.PHONY: tools-update
tools-update:
	@echo "Updating all tool dependencies..."
	@go get -u tool
	@go mod tidy
	@echo "Tool dependencies updated."


.PHONY: tools-list
tools-list:
	@echo "Registered tool dependencies:"
	@go tool