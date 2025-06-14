# Try to get the commit hash from 1) git 2) the VERSION file 3) fallback.
LAST_COMMIT := $(or $(shell git rev-parse --short HEAD 2> /dev/null),$(shell head -n 1 VERSION | grep -oP -m 1 "^[a-z0-9]+$$"), "")

# Try to get the semver from 1) git 2) the VERSION file 3) fallback.
VERSION := $(or $(LIBREDESK_VERSION),$(shell git describe --tags --abbrev=0 2> /dev/null),$(shell grep -oP 'tag: \Kv\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?' VERSION),"v0.0.0")

BUILDSTR := ${VERSION} (\#${LAST_COMMIT} $(shell date -u +"%Y-%m-%dT%H:%M:%S%z"))

# Binary names and paths
BIN := libredesk
FRONTEND_DIR := frontend
FRONTEND_DIST := ${FRONTEND_DIR}/dist
STATIC := ${FRONTEND_DIST} i18n schema.sql static
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
install-deps: $(STUFFBIN)
	@echo "→ Installing frontend dependencies..."
	@cd ${FRONTEND_DIR} && pnpm install

# Build the frontend for production.
.PHONY: frontend-build
frontend-build: install-deps
	@echo "→ Building frontend for production..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm build

# Run the Go backend server in development mode.
.PHONY: run-backend
run-backend:
	@echo "→ Running backend..."
	CGO_ENABLED=0 go run -ldflags="-s -w -X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'github.com/abhinavxd/libredesk/internal/version.Version=${VERSION}' -X 'main.frontendDir=frontend/dist'" cmd/*.go

# Run the JS frontend server in development mode.
.PHONY: run-frontend
run-frontend:
	@echo "→ Installing frontend dependencies (if not already installed)..."
	@cd ${FRONTEND_DIR} && pnpm install
	@echo "→ Running frontend..."
	@export VITE_APP_VERSION="${VERSION}" && cd ${FRONTEND_DIR} && pnpm dev

# Build the backend binary.
.PHONY: build-backend
build-backend: $(STUFFBIN)
	@echo "→ Building backend..."
	@CGO_ENABLED=0 go build -a \
		-ldflags="-X 'main.buildString=${BUILDSTR}' -X 'main.versionString=${VERSION}' -X 'github.com/abhinavxd/libredesk/internal/version.Version=${VERSION}' -s -w" \
		-o ${BIN} cmd/*.go

# Main build target: builds both frontend and backend, then stuffs static assets into the binary.
.PHONY: build
build: frontend-build build-backend stuff
	@echo "→ Build successful. Current version: $(VERSION)"

# Stuff static assets into the binary using stuffbin.
.PHONY: stuff
stuff: $(STUFFBIN)
	@echo "→ Stuffing static assets into binary..."
	@$(STUFFBIN) -a stuff -in ${BIN} -out ${BIN} ${STATIC}

# Build the application in demo mode.
.PHONY: demo-build
demo-build:
	@echo "→ Building in demo mode..."
	@export VITE_DEMO_BUILD="true" && $(MAKE) build

# Run tests.
.PHONY: test
test:
	@echo "→ Running tests..."
	go test -count=1 ./...
