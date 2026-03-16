# Contributing to plane-cli

This guide is for humans and AI agents contributing to the codebase. It covers architecture, conventions, and how things fit together.

## Build and test

```bash
make build      # Build to bin/plane
make install    # Install to $GOPATH/bin
make test       # go test ./... -v
make lint       # golangci-lint run ./...
make hooks      # Set up git pre-commit hooks (build + test + lint)
```

Go 1.25+ required. Static binaries (`CGO_ENABLED=0`) for release.

### Git hooks

Run `make hooks` to enable the pre-commit hook in `.githooks/`. It runs build, test (with race detector), vet, and golangci-lint before each commit. The hook gracefully skips lint if `golangci-lint` is not installed.

### Live integration tests

The project includes a bash-based live test suite that runs against a real Plane instance:

```bash
bash test_live.sh
```

Requires `plane auth login` to be configured first. The script creates and cleans up test resources (issues, states, labels, cycles, modules, stickies). Pages are skipped if the API returns 404 (session auth not configured). The script exits with a non-zero code equal to the number of failures.

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
    errors.go           APIError type, exit codes (0-5)

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
    cache.go            CacheDir() function, docs index cache (~/.cache/plane-cli/)
    spec_cache.go       Per-command endpoint spec cache (~/.cache/plane-cli/specs/{profile}/)

  models/               Typed data structures
    project.go          Project, ProjectList (with RenderTable)
    user.go             User
    pagination.go       Generic PaginatedResponse[T]

  output/               Output formatting
    formatter.go        Formatter interface, JSON/Table factory
    table.go            WriteTable/WriteTableAligned helpers
    dynamic_table.go    Auto-schema table from JSON responses
```

## Request lifecycle

When a user runs `plane issue create -p FOO --name "bar" --state "Todo"`:

1. **Startup** (`cmd/root.go`): `Execute()` calls `registerDynamicCommands()`, which loads auth config, iterates `docs.DefaultTopics`, loads cached endpoint specs per topic, and calls `cmdgen.BuildTopicCommand()` to register each resource as a Cobra subcommand.

2. **Command dispatch**: Cobra parses global flags (`-p FOO`) and routes to the `issue create` subcommand.

3. **Mode A execution** (`cmdgen/executor.go → ExecuteSpec`):
   - Creates API client via `deps.NewClient()` (resolves auth credentials)
   - Checks workspace/project requirements from the endpoint spec
   - Resolves project identifier `FOO` → UUID via API call (cached per session)
   - Builds URL from the spec's path template, substituting `{workspace_slug}` and `{project_id}`
   - Collects body params from flags: `--name` → `{"name": "bar"}`, `--state` → resolves "Todo" to UUID via `resolveIfNeeded()`
   - Injects `project_id` into body if the spec requires it (`InjectGlobalBodyParams`)
   - Executes `client.Post()` with the URL and body
   - Formats response via `formatResponse()` (JSON or dynamic table)

4. **Post-execution** (`cmd/root.go`): `waitForPendingUpdates()` waits up to 5 seconds for any background stale-spec refresh goroutines to finish.

**Mode B** follows the same flow but first fetches the endpoint spec from the docs site, parses raw `--key=value` args instead of typed Cobra flags, and prints a "hint: spec cached" message to stderr.

## Key design decisions

### Dynamic command generation

Commands are **not** hardcoded. The CLI generates Cobra commands at startup from `docs.DefaultTopics` (hardcoded topic list with endpoint URLs) combined with cached endpoint specs (parsed from Plane's API docs HTML).

Two modes of execution:

- **Mode A (cached specs)**: Endpoint spec is pre-cached. Full flag support with typed flags (`--name`, `--priority`, `--state-id`). Preferred path.
- **Mode B (lazy)**: No cached spec. Fetches spec on first use, falls back to raw `--key=value` argument parsing. Spec is then cached for next time. Prints `Fetching API spec for '...'` to stderr.

Background spec refresh runs with a 5-second timeout at CLI exit (`waitForPendingUpdates`).

### The Deps injection pattern

The `cmdgen` package cannot import `cmd` (circular dependency), but needs access to functions like `NewClient()`, `RequireProject()`, and flag values. The `Deps` struct in `cmdgen/executor.go` solves this via function-pointer injection: `cmd/root.go` creates a `Deps` with pointers to its functions and passes it to `cmdgen.BuildTopicCommand()`. If you add a new dependency that `cmdgen` needs from `cmd`, add it to the `Deps` struct.

### Authentication

Two auth transports:

1. **AuthTransport** (`X-API-Key` header) — standard API key auth for public API endpoints
2. **SessionTransport** (session cookie) — for internal API endpoints (pages). Used via `NewSessionClient()` in `cmd/root.go`. Currently the user must manually invoke `plane auth session` to store a session cookie; no automatic fallback from API key to session auth exists.

Credentials are resolved in order: CLI flag → env var → OS keyring. Tokens are stored as `{profile}/api-key` and `{profile}/session-token` in the keyring. The `Credential` type zeroes token bytes after use.

### Output formatting

Two output paths:

1. **Typed models** (e.g., `ProjectList.RenderTable()`) — used for the few typed service responses
2. **Dynamic table** (`FormatDynamicTable`) — used for all dynamically-generated commands. Auto-detects columns from JSON response fields using a preferred column list.

Dynamic table features: title case headers, UUID shortening (first 8 chars), date-only timestamps, per-column alignment, configurable max cell width per column type.

### Name-to-UUID resolution

Parameters ending in `_id` (plus `state`, `module`, `parent`, `cycle`, `label`) are automatically resolved from human-readable names to UUIDs. Resolution calls the list endpoint for the resource type and matches by `name` or `display_name` (case-insensitive). Resolvable flags are annotated with `(accepts name or UUID)` in per-command `--help` output.

Sequence IDs (`IDENTIFIER-N` format, e.g. `PROJ-42`) are supported for work-item reference flags (`--work-item-id`, `--parent`, `--issues`). These flags are annotated with `(accepts UUID or sequence ID, e.g. PROJ-42)` in help output.

Run `plane resolution` for a full reference covering supported entities, matching fields, cache behavior (soft TTL 1h, hard TTL 7d), and the `--strict`/`--no-resolve` control flags.

**Silent fallback:** If resolution fails (name not found, API error), the original string value is passed through with a warning to stderr. This means typos in names will reach the API as literal strings and produce a 400/422 error. Use `--strict` to treat resolution failures as hard errors (exit code 4).

### Pagination

Plane uses cursor-based pagination. The `--all` flag loops through pages accumulating results, then wraps them in a `{"results": [...], "total_count": N}` envelope (stripping pagination metadata like `next_cursor`). Per-page is clamped to 1-100.

## Conventions

### Code style

- Standard Go formatting (`gofmt`/`goimports`)
- Errors wrap with `fmt.Errorf("context: %w", err)`
- Data output goes to stdout; all messages, hints, and errors go to stderr
- No panics in library code; return errors

### Naming

- CLI flag names: snake_case API params converted to kebab-case (`state_id` → `--state-id`)
- Subcommand names: derived from API doc titles (`Create Work Item` → `create`, `List Archived Modules` → `list-archived`)
- Topic names: singular lowercase (`issue`, `cycle`, `module`)

### Adding a new API resource

Most resources are automatically supported via `DefaultTopics` in `internal/docs/defaults.go`. To add a new resource:

1. Add a `Topic` entry in `DefaultTopics` with the resource name and doc URLs
2. Add word aliases for the topic in `topicAliases()` in `internal/cmdgen/naming.go` (at minimum, add the plural form; this controls how subcommand names are derived from doc titles)
3. Run `plane docs update-specs` to cache the endpoint specs
4. The CLI will generate commands automatically

**Filtered topics:** `OverriddenTopics` in `cmdgen/factory.go` lists topics that are excluded from dynamic generation (currently `introduction` and `user`). If your new topic name collides with a static command, add it to this map and register the command manually instead.

### Adding typed services

For resources that need special handling beyond dynamic commands:

1. Add a service in `internal/api/` (see `projects.go` as a template)
2. Add models in `internal/models/`
3. Implement `TableRenderable` interface for table output
4. Register the command in `cmd/`

### Keeping global flag maps in sync

Two maps define global flags and **must stay in sync**:

- `globalFlagNames` in `cmdgen/factory.go` — used by Mode A to skip spec params that conflict with root persistent flags
- `globalFlags` in `cmdgen/argparse.go` — used by Mode B's raw arg parser to recognize global flags (includes short aliases like `w`, `p`, `o`)

If you add a new persistent flag to `cmd/root.go`, add the flag name to both maps.

### Testing

- Unit tests go next to the code (`*_test.go`)
- Live integration tests go in `test_live.sh`
- Test functions use table-driven patterns where applicable
- No mocks for the API client — unit tests cover parsing/formatting logic; live tests cover API integration

### Environment

- `PLANE_API_KEY`, `PLANE_URL`, `PLANE_WORKSPACE` for CI/scripting
- `--verbose` flag for debugging HTTP issues (tokens are redacted)
- Config at `~/.config/plane-cli/config.json` (respects `XDG_CONFIG_HOME`)
- Endpoint spec cache at `~/.cache/plane-cli/specs/{profile}/` (respects `XDG_CACHE_HOME`)
- Docs index cache at `~/.cache/plane-cli/docs-{profile}.json`

## Common tasks

### Debugging a failing command

```bash
plane issue list -p MYPROJECT --verbose 2>debug.log
```

This logs full HTTP request/response to stderr with tokens redacted.

### Debugging missing flags on a command

If a command has no flags or wrong flags, the endpoint spec may be stale or unparseable:

```bash
# Check what the parser extracted
plane docs issue create

# Re-fetch all specs
plane docs update-specs
```

The endpoint parser (`internal/docs/endpoint_parser.go`) uses regex to extract method, path, and parameters from the Plane docs site HTML. If Plane changes their docs format, the parser may silently produce specs with no parameters.

### Refreshing endpoint specs

```bash
plane docs update          # Refresh topic index
plane docs update-specs    # Re-fetch all endpoint specs
```

Specs are cached under `~/.cache/plane-cli/specs/{profile}/` and auto-refresh in the background when older than 24 hours. If a command has wrong flags or behavior, refreshing specs usually fixes it.

### Testing changes against live Plane

```bash
make build && bash test_live.sh
```

The test script creates temporary resources, validates CRUD operations, and cleans up. It exits with a non-zero code equal to the failure count.
