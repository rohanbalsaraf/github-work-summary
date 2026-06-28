.PHONY: all build test lint fmt tidy clean

BINARY_NAME=gws

all: fmt lint test build

build:
	@echo "==> Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go

test:
	@echo "==> Running tests..."
	go test -v ./...

lint:
	@echo "==> Running linters..."
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run --go=1.24; else echo "golangci-lint not installed, skipping full lint..."; fi

fmt:
	@echo "==> Formatting code..."
	go fmt ./...

tidy:
	@echo "==> Tidying dependencies..."
	go mod tidy

clean:
	@echo "==> Cleaning build cache..."
	go clean
	rm -f $(BINARY_NAME)
