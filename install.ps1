<#
.SYNOPSIS
    Installs plane-cli on Windows.

.DESCRIPTION
    Downloads the latest plane-cli release from GitHub, verifies the checksum,
    extracts the binary, adds it to PATH, and optionally configures the MCP
    server for Claude Code.

    This script is idempotent and safe to re-run for upgrades.

.PARAMETER WithMcp
    Configure the Plane MCP server in ~/.claude/settings.json.

.PARAMETER InstallDir
    Install to a custom directory instead of $env:LOCALAPPDATA\plane-cli.

.EXAMPLE
    # Basic install
    irm https://raw.githubusercontent.com/mggarofalo/plane-cli/main/install.ps1 | iex

    # Install with MCP setup
    & ([scriptblock]::Create((irm https://raw.githubusercontent.com/mggarofalo/plane-cli/main/install.ps1))) -WithMcp

.LINK
    https://github.com/mggarofalo/plane-cli
#>

[CmdletBinding()]
param(
    [switch]$WithMcp,
    [string]$InstallDir
)

$ErrorActionPreference = 'Stop'

$Repo = 'mggarofalo/plane-cli'
$BinaryName = 'plane.exe'

# ── Helpers ──────────────────────────────────────────────────────────────────

function Write-Info {
    param([string]$Message)
    Write-Host "[plane-cli] $Message" -ForegroundColor Cyan
}

function Write-Err {
    param([string]$Message)
    Write-Host "[plane-cli] ERROR: $Message" -ForegroundColor Red
    exit 1
}

# ── Detect architecture ─────────────────────────────────────────────────────

$RawArch = $env:PROCESSOR_ARCHITECTURE
switch ($RawArch) {
    'AMD64'  { $Arch = 'amd64' }
    'ARM64'  { $Arch = 'arm64' }
    default  { Write-Err "Unsupported architecture: $RawArch" }
}

Write-Info "Detected platform: windows/$Arch"

# ── Resolve install directory ────────────────────────────────────────────────

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'plane-cli'
}

Write-Info "Install directory: $InstallDir"

# ── Fetch latest version ─────────────────────────────────────────────────────

Write-Info 'Querying GitHub for latest release...'
$ReleaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
try {
    $Release = Invoke-RestMethod -Uri $ReleaseUrl -UseBasicParsing
} catch {
    Write-Err "Failed to fetch latest release from GitHub: $_"
}

$Version = $Release.tag_name
if (-not $Version) {
    Write-Err 'Could not determine latest version from GitHub API'
}

# Strip leading 'v' for archive naming
$VersionNum = $Version.TrimStart('v')
Write-Info "Latest version: $Version ($VersionNum)"

# ── Download archive and checksums ────────────────────────────────────────────

$ArchiveName = "plane_${VersionNum}_windows_${Arch}.zip"
$DownloadBase = "https://github.com/$Repo/releases/download/$Version"
$ArchiveUrl = "$DownloadBase/$ArchiveName"
$ChecksumsUrl = "$DownloadBase/checksums.txt"

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Info "Downloading $ArchiveName..."
    $ArchivePath = Join-Path $TmpDir $ArchiveName
    Invoke-WebRequest -Uri $ArchiveUrl -OutFile $ArchivePath -UseBasicParsing

    Write-Info 'Downloading checksums.txt...'
    $ChecksumsPath = Join-Path $TmpDir 'checksums.txt'
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing

    # ── Verify checksum ──────────────────────────────────────────────────────

    Write-Info 'Verifying checksum...'
    $ChecksumLines = Get-Content $ChecksumsPath
    $ExpectedLine = $ChecksumLines | Where-Object { $_ -match [regex]::Escape($ArchiveName) } | Select-Object -First 1
    if (-not $ExpectedLine) {
        Write-Err "Archive $ArchiveName not found in checksums.txt"
    }

    $ExpectedSum = ($ExpectedLine -split '\s+')[0]
    $ActualSum = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash.ToLower()

    if ($ExpectedSum -ne $ActualSum) {
        Write-Err "Checksum mismatch!`n  Expected: $ExpectedSum`n  Got:      $ActualSum"
    }
    Write-Info 'Checksum verified.'

    # ── Extract and install ──────────────────────────────────────────────────

    Write-Info "Extracting $BinaryName..."
    $ExtractDir = Join-Path $TmpDir 'extract'
    Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir -Force

    $BinaryPath = Get-ChildItem -Path $ExtractDir -Filter $BinaryName -Recurse | Select-Object -First 1
    if (-not $BinaryPath) {
        Write-Err "Binary '$BinaryName' not found in archive"
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $DestPath = Join-Path $InstallDir $BinaryName
    Copy-Item -Path $BinaryPath.FullName -Destination $DestPath -Force
    Write-Info "Installed $BinaryName to $DestPath"

    # ── Update PATH ──────────────────────────────────────────────────────────

    $UserPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($UserPath -split ';' | Where-Object { $_ -eq $InstallDir }) {
        Write-Info "$InstallDir is already in your PATH."
    } else {
        Write-Info "Adding $InstallDir to your user PATH..."
        $NewPath = if ($UserPath) { "$UserPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable('PATH', $NewPath, 'User')
        # Also update the current session so plane is available immediately
        $env:PATH = "$env:PATH;$InstallDir"
        Write-Info 'PATH updated. You may need to restart your terminal for it to take effect.'
    }

    # ── MCP setup (optional) ─────────────────────────────────────────────────

    if ($WithMcp) {
        $ClaudeDir = Join-Path $HOME '.claude'
        $SettingsFile = Join-Path $ClaudeDir 'settings.json'

        Write-Info "Configuring MCP server in $SettingsFile..."

        if (-not (Test-Path $ClaudeDir)) {
            New-Item -ItemType Directory -Path $ClaudeDir -Force | Out-Null
        }

        $PlaneServer = [ordered]@{
            command = 'plane'
            args    = @('mcp', '--quiet')
        }

        if (Test-Path $SettingsFile) {
            # Read and merge
            try {
                $Settings = Get-Content $SettingsFile -Raw | ConvertFrom-Json
            } catch {
                Write-Info "WARNING: Could not parse existing settings.json, creating fresh."
                $Settings = $null
            }

            if (-not $Settings) {
                $Settings = [PSCustomObject]@{}
            }

            # Ensure mcpServers property exists
            if (-not ($Settings.PSObject.Properties.Name -contains 'mcpServers')) {
                $Settings | Add-Member -NotePropertyName 'mcpServers' -NotePropertyValue ([PSCustomObject]@{})
            }

            # Add or update the plane server
            $McpServers = $Settings.mcpServers
            if ($McpServers.PSObject.Properties.Name -contains 'plane') {
                $McpServers.plane = [PSCustomObject]$PlaneServer
            } else {
                $McpServers | Add-Member -NotePropertyName 'plane' -NotePropertyValue ([PSCustomObject]$PlaneServer)
            }

            # Write UTF-8 without BOM (PS 5.x's -Encoding UTF8 emits a BOM which breaks JSON parsers)
            $Utf8NoBom = [System.Text.UTF8Encoding]::new($false)
            [System.IO.File]::WriteAllText($SettingsFile, (($Settings | ConvertTo-Json -Depth 10) + "`n"), $Utf8NoBom)
            Write-Info "Merged Plane MCP configuration into $SettingsFile."
        } else {
            # Create new file
            $Settings = [PSCustomObject]@{
                mcpServers = [PSCustomObject]@{
                    plane = [PSCustomObject]$PlaneServer
                }
            }
            # Write UTF-8 without BOM (PS 5.x's -Encoding UTF8 emits a BOM which breaks JSON parsers)
            $Utf8NoBom = [System.Text.UTF8Encoding]::new($false)
            [System.IO.File]::WriteAllText($SettingsFile, (($Settings | ConvertTo-Json -Depth 10) + "`n"), $Utf8NoBom)
            Write-Info "Created $SettingsFile with Plane MCP configuration."
        }
    }

    # ── Initialize spec cache ────────────────────────────────────────────────

    if (Get-Command plane -ErrorAction SilentlyContinue) {
        Write-Info 'Initializing API spec cache...'
        try {
            & plane docs update-specs 2>$null
        } catch {
            Write-Info 'Spec cache initialization skipped (authentication may be required).'
        }
    } else {
        Write-Info 'Skipping spec cache initialization (restart your terminal and run: plane docs update-specs).'
    }

    # ── Done ──────────────────────────────────────────────────────────────────

    Write-Info ''
    Write-Info "plane-cli $Version installed successfully!"
    if ($WithMcp) {
        Write-Info 'MCP server configured. Restart Claude Code to activate.'
    }
    Write-Info ''
    Write-Info 'Get started:'
    Write-Info '  plane auth login'
    Write-Info '  plane --help'

} finally {
    # Clean up temp directory
    if (Test-Path $TmpDir) {
        Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
