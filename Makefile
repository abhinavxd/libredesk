# Git version for injecting into Go bins.
LAST_COMMIT := $(shell git rev-parse --short HEAD)
LAST_COMMIT_DATE := $(shell git show -s --format=%ci ${LAST_COMMIT})
VERSION := $(shell git describe --tags)
BUILDSTR := ${VERSION} (Commit: ${LAST_COMMIT_DATE} (${LAST_COMMIT}), Build: $(shell date +"%Y-%m-%d %H:%M:%S %z"))

BIN_ARTEMIS := artemis.bin
STATIC := frontend/dist i18n schema.sql
GOPATH ?= $(HOME)/go
STUFFBIN ?= $(GOPATH)/bin/stuffbin

$(STUFFBIN):
	@echo "Installing stuffbin."
	go install github.com/knadh/stuffbin/...
.PHONY: $(STUFFBIN)

.PHONY: $(BIN_ARTEMIS)
$(BIN_ARTEMIS): $(STUFFBIN)
	CGO_ENABLED=0 go build -a -ldflags="-X 'main.buildVersion=${BUILDSTR}' -X 'main.buildDate=${LAST_COMMIT_DATE}' -s -w" -o ${BIN_ARTEMIS} cmd/*.go
	@echo "Build successful. Current build version: $(VERSION)"
	$(STUFFBIN) -a stuff -in ${BIN_ARTEMIS} -out ${BIN_ARTEMIS} ${STATIC}

stuff:
	$(STUFFBIN) -a stuff -in ${BIN_ARTEMIS} -out ${BIN_ARTEMIS} ${STATIC}
.PHONY: stuff

test:
	@go test -v ./...
.PHONY: test

clean:
	@go clean
	-@rm -f ${BIN_ARTEMIS}
.PHONY: clean
