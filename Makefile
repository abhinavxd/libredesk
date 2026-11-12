# Try to get the commit hash from 1) git 2) the VERSION file 3) fallback.
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),$(shell head -n 1 VERSION | grep -oP -m 1 "^[a-z0-9]+$$"), "")

# Try to get the semver from 1) git 2) the VERSION file 3) fallback.
VERSION := $(or $(LIBREDESK_VERSION),$(shell git describe --tags --abbrev=0 2> /dev/null),$(shell grep -oP 'tag: \Kv\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?' VERSION),"v0.0.0")

BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

# Binary names and paths
BIN := libredesk
FRONTEND_DIR := frontend
FRONTEND_DIST := ${FRONTEND_DIR}/dist
WIDGET_JS_MIN := static/widget.min.js
STATIC := ${FRONTEND_DIST} schema.sql static/email-templates static/public ${WIDGET_JS_MIN}:static/widget.js
GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

# The default target to run when `make` is executed.
.DEFAULT_GOAL := build

# Install stuffbin if it doesn't exist.
$(STUFFBIN):
	@echo "→ Installing stuffbin..."
	@go install github.com/knadh/stuffbin/...

# Install dependencies for both backend and frontend.
.PHONY: install-deps
install-deps: $(STUFFBIN) install-deps-frontend

.PHONY: install-deps-frontend
install-deps-frontend:
	@echo "→ Installing frontend dependencies..."
	@cd ${FRONTEND_DIR} && pnpm install

# Build the frontend for production (both apps).
.PHONY: frontend-build
frontend-build: frontend-build-main frontend-build-widget

.PHONY: minify-widget-js
minify-widget-js: install-deps
	@echo "→ Minifying widget loader..."
	@cd ${FRONTEND_DIR} && pnpm build:widget-loader

# Build only the main frontend app.
.PHONY: frontend-build-main
frontend-build-main: install-deps
	@echo "→ Building main frontend app for production..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm build:main

# Build only the widget frontend app.
.PHONY: frontend-build-widget
frontend-build-widget: install-deps
	@echo "→ Building widget frontend app for production..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm build:widget

# Run the Go backend server in development mode.
.PHONY: run-backend
run-backend: i18n-compile
	@echo "→ Running backend..."
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'github.com/abhinavxd/libredesk/internal/version.Version=${VERSION}' -X 'main.frontendDir=frontend/dist'" ./cmd/

# Run the JS frontend server in development mode (main app only).
.PHONY: run-frontend
run-frontend:
	@echo "→ Installing frontend dependencies (if not already installed)..."
	@cd ${FRONTEND_DIR} && pnpm install
	@echo "→ Running main frontend app..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm dev:main

# Run the main frontend app in development mode.
.PHONY: run-frontend-main
run-frontend-main:
	@echo "→ Installing frontend dependencies (if not already installed)..."
	@cd ${FRONTEND_DIR} && pnpm install
	@echo "→ Running main frontend app..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm dev:main

# Run the widget frontend app in development mode.
.PHONY: run-frontend-widget
run-frontend-widget:
	@echo "→ Installing frontend dependencies (if not already installed)..."
	@cd ${FRONTEND_DIR} && pnpm install
	@echo "→ Running widget frontend app..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm dev:widget

# Build the backend binary.
.PHONY: build-backend
build-backend: $(STUFFBIN) i18n-compile
	@echo "→ Building backend..."
	@CGO_ENABLED=0 go build \
		-ldflags="-X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'github.com/abhinavxd/libredesk/internal/version.Version=${VERSION}' -s -w" \
		-o ${BIN} ./cmd

# Main build target: builds both frontend and backend, then stuffs static assets into the binary.
.PHONY: build
build: frontend-build build-backend
	@$(MAKE) --no-print-directory stuff
	@echo "→ Build successful. Current version: $(VERSION)"

# Stuff static assets into the binary using stuffbin.
.PHONY: stuff
stuff: $(STUFFBIN) minify-widget-js
	@echo "→ Stuffing static assets into binary..."
	@$(STUFFBIN) -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# Build the application in demo mode.
.PHONY: demo-build
demo-build:
	@echo "→ Building in demo mode..."
	@export VITE_DEMO_BUILD="true" && $(MAKE) build

# Run tests. Integration tests need a Postgres; start one with `make test-db` first, else they skip.
.PHONY: test
test: i18n-compile
	@echo "→ Running Go tests..."
	go test -count=1 ./...
	@echo "→ Running frontend tests..."
	cd ${FRONTEND_DIR} && npx pnpm install --frozen-lockfile && npx pnpm test:run

# Start a throwaway Postgres for integration tests. Remove it with `make test-db-down`.
.PHONY: test-db
test-db:
	docker run -d --name libredesk-test-db -p 127.0.0.1:5433:5432 \
		-e POSTGRES_USER=libredesk -e POSTGRES_PASSWORD=libredesk -e POSTGRES_DB=libredesk \
		postgres:17-alpine

.PHONY: test-db-down
test-db-down:
	docker rm -f libredesk-test-db

# Go catalogs compile to MO; Lingui frontend catalogs compile to JSON.
BACKEND_PO := $(wildcard locales/*/LC_MESSAGES/backend.po locales/*/LC_MESSAGES/templates.po)
FRONTEND_PO := $(wildcard locales/*/LC_MESSAGES/frontend.po)
BACKEND_MO := $(BACKEND_PO:.po=.mo)

.PHONY: i18n-extract
i18n-extract: i18n-extract-backend i18n-extract-frontend

.PHONY: i18n-extract-backend
i18n-extract-backend:
	@set -e; for domain in backend templates; do \
		go run ./scripts/gettext -domain $$domain > locales/$$domain.pot.tmp; \
		mv locales/$$domain.pot.tmp locales/$$domain.pot; \
	done

.PHONY: i18n-extract-frontend
i18n-extract-frontend:
	@cd ${FRONTEND_DIR} && node scripts/gettext.mjs extract

.PHONY: i18n-update
i18n-update: i18n-update-backend i18n-update-frontend

.PHONY: i18n-update-backend
i18n-update-backend: i18n-extract-backend
	@set -e; for po in $(BACKEND_PO); do \
		domain=$$(basename "$$po" .po); \
		# Disable fuzzy matching: a changed English message needs a reviewed translation. \
		msgmerge --quiet --no-fuzzy-matching --no-wrap "$$po" locales/$$domain.pot -o "$$po.merge"; \
		msgattrib --no-obsolete --no-wrap "$$po.merge" -o "$$po"; \
		rm "$$po.merge"; \
	done

.PHONY: i18n-update-frontend
i18n-update-frontend: i18n-extract-frontend
	@set -e; for po in $(FRONTEND_PO); do \
		# Disable fuzzy matching: a changed English message needs a reviewed translation. \
		msgmerge --quiet --no-fuzzy-matching --no-wrap "$$po" locales/frontend.pot -o "$$po.merge"; \
		msgattrib --no-obsolete --no-wrap "$$po.merge" -o "$$po"; \
		rm "$$po.merge"; \
	done

.PHONY: i18n-compile
i18n-compile: i18n-compile-backend i18n-compile-frontend

.PHONY: i18n-compile-backend
i18n-compile-backend: $(BACKEND_MO)

.PHONY: i18n-compile-frontend
i18n-compile-frontend: install-deps-frontend
	@cd ${FRONTEND_DIR} && node scripts/gettext.mjs compile

locales/%/LC_MESSAGES/backend.mo: locales/%/LC_MESSAGES/backend.po
	msgfmt --check --check-format -o $@ $<

locales/%/LC_MESSAGES/templates.mo: locales/%/LC_MESSAGES/templates.po
	msgfmt --check --check-format -o $@ $<

# Read-only verification of source coverage, PO validity and generated MO/JSON files.
.PHONY: i18n-check
i18n-check: i18n-check-backend i18n-check-frontend

.PHONY: i18n-check-backend
i18n-check-backend:
	@set -e; for domain in backend templates; do \
		go run ./scripts/gettext -domain $$domain | cmp - locales/$$domain.pot; \
	done; \
	for po in $(BACKEND_PO); do \
		domain=$$(basename "$$po" .po); \
		msgfmt --check --check-format -o /dev/null "$$po"; \
		msgcmp --use-untranslated "$$po" locales/$$domain.pot; \
		msgcmp --use-untranslated locales/$$domain.pot "$$po"; \
		msgfmt -o - "$$po" | cmp - "$${po%.po}.mo"; \
	done

.PHONY: i18n-check-frontend
i18n-check-frontend:
	@cd ${FRONTEND_DIR} && node scripts/gettext.mjs check
