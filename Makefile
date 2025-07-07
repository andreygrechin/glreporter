.PHONY: all lint format fmt test build coverage-report coverage-report-html security_scan check_clean release-test release

# Build variables
VERSION    := $(shell git describe --tags --always --dirty)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILDTIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
MOD_PATH   := $(shell go list -m)
APP_NAME   := glreporter
GOCOVERDIR := ./covdatafiles

all: lint format test build

lint: fmt
	golangci-lint run --show-stats

format:
	gofumpt -l -w $(shell find . -type f -name "*.go" -not -path "./vendor/*")

fmt: format

test:
	go test ./...

build:
	CGO_ENABLED=0 \
	go build \
		-ldflags \
		"-s \
		-w \
		-X main.Version=$(VERSION) \
		-X main.BuildTime=$(BUILDTIME) \
		-X main.Commit=$(COMMIT)" \
		-o bin/$(APP_NAME)

coverage-report:
	go test -race -coverprofile="${GOCOVERDIR}/coverage.out" ./... && go tool cover -func="${GOCOVERDIR}/coverage.out"

coverage-report-html:
	go test -race -coverprofile="${GOCOVERDIR}/coverage.out" ./... && go tool cover -func="${GOCOVERDIR}/coverage.out"
	go tool cover -html="${GOCOVERDIR}/coverage.out"

security_scan:
	gosec ./...
	govulncheck

check_clean:
	@if [ -n "$(shell git status --porcelain)" ]; then \
		echo "Error: Dirty working tree. Commit or stash changes before proceeding."; \
		exit 1; \
	fi

release-test: lint test security_scan
	goreleaser check
	goreleaser release --snapshot --clean

release: check_clean lint test security_scan
	goreleaser release --clean
