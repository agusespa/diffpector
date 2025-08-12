.PHONY: all build run clean tidy setup devrun test test-coverage build-release

PROJECT_NAME := diffpector
MAIN_GO_FILE := ./cmd/diffpector/main.go
BUILD_DIR := build
RELEASE_DIR := release
BINARY_NAME := $(BUILD_DIR)/$(PROJECT_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

include evaluation/Makefile

all: build run

setup:
	@echo "Setting up Go modules..."
	go mod tidy
	@echo "Go modules are set up."

build: setup
	@echo "Creating build directory if it doesn't exist..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME) $(MAIN_GO_FILE)
	@echo "Build complete. Executable: $(BINARY_NAME)"

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

devrun: setup
	@echo "Running $(MAIN_GO_FILE) directly..."
	go run $(MAIN_GO_FILE)

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

tidy:
	@echo "Tidying go.mod and go.sum..."
	go mod tidy
	@echo "go.mod and go.sum tidied."

test:
	@echo "Running tests..."
	go test ./... -v
	@echo "Tests complete."

test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

build-release: setup
	@echo "Building release binaries..."
	@mkdir -p $(RELEASE_DIR)
	
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(RELEASE_DIR)/release/$(PROJECT_NAME)-linux-amd64 $(MAIN_GO_FILE)
	
	@echo "Building for macOS arm64..."
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(RELEASE_DIR)/release/$(PROJECT_NAME)-darwin-arm64 $(MAIN_GO_FILE)
	
	@echo "Release binaries built in $(RELEASE_DIR)"