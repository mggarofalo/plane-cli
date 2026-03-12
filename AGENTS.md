# Contributing to plane-cli

This guide is for humans and AI agents contributing to the codebase. It covers architecture, conventions, and how things fit together.

## Build and test

```bash
make build      # Build to bin/plane
make install    # Install to $GOPATH/bin
make test       # go test ./... -v
make lint       # golangci-lint run ./...
```

Go 1.25+ required. Static binaries (`CGO_ENABLED=0`) for release.

### Live integration tests

The project includes a bash-based live test suite that runs against a real Plane instance:

```bash
bash test_live.sh
```

Requires `plane auth login` to be configured first. The script creates and cleans up test resources (issues, states, labels, cycles, modules, stickies). Pages are skipped if the API returns 404 (session auth not configured).

## Architecture

```
cmd/                    Cobra command definitions
  root.go               Root command, global flags, dynamic command registration
  auth.go               auth login/logout/status/switch/session
  me.go                 Current user info
  docs.go               API documentation browser
  version.go            Version/commit/date info

internal/
  api/                  HTTP client layer
    client.go           Core HTTP client (Get, Post, Patch, Put, Delete)
    transport.go        AuthTransport — injects X-API-Key header
    session.go          SessionTransport — injects session cookie
    pagination.go       Cursor-based pagination (PaginationParams, AutoPaginate)
    projects.go         Typed ProjectsService
    users.go            Typed UsersService
    errors.go           APIError type, exit codes

  auth/                 Authentication and configuration
    config.go           Config file (~/.config/plane-cli/config.json)
    credential.go       Credential type with zeroing
    keyring.go          OS keyring integration (SecretStore interface)
    resolver.go         Credential resolution chain (flag → env → keyring)
    env.go              Environment variable names
    profile.go          Multi-profile management

  cmdgen/               Dynamic command generation
    factory.go          Builds Cobra commands from endpoint specs
    executor.go         Runs API calls from specs + flags
    naming.go           Derives subcommand names and flag names from API params
    argparse.go         Parses CLI arguments for Mode B (lazy spec loading)
    resolver.go         Name-to-UUID resolution (states, labels, members, etc.)

  docs/                 API documentation system
    defaults.go         Hardcoded topic/entry definitions (fallback)
    registry.go         Topic index loading and management
    fetch.go            HTTP fetching with local caching
    endpoint_parser.go  Extracts EndpointSpec from parsed docs pages
    cache.go            File-based cache under ~/.config/plane-cli/
    spec_cache.go       Per-command endpoint spec cache

  models/               Typed data structures
    project.go          Project, ProjectList (with RenderTable)
    user.go             User
    pagination.go       Generic PaginatedResponse[T]

  output/               Output formatting
    formatter.go        Formatter interface, JSON/Table factory
    table.go            WriteTable/WriteTableAligned helpers
    dynamic_table.go    Auto-schema table from JSON responses
```

## Key design decisions

### Dynamic command generation

Commands are **not** hardcoded. The CLI generates Cobra commands at startup from `docs.DefaultTopics` (hardcoded topic list with endpoint URLs) combined with cached endpoint specs (parsed from Plane's API docs HTML).

Two modes of execution:

- **Mode A (cached specs)**: Endpoint spec is pre-cached. Full flag support with typed flags (`--name`, `--priority`, `--state-id`). Preferred path.
- **Mode B (lazy)**: No cached spec. Fetches spec on first use, falls back to raw `--key=value` argument parsing. Spec is then cached for next time.

Background spec refresh runs with a 5-second timeout at CLI exit (`waitForPendingUpdates`).

### Authentication

Two auth transports:

1. **AuthTransport** (`X-API-Key` header) — standard API key auth for public API endpoints
2. **SessionTransport** (session cookie) — for internal API endpoints (pages)

Credentials are resolved in order: CLI flag → env var → OS keyring. Tokens are stored as `{profile}/api-key` and `{profile}/session-token` in the keyring. The `Credential` type zeroes token bytes after use.

### Output formatting

Two output paths:

1. **Typed models** (e.g., `ProjectList.RenderTable()`) — used for the few typed service responses
2. **Dynamic table** (`FormatDynamicTable`) — used for all dynamically-generated commands. Auto-detects columns from JSON response fields using a preferred column list.

Dynamic table features: title case headers, UUID shortening (first 8 chars), date-only timestamps, per-column alignment, configurable max cell width per column type.

### Name-to-UUID resolution

Parameters ending in `_id` (plus `state`, `module`, `parent`, `cycle`, `label`) are automatically resolved from human-readable names to UUIDs. Resolution calls the list endpoint for the resource type and matches by `name` or `display_name` (case-insensitive).

### Pagination

Plane uses cursor-based pagination. The `--all` flag loops through pages accumulating results, then wraps them in a `{results: [...], total_count: N}` envelope. Per-page is clamped to 1-100.

## Conventions

### Code style

- Standard Go formatting (`gofmt`/`goimports`)
- Errors wrap with `fmt.Errorf("context: %w", err)`
- User-facing messages go to stderr; data output goes to stdout
- No panics in library code; return errors

### Naming

- CLI flag names: snake_case API params converted to kebab-case (`state_id` → `--state-id`)
- Subcommand names: derived from API doc titles (`Create Work Item` → `create`, `List Archived Modules` → `list-archived`)
- Topic names: singular lowercase (`issue`, `cycle`, `module`)

### Adding a new API resource

Most resources are automatically supported via `DefaultTopics` in `internal/docs/defaults.go`. To add a new resource:

1. Add a `Topic` entry in `DefaultTopics` with the resource name and doc URLs
2. Run `plane docs update-specs` to cache the endpoint specs
3. The CLI will generate commands automatically

No Go code changes needed for standard CRUD resources.

### Adding typed services

For resources that need special handling beyond dynamic commands:

1. Add a service in `internal/api/` (see `projects.go` as a template)
2. Add models in `internal/models/`
3. Implement `TableRenderable` interface for table output
4. Register the command in `cmd/`

### Testing

- Unit tests go next to the code (`*_test.go`)
- Live integration tests go in `test_live.sh`
- Test functions use table-driven patterns where applicable
- No mocks for the API client — unit tests cover parsing/formatting logic; live tests cover API integration

### Environment

- `PLANE_API_KEY`, `PLANE_URL`, `PLANE_WORKSPACE` for CI/scripting
- `--verbose` flag for debugging HTTP issues (tokens are redacted)
- Config at `~/.config/plane-cli/config.json` (respects `XDG_CONFIG_HOME`)
- Endpoint spec cache at `~/.config/plane-cli/docs/`

## Common tasks

### Debugging a failing command

```bash
plane issue list -p MYPROJECT --verbose 2>debug.log
```

This logs full HTTP request/response to stderr with tokens redacted.

### Refreshing endpoint specs

```bash
plane docs update          # Refresh topic index
plane docs update-specs    # Re-fetch all endpoint specs
```

Specs are cached locally. If a command has wrong flags or behavior, refreshing specs usually fixes it.

### Testing changes against live Plane

```bash
make build && bash test_live.sh
```

The test script creates temporary resources, validates CRUD operations, and cleans up. It exits with a non-zero code if any test fails.
