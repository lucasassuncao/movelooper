# Installs the shellcheck binary of release <version> (e.g. v0.11.0) into <dir>.
# Written in Haskell, so it comes from its GitHub release, not go install.
param(
    [Parameter(Position = 0)][string]$Version,
    [Parameter(Position = 1)][string]$Dir
)
$ErrorActionPreference = 'Stop'

if (-not $Version -or -not $Dir) {
    [Console]::Error.WriteLine('usage: install-shellcheck.ps1 <version> <dir>')
    exit 2
}

if (-not $IsWindows) {
    [Console]::Error.WriteLine('install-shellcheck.ps1: only Windows is supported, use install-shellcheck.sh')
    exit 1
}

# The release has a single Windows build, x86_64, which Arm64 Windows runs emulated.
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    $zip = Join-Path $tmp 'shellcheck.zip'
    Invoke-WebRequest -OutFile $zip `
        -Uri "https://github.com/koalaman/shellcheck/releases/download/$Version/shellcheck-$Version.zip"
    Expand-Archive -Path $zip -DestinationPath $tmp
    New-Item -ItemType Directory -Force -Path $Dir | Out-Null
    Copy-Item -LiteralPath (Join-Path $tmp 'shellcheck.exe') -Destination (Join-Path $Dir 'shellcheck.exe')
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force
}
