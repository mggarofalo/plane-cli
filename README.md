# plane-cli

A command-line interface for the [Plane](https://plane.so) project management API. Designed for both humans and AI agents.

Commands are dynamically generated from the Plane API documentation, so the CLI stays current as the API evolves.

## Install

### From source

```bash
go install github.com/mggarofalo/plane-cli@latest
```

### Build locally

```bash
git clone https://github.com/mggarofalo/plane-cli.git
cd plane-cli
make build    # output: bin/plane
make install  # installs to $GOPATH/bin
```

### Prebuilt binaries

Download from [Releases](https://github.com/mggarofalo/plane-cli/releases) for Linux, macOS, and Windows (amd64/arm64).

## Quick start

```bash
# Authenticate (stores credentials in OS keyring)
plane auth login

# List projects
plane project list --output table

# List work items
plane issue list -p MYPROJECT --output table

# Create a work item
plane issue create -p MYPROJECT --name "Fix login bug" --priority high --state "In Progress"

# Search work items
plane issue search -p MYPROJECT --search "login"

# Get your user info
plane me
```

## Authentication

Credentials are stored securely in the OS keyring (macOS Keychain, GNOME Keyring/libsecret, Windows Credential Manager).

```bash
plane auth login      # Interactive login
plane auth status     # Show current auth info
plane auth logout     # Remove stored credentials
```

### Multiple profiles

```bash
plane auth login                      # Default profile
PLANE_PROFILE=staging plane auth login  # Create "staging" profile
plane auth switch staging             # Switch active profile
```

### Environment variables

| Variable | Description |
|---|---|
| `PLANE_API_KEY` | API token (overrides keyring) |
| `PLANE_URL` | Plane instance base URL |
| `PLANE_WORKSPACE` | Default workspace slug |
| `PLANE_PROFILE` | Override active profile |
| `PLANE_DOCS_URL` | Custom docs base URL |

Resolution order: CLI flag > environment variable > config file > default.

### Session auth (pages)

Some Plane endpoints (e.g., pages) use internal APIs that require session cookies instead of API keys. Copy the `session_id` cookie from your browser dev tools and store it:

```bash
plane auth session
```

## Usage

### Global flags

```
-w, --workspace   Workspace slug
-p, --project     Project ID or identifier (e.g., MYPROJECT)
-o, --output      Output format: json (default), table
    --all         Auto-paginate and return all results
    --per-page    Items per page, 1-100 (default: 100)
    --cursor      Pagination cursor for next page
    --verbose     Debug HTTP logging (tokens redacted)
```

### Output formats

```bash
# JSON (default) - full response, good for scripting
plane issue list -p MYPROJECT

# Table - human-readable with auto-detected columns
plane issue list -p MYPROJECT --output table
```

Table output features:
- Title Case headers
- Short UUIDs (first 8 characters)
- Dates stripped to YYYY-MM-DD
- Numeric columns right-aligned
- Auto-selected columns based on response fields

**stdout/stderr contract:** Data (JSON or table) always goes to stdout. All messages, progress, hints, and errors go to stderr. This means `plane issue list -p X | jq .` always works.

**DELETE commands** print `Deleted.` to stderr and produce no stdout.

### Pagination

```bash
# Get first page (default: 100 items)
plane issue list -p MYPROJECT

# Custom page size
plane issue list -p MYPROJECT --per-page 10

# Fetch all results across pages
plane issue list -p MYPROJECT --all

# Manual cursor-based pagination
plane issue list -p MYPROJECT --per-page 10 --cursor <next_cursor>
```

**Note:** `--all` changes the JSON envelope. Without `--all`, the raw API response is returned (includes `next_cursor`, `prev_cursor`, `next_page_results`). With `--all`, the response is simplified to `{"results": [...], "total_count": N}` with pagination metadata stripped.

### Name resolution

Path and body parameters that accept UUIDs also accept human-readable names. The CLI resolves names to UUIDs automatically:

```bash
# These are equivalent:
plane issue update -p MYPROJECT --work-item-id <uuid> --state <state-uuid>
plane issue update -p MYPROJECT --work-item-id <uuid> --state "In Progress"

# Project identifiers resolve to UUIDs:
plane issue list -p MYPROJECT    # resolved from identifier
plane issue list -p <project-uuid>  # direct UUID
```

Supported: states, labels, cycles, modules, members, projects.

**Note:** If resolution fails (e.g., a typo like `--state "In Progrss"`), the literal string is sent to the API, which will return a 400 error. Resolution does not produce a warning on failure.

### Available resources

| Resource | Commands |
|---|---|
| `project` | list, create, get, update, delete |
| `issue` | list, create, get, search, update, delete |
| `state` | list, create, get, update, delete |
| `label` | list, create, get, update, delete |
| `cycle` | list, create, get, add-work-items, archive, unarchive, delete |
| `module` | list, create, get, add-work-items, archive, unarchive, delete |
| `comment` | list, add, get, update, delete |
| `link` | list, add, get, update, delete |
| `attachment` | list, get, upload, delete |
| `activity` | list |
| `page` | add-workspace, add-project, get-workspace, get-project |
| `intake` | list, add, get, update, delete |
| `worklog` | create, get, update, delete |
| `epic` | list, get |
| `initiative` | list, create, get, update, delete |
| `customer` | list, add, get, link-work-items, delete |
| `teamspace` | list, create, get, update, delete |
| `sticky` | list, add, get, update, delete |
| `member` | list |

### Browsing API docs

```bash
plane docs                    # List all topics
plane docs issue              # List issue endpoints
plane docs issue create       # Show endpoint details
plane docs update             # Refresh docs index
plane docs update-specs       # Pre-cache all endpoint specs
```

## Exit codes

| Code | Meaning | HTTP status |
|------|---------|-------------|
| 0 | Success | 2xx |
| 1 | General error | 5xx / network / unknown |
| 2 | Authentication error | 401, 403 |
| 3 | Not found | 404 |
| 4 | Validation error | 400, 422 |
| 5 | Rate limited | 429 |

Error messages are printed to stderr in the format: `API error {code} ({url}): {body}`

## For automation and AI agents

When scripting or using the CLI from an AI agent:

- **Skip interactive login.** Set `PLANE_API_KEY`, `PLANE_URL`, and `PLANE_WORKSPACE` as environment variables instead of running `plane auth login`.
- **Pre-cache endpoint specs.** Run `plane docs update-specs` once before automating. Without cached specs, commands run in "Mode B" which prints extra stderr messages and lacks full `--help` support.
- **Parse stdout only.** stdout is always clean JSON or table data. All messages, progress hints, and errors go to stderr.
- **Check exit codes.** Use the exit code table above for error handling. Non-zero exit codes indicate specific failure categories.
- **DELETE produces no stdout.** Only prints `Deleted.` to stderr on success.
- **`--all` changes the envelope.** See the pagination section for details on the simplified envelope.
- **Use `--output json`** (the default) for programmatic parsing. Table output is for human consumption.

## Configuration

Config is stored at `~/.config/plane-cli/config.json` (respects `XDG_CONFIG_HOME`).

```json
{
  "active_profile": "default",
  "profiles": {
    "default": {
      "api_url": "https://plane.example.com",
      "workspace": "my-workspace",
      "docs_url": "https://developers.plane.so"
    }
  }
}
```

Endpoint specs are cached under `~/.cache/plane-cli/specs/{profile}/` (respects `XDG_CACHE_HOME`) and refreshed in the background when stale (>24 hours).
