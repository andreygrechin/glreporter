.PHONY: all build format fmt lint vuln test release check_clean bump_patch bump_minor bump_major

# Build variables
VERSION    := $(shell git describe --tags --always --dirty)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILDTIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
MOD_PATH   := $(shell go list -m)
APP_NAME   := glreporter
GOCOVERDIR := ./covdatafiles

# Build targets
all: lint test build

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

format:
	gofumpt -l -w .

fmt: format

lint: fmt
	go vet ./...
	staticcheck ./...
	golangci-lint run --show-stats

vuln:
	gosec ./...
	govulncheck

cov-integration:
	rm -fr "${GOCOVERDIR}" && mkdir -p "${GOCOVERDIR}"
	go build \
		-ldflags \
		"-s \
		-w \
		-X $(MOD_PATH)/internal/config.Version=$(VERSION) \
		-X $(MOD_PATH)/internal/config.BuildTime=$(BUILDTIME) \
		-X $(MOD_PATH)/internal/config.Commit=$(COMMIT)" \
		-o bin/$(APP_NAME) \
		-cover
	go tool covdata percent -i=covdatafiles

cov-unit:
	rm -fr "${GOCOVERDIR}" && mkdir -p "${GOCOVERDIR}"
	go test -coverprofile="${GOCOVERDIR}/cover.out" ./...
	go tool cover -func="${GOCOVERDIR}/cover.out"
	go tool cover -html="${GOCOVERDIR}/cover.out"
	go tool cover -html="${GOCOVERDIR}/cover.out" -o "${GOCOVERDIR}/coverage.html"

test:
	go test ./...

check_clean:
	@if [ -n "$(shell git status --porcelain)" ]; then \
		echo "Error: Dirty working tree. Commit or stash changes before proceeding."; \
		exit 1; \
	fi

release-test: lint test vuln
	goreleaser check
	goreleaser release --snapshot --clean

release: check_clean lint test vuln
	goreleaser release --clean

define bump_version
	$(eval NEW_VERSION := $(shell gosemver bump $(1) $(VERSION)))
	@echo "Old version $(VERSION)"
	@echo "Bumped to version $(NEW_VERSION)"
	@git tag -a "v$(NEW_VERSION)" -m "v$(NEW_VERSION)"
	@git push origin "v$(NEW_VERSION)"
endef

bump_patch:
	$(call bump_version,patch)

bump_minor:
	$(call bump_version,minor)

bump_major:
	$(call bump_version,major)
