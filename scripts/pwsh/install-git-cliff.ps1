# Installs the git-cliff binary of release <version> (e.g. v2.14.2) into <dir>.
# git-cliff is written in Rust, so it comes from its GitHub release, not go install.
param(
    [Parameter(Position = 0)][string]$Version,
    [Parameter(Position = 1)][string]$Dir
)
$ErrorActionPreference = 'Stop'

if (-not $Version -or -not $Dir) {
    [Console]::Error.WriteLine('usage: install-git-cliff.ps1 <version> <dir>')
    exit 2
}

if (-not $IsWindows) {
    [Console]::Error.WriteLine('install-git-cliff.ps1: only Windows is supported, use install-git-cliff.sh')
    exit 1
}

$osArch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
$arch = switch ($osArch) {
    'X64' { 'x86_64' }
    'Arm64' { 'aarch64' }
    default {
        [Console]::Error.WriteLine("install-git-cliff.ps1: unsupported architecture $osArch")
        exit 1
    }
}

$number = $Version.TrimStart('v')
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    $zip = Join-Path $tmp 'git-cliff.zip'
    Invoke-WebRequest -OutFile $zip `
        -Uri "https://github.com/orhun/git-cliff/releases/download/v$number/git-cliff-$number-$arch-pc-windows-msvc.zip"
    Expand-Archive -Path $zip -DestinationPath $tmp
    New-Item -ItemType Directory -Force -Path $Dir | Out-Null
    Copy-Item -LiteralPath (Join-Path $tmp "git-cliff-$number/git-cliff.exe") -Destination (Join-Path $Dir 'git-cliff.exe')
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force
}
