.PHONY: help run build clean generate install-deps test dev fmt tidy create-user css css-watch docker-build docker-run dupl

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install project dependencies
	go mod download
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest

generate: ## Generate Templ templates
	$(HOME)/go/bin/templ generate

css: ## Build CSS with Tailwind v4
	bun x tailwindcss -i static/css/input.css -o static/css/style.css --minify

css-watch: ## Watch CSS changes
	bun x tailwindcss -i static/css/input.css -o static/css/style.css --watch

air:
	$(HOME)/go/bin/air

dev: generate ## Run with live-reload (requires air)
	@echo "Starting development servers..."
	make -j2 air css-watch

run: generate ## Run the application
	CGO_ENABLED=1 go run -tags "fts5" cmd/server/main.go

build: generate css ## Build the application
	CGO_ENABLED=1 go build -tags "fts5" -trimpath -o bin/server cmd/server/main.go

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f db/*.db
	rm -f db/*.db-shm
	rm -f db/*.db-wal
	rm -rf static/uploads/
	rm -rf uploads/
	find . -name "*_templ.go" -delete

unit-test: ## Run tests
	go test -tags fts5 $(ARGS) $$(go list ./... | grep -vE 'templates|static|models|db|config') -cover 

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

security: ## Run security scans (gosec, govulncheck)
	@echo "Running security checks..."
	@test -f $(HOME)/go/bin/gosec || { echo "Installing gosec..."; go install github.com/securego/gosec/v2/cmd/gosec@latest; }
	@test -f $(HOME)/go/bin/govulncheck || { echo "Installing govulncheck..."; go install golang.org/x/vuln/cmd/govulncheck@latest; }
	@echo ">> Running gosec..."
	@$(HOME)/go/bin/gosec -exclude-dir=templates -exclude-dir=node_modules ./...
	@echo ">> Running govulncheck..."
	@$(HOME)/go/bin/govulncheck ./...
