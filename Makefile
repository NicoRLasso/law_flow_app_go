.PHONY: help run build clean generate install-deps test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install project dependencies
	go mod download
	go install github.com/a-h/templ/cmd/templ@latest

generate: ## Generate Templ templates
	templ generate

run: generate ## Run the application
	go run cmd/server/main.go

build: generate ## Build the application
	go build -o bin/server cmd/server/main.go

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f db/*.db
	find . -name "*_templ.go" -delete

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...
	templ fmt .

tidy: ## Tidy dependencies
	go mod tidy
