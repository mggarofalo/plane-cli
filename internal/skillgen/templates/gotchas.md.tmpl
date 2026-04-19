# Plane CLI Gotchas

Common failure patterns when using the Plane CLI or MCP tools. Read this before your first call.

## Workspace and Project

- **The workspace slug is NOT your GitHub username.** Discover it first:
  ```bash
  plane project list -w dev
  ```
  If you get a 403, the slug is wrong. Use `--verbose` to see the request URL.

- **`--project` (`-p`) accepts a project identifier (e.g. `PLANECLI`) or UUID.** It does NOT accept a project name. Use `plane project list` to discover identifiers.

- **`--pk` means the resource UUID.** Many endpoints use a generic `--pk` flag for the resource's primary key. Pass the UUID of the specific resource you're operating on. Use `plane <topic> list` first to discover resource UUIDs.

## Name Resolution

- **Name resolution is case-insensitive but exact-match.** `"In Progress"` resolves; `"in progress"` resolves; `"Progress"` does NOT.

- **Resolution cache can be stale.** If you just created a resource and the CLI can't resolve its name, pass the UUID directly or use `--no-resolve`.

- **Sequence IDs are project-scoped.** `PROJ-42` only resolves within project `PROJ`. If you pass `-p OTHER-PROJECT`, the same sequence ID refers to a different issue or doesn't exist.

## Modules and Other Associations

- **`plane issue create` has no `--module-id` flag.** Module attachment is a separate call after creation:
  ```bash
  ID=$(plane issue create --name "Fix login bug" --id-only -w dev -p PROJ)
  plane module add-work-items --module-id "Current Sprint" --issues "$ID" -w dev -p PROJ
  ```
  Skipping this step silently orphans the issue from sprint/roadmap views. Many projects require every issue to have a module — check the project's `AGENTS.md`, `CLAUDE.md`, or other docs for required metadata (modules, labels, priority) before creating issues.

- **Labels are set at create time; modules are not.** `--labels` accepts a comma-separated list on `issue create`. Modules must be attached after.

- **Automation and child issues do not inherit the parent's module.** If a workflow creates follow-up issues programmatically, add an explicit `module add-work-items` step — creation alone does not copy module membership.

## Data Formats

- **`--description` accepts Markdown.** The CLI converts it to HTML before sending. Do NOT pass raw HTML to `--description` — use `--description-html` for that.

- **`--batch` reads JSONL, not a JSON array.** Each line must be a separate JSON object:
  ```bash
  echo '{"name":"Task 1"}
  {"name":"Task 2"}' | plane issue create --batch -w dev -p PROJ
  ```

- **`--all` changes the response envelope.** Without `--all`, list endpoints return the raw API response with pagination metadata. With `--all`, the response is `{"results": [...], "total_count": N}`.

- **`--field` uses dotted path syntax.** Extract nested values with dots: `--field state_detail.name`, not `state_detail->name` or `state_detail['name']`.

## Ensure (Idempotent Upsert)

- **`ensure` matches by exact name.** If a resource named `"Backend"` exists, `--name "backend"` creates a duplicate (case-sensitive match on the API side).

- **`ensure` only works on topics that support it.** Check the Resource Topics table in SKILL.md for which topics have `Ensure: yes`.

## MCP Tools

- **MCP tool names have NO prefix.** The tool for `plane issue create` is `issue_create`, NOT `plane_issue_create`.

- **MCP tools accept `workspace` and `project` params directly.** These override the server defaults. You don't need separate calls to set context.

## Specs and Generation

- **Run `plane docs update-specs` before `plane skills generate`.** Without cached specs, most actions show "No cached spec available" and agents lose parameter documentation.

- **Specs are cached per profile.** If you have multiple Plane instances, each profile has its own spec cache. Generate skills for each profile separately if their APIs differ.
