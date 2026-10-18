# Saves the PSScriptAnalyzer module <version> (e.g. 1.25.0) from the PowerShell
# Gallery into <dir>. It is a pwsh module, so this needs pwsh on any OS.
param(
    [Parameter(Position = 0)][string]$Version,
    [Parameter(Position = 1)][string]$Dir
)
$ErrorActionPreference = 'Stop'

if (-not $Version -or -not $Dir) {
    [Console]::Error.WriteLine('usage: install-psscriptanalyzer.ps1 <version> <dir>')
    exit 2
}

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
Save-PSResource -Name PSScriptAnalyzer -Version $Version -Path $Dir -TrustRepository
