---
name: release
description: >
  Automate the release lifecycle: determine next semver from git tags, generate
  grouped release notes from conventional commits, present for approval, create
  an annotated tag, push to trigger the GitHub Actions release workflow, and
  verify the workflow started. Invoke with /release [major|minor|patch].
allowed-tools: Bash, Read, Glob, Grep, AskUserQuestion
---

# Release Skill

Orchestrate a semver release: bump version, generate release notes, tag, push, and verify CI.

## Invocation

```
/release [major|minor|patch]
```

Default bump type is **patch** if omitted.

---

## YOU MUST EXECUTE ALL STEPS BELOW

This is a checklist you must execute, not documentation. Complete every step in order. If you stop early, the release is incomplete.

---

## Step 1: Pre-flight checks

Run all of the following checks before proceeding. If any check fails, stop and tell the user what to fix.

### 1a. Clean working tree

```bash
git status --porcelain
```

If there is any output, **stop** and tell the user:
> Your working tree has uncommitted changes. Please commit or stash them before releasing.

### 1b. Current branch is main

```bash
git branch --show-current
```

If the branch is not `main`, **stop** and tell the user:
> Releases must be created from the main branch. You are on `<branch>`. Switch to main first.

### 1c. Up to date with remote

```bash
git fetch origin main
git rev-list --count HEAD..origin/main
```

If the count is greater than 0, **stop** and tell the user:
> Your local main is behind origin/main by N commit(s). Run `git pull` first.

Also check for unpushed commits:

```bash
git rev-list --count origin/main..HEAD
```

If the count is greater than 0, **stop** and tell the user:
> You have N unpushed commit(s) on main. Run `git push` first, or those commits will not be included in the release tag.

## Step 2: Determine the current version

```bash
git describe --tags --abbrev=0 2>/dev/null
```

- **If a tag exists**: Parse the tag as `vMAJOR.MINOR.PATCH`. Strip the leading `v` for arithmetic.
- **If no tags exist** (first release): Use `v0.0.0` as the base version and inform the user this will be the first release.

## Step 3: Compute the next version

Parse the bump type from the skill argument. Default to `patch` if none was provided.

| Bump type | Rule |
|-----------|------|
| `major` | Increment MAJOR, reset MINOR and PATCH to 0 |
| `minor` | Increment MINOR, reset PATCH to 0 |
| `patch` | Increment PATCH |

The next version string is `vMAJOR.MINOR.PATCH` (with `v` prefix).

Validate that the computed tag does not already exist:

```bash
git tag -l "v<NEXT_VERSION>"
```

If output is non-empty, **stop** and tell the user the tag already exists.

## Step 4: Generate release notes

Collect commits since the last tag (or all commits if this is the first release):

```bash
# If a previous tag exists:
git log <LAST_TAG>..HEAD --oneline --no-merges

# If no previous tag:
git log --oneline --no-merges
```

Group the commits by conventional commit type prefix. Use these categories and ordering:

1. **Breaking Changes** -- commits containing `!:` (e.g., `feat!:`, `fix!:`)
2. **Features** -- commits starting with `feat:` or `feat(scope):`
3. **Bug Fixes** -- commits starting with `fix:` or `fix(scope):`
4. **Performance** -- commits starting with `perf:` or `perf(scope):`
5. **Refactoring** -- commits starting with `refactor:` or `refactor(scope):`
6. **Documentation** -- commits starting with `docs:` or `docs(scope):`
7. **Testing** -- commits starting with `test:` or `test(scope):`
8. **CI/CD** -- commits starting with `ci:` or `ci(scope):`
9. **Chores** -- commits starting with `chore:` or `chore(scope):`
10. **Other** -- commits that do not match any conventional commit prefix

Format each commit as a bullet point: `- <commit message> (<short hash>)`

Strip the type prefix from each message for cleaner display within its category. For example, under "Features", `feat: add batch mode` becomes `- add batch mode (abc1234)`.

Omit any category section that has zero commits.

Build the full release notes as a markdown document:

```markdown
## What's Changed in v<NEXT_VERSION>

### Breaking Changes
- <message> (<hash>)

### Features
- <message> (<hash>)

### Bug Fixes
- <message> (<hash>)

...

**Full Changelog**: <LAST_TAG>...v<NEXT_VERSION>
```

If this is the first release, omit the "Full Changelog" comparison link line.

## Step 5: Present for approval

Display the following to the user and ask for confirmation:

```
=== Release Summary ===

Current version:  <CURRENT_TAG> (or "none" if first release)
Next version:     v<NEXT_VERSION>
Bump type:        <major|minor|patch>
Commits included: <N>

=== Release Notes ===

<generated release notes from Step 4>

=== Action Required ===

This will:
1. Create annotated tag v<NEXT_VERSION>
2. Push the tag to origin (triggers GitHub Actions release workflow)

Proceed with release? [y/N]
```

Wait for the user to confirm. If they say no or want changes, stop and let them adjust.

**Do NOT proceed past this point without explicit user approval.**

## Step 6: Create the annotated tag

```bash
git tag -a "v<NEXT_VERSION>" -m "Release v<NEXT_VERSION>"
```

Verify the tag was created:

```bash
git tag -l "v<NEXT_VERSION>"
```

If the tag does not appear in the output, stop and report the error.

## Step 7: Push the tag

```bash
git push origin "v<NEXT_VERSION>"
```

If the push fails, stop and report the error. Do NOT force-push.

## Step 8: Verify the release workflow

Wait a few seconds for GitHub to register the push event, then check for the workflow run:

```bash
gh run list --workflow=release.yml --limit=5
```

Look for a run triggered by the tag `v<NEXT_VERSION>` with status "in_progress" or "queued". If found, display:

```
Release workflow started successfully!
Run: <URL>
Tag: v<NEXT_VERSION>

The workflow will build binaries for linux/darwin/windows (amd64/arm64),
generate checksums, and create a GitHub Release with auto-generated changelog.

Monitor progress: gh run watch <RUN_ID>
```

If no matching run is found after checking, show the workflow URL and suggest the user check manually:

```
Could not confirm the release workflow started automatically.
Check the Actions tab: gh run list --workflow=release.yml
```

## Done

The release is complete. The tag has been pushed and the GitHub Actions release workflow has been triggered. GoReleaser will handle building binaries, generating checksums, and creating the GitHub Release page.

---

## Edge Cases

- **No existing tags**: Treat as `v0.0.0` base, compute first version accordingly.
- **Dirty working tree**: Abort in Step 1a with a clear message.
- **Unpushed commits**: Abort in Step 1c -- the tag must be on a pushed commit.
- **Behind remote**: Abort in Step 1c -- user must pull first.
- **Tag already exists**: Abort in Step 3 -- user chose a version that is already tagged.
- **Not on main**: Abort in Step 1b -- releases should come from the main branch.
- **Push fails**: Stop in Step 7 and report the error. Do not retry or force-push.
- **Workflow not detected**: Provide manual check instructions in Step 8 rather than failing.

## When to Use

- User says "release", "cut a release", "tag a release", "bump version", "ship it"
- User wants to create a new versioned release of the project

## When NOT to Use

- User wants to modify the GoReleaser config or release workflow (edit those files directly)
- User wants to create a pre-release or release candidate (not supported by this skill)
- User wants to release from a non-main branch (not supported -- releases are from main only)
