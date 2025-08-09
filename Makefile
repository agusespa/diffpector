.PHONY: all build run clean tidy setup devrun test test-coverage eval eval-build eval-run eval-compare eval-clean eval-list-prompts eval-quick eval-comprehensive eval-security eval-performance

PROJECT_NAME := diffpector
MAIN_GO_FILE := ./cmd/diffpector/main.go
EVAL_GO_FILE := ./cmd/eval/main.go
BUILD_DIR := build
BINARY_NAME := $(BUILD_DIR)/$(PROJECT_NAME)
EVAL_BINARY_NAME := $(BUILD_DIR)/$(PROJECT_NAME)-eval

all: build run

setup:
	@echo "Setting up Go modules..."
	go mod tidy
	@echo "Go modules are set up."

build: setup
	@echo "Creating build directory if it doesn't exist..."
	@mkdir -p $(BUILD_DIR)
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

eval-build: setup
	@echo "Creating build directory if it doesn't exist..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building evaluation tool..."
	go build -o $(EVAL_BINARY_NAME) $(EVAL_GO_FILE)
	@echo "Evaluation tool built: $(EVAL_BINARY_NAME)"

eval-run: eval-build
	@echo "Running evaluation with default settings..."
	./$(EVAL_BINARY_NAME)

eval-compare: eval-build
	@echo "Comparing existing evaluation results..."
	./$(EVAL_BINARY_NAME) --compare

eval-list-prompts: eval-build
	@echo "Listing available prompt variants..."
	./$(EVAL_BINARY_NAME) --list-prompts

eval-all: eval-build
	@echo "Running all evaluations..."
	./$(EVAL_BINARY_NAME)

eval-prompt: eval-build
	@echo "Running prompt evaluation..."
	./$(EVAL_BINARY_NAME) --variant prompt-comparison

eval-model: eval-build
	@echo "Running model evaluation..."
	./$(EVAL_BINARY_NAME) --variant model-comparison

eval-comprehensive: eval-build
	@echo "Running comprehensive evaluation..."
	./$(EVAL_BINARY_NAME) --variant full-comprehensive
