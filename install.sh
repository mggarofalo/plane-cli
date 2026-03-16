#!/bin/sh
# install.sh - Cross-platform installer for plane-cli (Linux/macOS)
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/mggarofalo/plane-cli/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/mggarofalo/plane-cli/main/install.sh | sh -s -- --with-mcp
#
# Options:
#   --with-mcp    Configure the Plane MCP server in ~/.claude/settings.json
#   --prefix DIR  Install to DIR instead of ~/.local/bin (e.g., /usr/local/bin)
#   --help        Show this help message
#
# The script is idempotent and safe to re-run for upgrades.

set -eu

REPO="mggarofalo/plane-cli"
BINARY_NAME="plane"
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"

# ── Helpers ──────────────────────────────────────────────────────────────────

usage() {
    cat <<'USAGE'
Usage: install.sh [OPTIONS]

Options:
  --with-mcp    Configure the Plane MCP server in ~/.claude/settings.json
  --prefix DIR  Install to DIR instead of ~/.local/bin
  --help        Show this help message
USAGE
}

info() {
    printf '[plane-cli] %s\n' "$*" >&2
}

error() {
    printf '[plane-cli] ERROR: %s\n' "$*" >&2
    exit 1
}

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1; then
        error "Required command not found: $1"
    fi
}

# ── Parse arguments ──────────────────────────────────────────────────────────

WITH_MCP=false
INSTALL_DIR=""

while [ $# -gt 0 ]; do
    case "$1" in
        --with-mcp)
            WITH_MCP=true
            shift
            ;;
        --prefix)
            if [ $# -lt 2 ]; then
                error "--prefix requires an argument"
            fi
            INSTALL_DIR="$2"
            shift 2
            ;;
        --prefix=*)
            INSTALL_DIR="${1#--prefix=}"
            shift
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# ── Detect platform ─────────────────────────────────────────────────────────

detect_os() {
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported operating system: $os" ;;
    esac
}

detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              error "Unsupported architecture: $arch" ;;
    esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
info "Detected platform: ${OS}/${ARCH}"

# ── Resolve install directory ────────────────────────────────────────────────

if [ -z "$INSTALL_DIR" ]; then
    # Use /usr/local/bin if running as root, otherwise ~/.local/bin
    if [ "$(id -u)" = "0" ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="$DEFAULT_INSTALL_DIR"
    fi
fi

info "Install directory: ${INSTALL_DIR}"

# ── Check dependencies ───────────────────────────────────────────────────────

need_cmd "curl"
need_cmd "tar"

# Determine which SHA-256 tool is available
if command -v sha256sum > /dev/null 2>&1; then
    SHA256_CMD="sha256sum"
elif command -v shasum > /dev/null 2>&1; then
    SHA256_CMD="shasum -a 256"
else
    error "Neither sha256sum nor shasum found. Cannot verify checksums."
fi

# ── Fetch latest version ─────────────────────────────────────────────────────

info "Querying GitHub for latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
RELEASE_JSON="$(curl -fsSL "$RELEASE_URL")" || error "Failed to fetch latest release from GitHub"

# Extract tag name (e.g., "v0.3.0") — works without jq
VERSION="$(printf '%s' "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
if [ -z "$VERSION" ]; then
    error "Could not determine latest version from GitHub API"
fi

# Strip leading 'v' for archive naming
VERSION_NUM="${VERSION#v}"
info "Latest version: ${VERSION} (${VERSION_NUM})"

# ── Download archive and checksums ────────────────────────────────────────────

ARCHIVE_NAME="${BINARY_NAME}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_BASE="https://github.com/${REPO}/releases/download/${VERSION}"
ARCHIVE_URL="${DOWNLOAD_BASE}/${ARCHIVE_NAME}"
CHECKSUMS_URL="${DOWNLOAD_BASE}/checksums.txt"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${ARCHIVE_NAME}..."
curl -fsSL -o "${TMPDIR}/${ARCHIVE_NAME}" "$ARCHIVE_URL" || error "Failed to download archive: ${ARCHIVE_URL}"

info "Downloading checksums.txt..."
curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" || error "Failed to download checksums"

# ── Verify checksum ──────────────────────────────────────────────────────────

info "Verifying checksum..."
EXPECTED_SUM="$(grep -F "${ARCHIVE_NAME}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
if [ -z "$EXPECTED_SUM" ]; then
    error "Archive ${ARCHIVE_NAME} not found in checksums.txt"
fi

ACTUAL_SUM="$(cd "$TMPDIR" && $SHA256_CMD "${ARCHIVE_NAME}" | awk '{print $1}')"
if [ "$EXPECTED_SUM" != "$ACTUAL_SUM" ]; then
    error "Checksum mismatch!\n  Expected: ${EXPECTED_SUM}\n  Got:      ${ACTUAL_SUM}"
fi
info "Checksum verified."

# ── Extract and install ──────────────────────────────────────────────────────

info "Extracting ${BINARY_NAME}..."
tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "$TMPDIR"

if [ ! -f "${TMPDIR}/${BINARY_NAME}" ]; then
    error "Binary '${BINARY_NAME}' not found in archive"
fi

mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
info "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"

# ── Check PATH ───────────────────────────────────────────────────────────────

case ":${PATH}:" in
    *":${INSTALL_DIR}:"*)
        # Already in PATH
        ;;
    *)
        info ""
        info "WARNING: ${INSTALL_DIR} is not in your PATH."
        info "Add it by appending this line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        info ""
        info "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        info ""
        ;;
esac

# ── MCP setup (optional) ─────────────────────────────────────────────────────

configure_mcp() {
    SETTINGS_DIR="${HOME}/.claude"
    SETTINGS_FILE="${SETTINGS_DIR}/settings.json"

    info "Configuring MCP server in ${SETTINGS_FILE}..."

    mkdir -p "$SETTINGS_DIR"

    if [ ! -f "$SETTINGS_FILE" ]; then
        # Create new settings file
        cat > "$SETTINGS_FILE" <<'MCPEOF'
{
  "mcpServers": {
    "plane": {
      "command": "plane",
      "args": [
        "mcp",
        "--quiet"
      ]
    }
  }
}
MCPEOF
        info "Created ${SETTINGS_FILE} with Plane MCP configuration."
        return
    fi

    # File exists — merge the plane MCP server entry
    # Try python3/python first (most reliable for JSON manipulation)
    if command -v python3 > /dev/null 2>&1; then
        PYTHON_CMD="python3"
    elif command -v python > /dev/null 2>&1; then
        PYTHON_CMD="python"
    else
        PYTHON_CMD=""
    fi

    if [ -n "$PYTHON_CMD" ]; then
        $PYTHON_CMD - "$SETTINGS_FILE" <<'PYEOF'
import json
import sys

settings_file = sys.argv[1]
try:
    with open(settings_file, "r") as f:
        settings = json.load(f)
except (json.JSONDecodeError, ValueError):
    settings = {}

if not isinstance(settings, dict):
    settings = {}

if "mcpServers" not in settings or not isinstance(settings["mcpServers"], dict):
    settings["mcpServers"] = {}

settings["mcpServers"]["plane"] = {
    "command": "plane",
    "args": ["mcp", "--quiet"]
}

with open(settings_file, "w") as f:
    json.dump(settings, f, indent=2)
    f.write("\n")
PYEOF
        info "Merged Plane MCP configuration into ${SETTINGS_FILE}."
    elif command -v jq > /dev/null 2>&1; then
        # Fall back to jq
        TMP_SETTINGS="$(mktemp)"
        jq '.mcpServers.plane = {"command": "plane", "args": ["mcp", "--quiet"]}' "$SETTINGS_FILE" > "$TMP_SETTINGS"
        mv "$TMP_SETTINGS" "$SETTINGS_FILE"
        info "Merged Plane MCP configuration into ${SETTINGS_FILE}."
    else
        info "WARNING: Neither python nor jq found. Cannot merge MCP config."
        info "Please manually add the following to ${SETTINGS_FILE}:"
        info '  "mcpServers": { "plane": { "command": "plane", "args": ["mcp", "--quiet"] } }'
    fi
}

if [ "$WITH_MCP" = "true" ]; then
    configure_mcp
fi

# ── Initialize spec cache ────────────────────────────────────────────────────

# Ensure the installed binary is found (add install dir to PATH for this session)
export PATH="${INSTALL_DIR}:${PATH}"

if command -v plane > /dev/null 2>&1; then
    info "Initializing API spec cache..."
    plane docs update-specs 2>/dev/null || info "Spec cache initialization skipped (authentication may be required)."
else
    info "Skipping spec cache initialization (binary not in PATH for this session)."
fi

# ── Done ──────────────────────────────────────────────────────────────────────

info ""
info "plane-cli ${VERSION} installed successfully!"
if [ "$WITH_MCP" = "true" ]; then
    info "MCP server configured. Restart Claude Code to activate."
fi
info ""
info "Get started:"
info "  plane auth login"
info "  plane --help"
