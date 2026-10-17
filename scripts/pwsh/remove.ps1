# Removes every path passed as an argument, file or directory, if it exists.
param([Parameter(ValueFromRemainingArguments)][string[]]$Paths)
$ErrorActionPreference = 'Stop'

if (-not $Paths) {
    [Console]::Error.WriteLine('usage: remove.ps1 <path>...')
    exit 2
}

foreach ($path in $Paths) {
    Write-Host "Removing $path..."
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Recurse -Force
    }
}
