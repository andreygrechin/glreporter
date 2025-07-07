# Development Guidelines

## Build & Test Commands

- Build Go projects: `make build`
- Run tests: `make test`
- Run specific test: `go test -run TestName ./path/to/package`
- Run tests with coverage: `go test -cover ./...`
- Run linting: `make lint`
- Format code: `make format`
- Run code generation: `go generate ./...`
- Coverage report: `go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
- On completion, use formatting (`make format`), tests (`make test`), and code generation (`make generate`)
- Never commit without running completion sequence

## Important Workflow Notes

- Always run tests, format, and linter
- For linter use `make lint`
- Run tests, format, and linter after making significant changes to verify functionality
- Go version: 1.24+
- Don't add "Generated with Claude Code" or "Co-Authored-By: Claude" to commit messages or PRs
- Do not include "Test plan" sections in PR descriptions
- Do not add comments that describe changes, progress, or historical modifications. Avoid comments like "new function," "added test," "now we changed this," or "previously used X, now using Y." Comments should only describe the current state and purpose of the code, not its history or evolution.
- Use `go:generate` for generating mocks, never modify generated files manually. Mocks are generated with `moq` and stored in the `mocks` package.
- After important functionality added, update README.md accordingly
- Always write unit tests instead of manual testing
- Don't manually test by running servers and using curl - write comprehensive unit tests instead

## Code Style Guidelines

- The codebase follows standard Go conventions. When in doubt, check the [Effective Go](https://golang.org/doc/effective_go) guide.
- Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use snake_case for filenames, camelCase for variables, PascalCase for exported names
- Group imports: standard library, then third-party, then local packages
- Error handling: check errors immediately and return them with context
- Use meaningful variable names; avoid single-letter names except in loops
- Validate function parameters at the start before processing
- Return early when possible to avoid deep nesting
- Prefer composition over inheritance
- Interfaces: Define interfaces in consumer packages
- Function size preferences:
  - Aim for functions around 50-60 lines when possible
  - Don't break down functions too small as it can reduce readability
  - Maintain focus on a single responsibility per function
- Comment style: in-function comments should be lowercase sentences
- Code width: keep lines under 120 characters when possible
- Format: Use `make format`
- Use existing structs from lower-level packages directly, don't duplicate them
  - When a struct is already defined in a lower-level package, use it directly instead of creating a duplicate definition
- Never add comments explaining what interface a struct implements - this is client-side concern
  - Don't write comments like "implements the Fetcher interface" - the consumer of the interface decides what implements it, not the provider
- In any file with structs and methods, order should be:
    1. Structs with methods first
    2. Interfaces after
    3. Data structs after

### Error Handling

- Use `fmt.Errorf("context: %w", err)` to wrap errors with context
- Check errors immediately after function calls
- Return detailed error information through wrapping

### Comments

- All comments inside functions should be lowercase
- Document all exported items with proper casing
- Use inline comments for complex logic
- Start comments with the name of the thing being described

### Testing

- Use table-driven tests where appropriate
- Use subtest with `t.Run()` to make test more structured
- Use `require` for fatal assertions, `assert` for non-fatal ones
- Use mock interfaces for dependency injection
- Test names follow pattern: `Test<Type>_<method>`

## Libraries

- Logging: `log/slog`
- CLI commands: `github.com/spf13/cobra`
- GitLab API client: `gitlab.com/gitlab-org/api/client-go`
- HTTP/REST: `github.com/go-resty/resty`
- Middleware: `github.com/didip/tollbooth/v8`
- Database: `github.com/jmoiron/sqlx` with `modernc.org/sqlite`
- Testing: `github.com/stretchr/testify`
- Testing helpers: `github.com/go-pkgz/testutils`
- Mock generation: `github.com/matryer/moq`
- OpenAI: `github.com/sashabaranov/go-openai`
- Frontend: HTMX v2. Try to avoid using JS.
- For containerized tests use `github.com/go-pkgz/testutils`
- To access libraries, figure how to use and check their documentation, use `go doc` command and `gh` tool
