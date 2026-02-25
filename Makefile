.PHONY: all build run test test-coverage clean frontend backend docker dev dev-full fmt fmt-check lint lint-frontend lint-backend lint-md typecheck typecheck-frontend typecheck-backend lint-golangci check-policy quality demo-db phone-setup-https phone-setup-ip-cert phone-frontend-dev phone-frontend-preview phone-caddy phone-help

# Default target
all: build

# Build everything
# Tip: use 'make build -j2' for parallel frontend+backend builds
build: frontend backend

# Build frontend
frontend:
	cd frontend && npm install && npm run build
	rm -rf backend/cmd/server/static/*
	cp -r frontend/build/* backend/cmd/server/static/

# Build backend
backend:
	VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	cd backend && CGO_ENABLED=1 go build -tags "fts5 sqlite_crypt" \
	  -ldflags="-X github.com/xela-io/xelanote/internal/api.Version=$$VERSION" \
	  -o ../bin/xelanote ./cmd/server

# Run development server (backend only)
run-backend:
	cd backend && CGO_ENABLED=1 go run -tags "fts5 sqlite_crypt" ./cmd/server -addr :8080

# Run frontend dev server
run-frontend:
	cd frontend && env -u VITE_API_BASE_URL -u VITE_API_PROXY_TARGET npm run dev:local

# Run tests
test:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod go test -tags "fts5 sqlite_crypt" -v ./...

# Run tests with coverage reports
test-coverage:
	@echo "=== Backend Coverage ==="
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod go test -tags "fts5 sqlite_crypt" -coverprofile=coverage.out -covermode=atomic ./...
	cd backend && go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "=== Frontend Coverage ==="
	cd frontend && npm run test:coverage

# Run frontend unit tests
test-frontend:
	cd frontend && npm run test

# Run frontend E2E tests
test-e2e:
	cd frontend && npm run test:e2e

# Run parser tests only
test-parser:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod go test -tags "fts5 sqlite_crypt" -v ./internal/parser/...

# Benchmark parser
bench-parser:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod go test -tags "fts5 sqlite_crypt" -bench=. -benchmem ./internal/parser/...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf frontend/build/
	rm -rf frontend/node_modules/
	rm -rf backend/cmd/server/static/*
	echo '<!DOCTYPE html><html><body>Build frontend first</body></html>' > backend/cmd/server/static/index.html

# Build Docker image
docker:
	docker build -t xelanote:latest .

# Run with Docker Compose
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Development: run backend with hot-reload (Air)
dev:
	@command -v air >/dev/null 2>&1 || { echo "Installing Air..."; go install github.com/air-verse/air@latest; }
	@cd backend && export $$(grep -v '^#' .env | xargs) && $(shell go env GOPATH)/bin/air

# Development: prints instructions for running frontend + backend in parallel.
# This target does NOT start any processes — open two terminals manually.
dev-full:
	@echo "Terminal 1: make dev       (Backend mit Hot-Reload)"
	@echo "Terminal 2: make run-frontend (Frontend mit Vite HMR)"

# Local mobile/iPhone testing helpers (frontend + HTTPS proxy)
phone-setup-https:
	bash scripts/setup-local-phone-https.sh

phone-setup-ip-cert:
	@test -n "$(IP)" || (echo "Usage: make phone-setup-ip-cert IP=10.22.22.60"; exit 1)
	@command -v mkcert >/dev/null 2>&1 || { echo "mkcert not found"; exit 1; }
	@mkdir -p .certs
	mkcert -cert-file .certs/lan-ip.pem -key-file .certs/lan-ip-key.pem $(IP)

phone-frontend-dev:
	cd frontend && npm run dev:phone

phone-frontend-preview:
	cd frontend && npm run build:phone && npm run preview:phone

phone-caddy:
	@command -v caddy >/dev/null 2>&1 || { echo "Caddy not found. Install from https://caddyserver.com/docs/install"; exit 1; }
	caddy run --config $(CURDIR)/Caddyfile.local

phone-help:
	@echo "Local iPhone testing setup"
	@echo "1) make phone-setup-https   (mkcert certs under .certs/)"
	@echo "2) make dev                 (Backend with Air on :8080)"
	@echo "3) make phone-frontend-dev  (Vite HMR on :5173) OR"
	@echo "   make phone-frontend-preview (build + preview on :4173)"
	@echo "4) make phone-caddy         (HTTPS reverse proxy)"
	@echo "5) Open https://dev.xelanote.local:8443 or https://pwa.xelanote.local:8443"
	@echo "   (LAN DNS/hosts/tunnel needed for iPhone name resolution)"
	@echo "   OR use direct IP after: make phone-setup-ip-cert IP=<LAN_IP>"
	@echo "   - PWA preview: https://<LAN_IP>:8443"
	@echo "   - Vite dev:    https://<LAN_IP>:8444"

# Initialize the project (first time setup)
init:
	cd frontend && npm install
	cd backend && go mod download
	@command -v lefthook >/dev/null 2>&1 || command -v $(shell go env GOPATH)/bin/lefthook >/dev/null 2>&1 || { echo "Installing lefthook..."; go install github.com/evilmartians/lefthook@latest; }
	@$(shell go env GOPATH)/bin/lefthook install || lefthook install

# Format code
fmt:
	cd backend && go fmt ./...
	cd frontend && npm run format 2>/dev/null || true

# Check formatting without writing changes
fmt-check:
	@cd backend && files="$$(find . -name '*.go' -not -path './vendor/*')" && \
		if [ -n "$$files" ]; then \
			test -z "$$(gofmt -l $$files)" || (echo "gofmt needed"; exit 1); \
		fi
	@cd frontend && npm run format:check 2>/dev/null || true

# Lint code
lint:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod \
		CGO_ENABLED=1 go vet -tags "fts5 sqlite_crypt" ./...
	cd frontend && npx eslint --max-warnings 0 .

lint-backend:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod \
		CGO_ENABLED=1 go vet -tags "fts5 sqlite_crypt" ./...

lint-frontend:
	cd frontend && npm run lint

lint-md:
	cd frontend && npm run lint:md

typecheck:
	@$(MAKE) typecheck-backend
	@$(MAKE) typecheck-frontend

typecheck-backend:
	@mkdir -p .cache/go-build .cache/go-mod
	cd backend && GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/go-mod \
		CGO_ENABLED=1 go test -run TestNonexistent -tags "fts5 sqlite_crypt" ./...

typecheck-frontend:
	cd frontend && npm run typecheck

lint-golangci:
	@command -v golangci-lint >/dev/null 2>&1 || command -v $(shell go env GOPATH)/bin/golangci-lint >/dev/null 2>&1 || \
		{ echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	cd backend && golangci-lint run ./...

check-policy:
	@echo "=== Policy Checks ==="
	@bash scripts/check-layer-violations.sh
	@bash scripts/check-svelte4-imports.sh
	@bash scripts/check-security-patterns.sh

quality:
	@$(MAKE) fmt-check
	@$(MAKE) lint
	@$(MAKE) lint-md
	@$(MAKE) typecheck
	@$(MAKE) lint-golangci
	@$(MAKE) check-policy

# Generate demo database with sample data (user: demo / demo1234)
demo-db:
	cd backend && CGO_ENABLED=1 go run -tags "fts5" scripts/generate_demo.go -output data/demo.db
	@echo "\nRun with: XELANOTE_DB=data/demo.db make dev"
