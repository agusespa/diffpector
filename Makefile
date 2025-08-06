.PHONY: all build run clean tidy setup devrun test test-coverage

PROJECT_NAME := codspectator
MAIN_GO_FILE := ./cmd/codspectator/main.go
BINARY_NAME := $(PROJECT_NAME)

all: build run

setup:
	@echo "Setting up Go modules..."
	go mod tidy
	@echo "Go modules are set up."

build: setup
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_GO_FILE)
	@echo "Build complete. Executable: $(BINARY_NAME)"

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

devrun: setup
	@echo "Running $(MAIN_GO_FILE) directly..."
	go run $(MAIN_GO_FILE)

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
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
