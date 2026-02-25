# Agent Usage Tracker Makefile

.PHONY: build test clean install run help

# Build configuration
BINARY_NAME=agent-usage
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go configuration
GO=go
GOFLAGS=-v

# Default target
help:
	@echo "Agent Usage Tracker - Build Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build the binary to $(BUILD_DIR)/"
	@echo "  build/osx    Build for macOS (darwin)"
	@echo "  build/linux  Build for Linux"
	@echo "  build/windows Build for Windows"
	@echo "  test         Run tests"
	@echo "  clean        Remove build artifacts"
	@echo "  install      Build and install to GOBIN"
	@echo "  run          Build and run"
	@echo ""

# Build binary
build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for different platforms
build/osx:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"

build/linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build/windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Run tests
test:
	$(GO) test $(GOFLAGS) ./...

test/verbose:
	$(GO) test -v ./...

test/coverage:
	$(GO) test -cover ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)

# Install binary
install:
	$(GO) install $(LDFLAGS) .

# Build and run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development
dev:
	$(GO) run .

# Lint
lint:
	$(GO) vet ./...
	golangci-lint run || true

# Format
fmt:
	$(GO) fmt ./...

# Tidy dependencies
tidy:
	$(GO) mod tidy
