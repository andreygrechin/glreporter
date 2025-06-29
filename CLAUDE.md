# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**glreporter** is a Go CLI tool that asynchronously fetches and displays information about GitLab groups and their associated projects using the GitLab API. It supports recursive traversal of resource hierarchies and concurrent data fetching for performance.

## Development Guidelines

### Core Philosophy

1. **TDD is non-negotiable**: Every single line of production code appears only after a failing go test. Follow the red-green-refactor loop without exception.
2. **Ship in tiny steps**: Keep the workflow runs green: run `make test`, `make build`, `make format`, and `make lint` after every change. The code must compile, pass tests, and conform to formatting at all times.

### Quick Reference – Idiomatic Defaults

1. **Tests**: Standard testing package plus `stretchr/testify` for assertions and `gomock` for mocks.
2. **Concurrency**: Use goroutines and channels for concurrency; sync.Mutex for shared state protection.
3. **Error Handling**: Return (value, error) pairs; use sentinel errors and wrap them with %w. Check errors with errors.Is or errors.As.
4. **Structs**: Prefer concrete types; use small interfaces only at the call-site boundary.
5. **Mutability**: Work with value copies; protect shared state with channels or mutexes; run the race detector regularly.
6. **Types**: Prefer concrete types; declare small interfaces only at the call-site boundary.
7. **Functions**: Keep them short and single-purpose.
8. **Fixtures**: Provide builders that return fully populated structs; allow field overrides via functional options.

### Behavior-Driven Testing

1. Write tests against exported APIs only.
2. Place tests in \*_test.go under the same package name (not a \*_test package) to prevent peeking at internals.
3. Aim for high coverage that reflects real business cases—focus on branches, not vanity numbers.

### Project and Code Structure

1. Packages are lower-snake and single-responsibility.
2. Filenames are simple: foo.go, foo_test.go.
3. Exported identifiers require godoc comments; internal code should read clearly without extra commentary.
4. Define struct tags (json, validate) and validate input at the boundaries with go-playground/validator.
5. Apply the functional-options pattern or plain struct literals for configuration; Go lacks default arguments.

### Error Handling

1. Return (value, error) pairs.
2. Create sentinel errors and wrap them with %w; check with errors.Is or errors.As.
3. Use early returns to fail fast.

### Workflow Checklist

1. Red: write a failing test.
2. Green: add the minimum code to pass.
3. Refactor: improve names, collapse duplication of knowledge (not just code), keep the public API stable.
4. Run `make format`, `make build`, `make test`, and `make lint`.
5. After cleaning all errors, ensure the code is still covered by tests and runs without issues.

## Key Architecture

### Project Structure

- **cmd/**: Contains CLI commands (root, groups, projects, tokens) using spf13/cobra
- **internal/glclient/**: GitLab API client with concurrent fetching capabilities (uses gitlab.com/gitlab-org/api/client-go)
- **internal/output/**: Formatters for table, JSON, and CSV output
- **internal/worker/**: Worker pool implementation for managing concurrent operations
- **main.go**: Entry point with version information injection

### Concurrency Model

- Uses a worker pool pattern with 10 concurrent workers
- Implements recursive fetching with goroutines and sync.Mutex for thread safety
- Handles pagination for API responses (100 items per page)

## Usage

For detailed usage instructions, refer to the [README.md](README.md#usage) file.

## Development Commands

### Building and Running

```shell
# Build the binary with version information
make build

# Run the built binary
./bin/glreporter
```

### Testing and Quality Checks

```shell
# Format the code
make format

# Run linters (includes gofumpt, go vet, staticcheck, golangci-lint)
make lint

# Run all tests
make test

# Security scanning
make vuln

# Coverage analysis
make cov-unit      # Unit test coverage
make cov-integration  # Integration test coverage
```

Note: The `GITLAB_TOKEN` environment variable for live tests will be provided in the working environment.
