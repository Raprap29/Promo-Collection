# --- Project Variables ---
APP_NAME ?= "promo-collection" 
BUILD_DIR ?= ./build 
BIN_NAME ?= $(APP_NAME) 
MAIN_GO ?= ./cmd/$(APP_NAME)/main.go 

# Set a default for the verbose flag.
# By default, --verbose is off. Set LINT_VERBOSE=--verbose to enable.
VERBOSE ?= "-v"

# LINT_COMMAND allows overriding the entire golangci-lint command.
# By default, it's set to 'run'.
LINT_COMMAND ?= run

# --- Go Commands ---
GO = go 
GO_BUILD ?= $(GO) build -o $(BUILD_DIR)/$(BIN_NAME) 
GO_TEST = $(GO) test $(VERBOSE)

# GO_LINT is now a combination of the command and the verbose parameter.
GO_LINT = golangci-lint $(LINT_COMMAND) $(VERBOSE)

GO_FMT = $(GO) fmt ./... 
GO_MOD_TIDY = $(GO) mod tidy 
MIN_COVERAGE ?= 80 

.PHONY: all build test lint fmt run docker-build docker-run mod-tidy help coverage 

all: build 

build: lint test ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) $(MAIN_GO)
	@echo "Build successful: $(BUILD_DIR)/$(BIN_NAME)"

test: ## Run all unit tests
	@echo "Running tests..."
	$(GO_TEST) ./...

lint: ## Run linters with golangci-lint. Use `make lint VERBOSE=-v` for verbose output.
	@echo "Running linters..."
	$(GO_LINT)

fmt: ## Format source code with gofmt
	@echo "Formatting code..."
	$(GO_FMT)

run: build ## Build and run the application locally
	@echo "Running $(APP_NAME)..." 
	@$(BUILD_DIR)/$(BIN_NAME) 

docker-build: ## Build Docker image
	@echo "Building Docker image $(APP_NAME):$(VERSION)..."
	docker build -t $(APP_NAME):$(VERSION) .

docker-run: ## Run Docker container (requires docker-build first)
	@echo "Running Docker container $(APP_NAME):$(VERSION)..." 
	docker run --rm -p 8080:8080 $(APP_NAME):$(VERSION) 

mod-tidy: ## Run go mod tidy
	@echo "Running go mod tidy..." 
	$(GO_MOD_TIDY) 

coverage: ## Run tests with coverage and ensure it's > MIN_COVERAGE%
	@echo "Running tests with coverage check...(min: $(MIN_COVERAGE)%)..."
	@$(GO_TEST) -coverprofile=coverage.out ./internal/... 
	@if [ ! -s coverage.out ]; then echo "Error: coverage.out is empty or not created"; exit 1; fi 
	@echo "Generating HTML coverage report: coverage.html" 
	@go tool cover -html=coverage.out -o coverage.html 
	@echo "HTML coverage report generated: coverage.html" 
	@go tool cover -func=coverage.out 
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | grep -o '[0-9]*\.[0-9]%'); \
	if [ -z "$$COVERAGE" ]; then echo "Error: Failed to extract coverage"; exit 1; fi; \
	echo "COVERAGE=$$COVERAGE"; \
	COVERAGE_NUM=$$(echo "$$COVERAGE" | sed 's/%.*//'); \
	if [ $$(echo "$$COVERAGE_NUM < $(MIN_COVERAGE)" | bc -l) -eq 1 ]; \
	then echo "Error: Coverage $$COVERAGE_NUM% is below 80%"; exit 1; fi; \
	echo "Coverage $$COVERAGE_NUM% meets the required $(MIN_COVERAGE)%."
	
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' 