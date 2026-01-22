.PHONY: help run build clean generate install-deps test dev fmt tidy create-user css-build css-watch docker-build docker-run dupl

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install project dependencies
	go mod download
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest
	npm install

generate: ## Generate Templ templates
	$(HOME)/go/bin/templ generate

css-build: ## Build CSS with PostCSS and Tailwind
	npm run build:css

css-watch: ## Watch and rebuild CSS on changes
	npm run watch:css

dev: css-build ## Run with live-reload (requires air and npm)
	@echo "Starting development servers..."
	@trap 'kill 0' EXIT; \
	npm run watch:css & \
	$(HOME)/go/bin/air

run: generate ## Run the application
	go run cmd/server/main.go

build: generate css-build ## Build the application
	go build -trimpath -o bin/server cmd/server/main.go

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f db/*.db
	rm -f db/*.db-shm
	rm -f db/*.db-wal
	rm -rf static/uploads/
	rm -rf uploads/
	find . -name "*_templ.go" -delete

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...
	$(HOME)/go/bin/templ fmt .

tidy: ## Tidy dependencies
	go mod tidy

create-user: ## Create a new user (interactive CLI)
	@go run cmd/create-user/main.go

docker-build: ## Build Docker image
	docker build -t lexlegalcloud-app .

docker-run: ## Run Docker container locally
	docker run -p 8080:8080 --env-file .env -v $(PWD)/db:/app/db lexlegalcloud-app

dupl: ## Find duplicate code (threshold: 100 tokens), excluding templ files
	@test -f $(HOME)/go/bin/dupl || { echo "Installing dupl..."; go install github.com/mibk/dupl@latest; }
	find . -type f -name "*.go" -not -name "*_templ.go" | $(HOME)/go/bin/dupl -files -t 100
