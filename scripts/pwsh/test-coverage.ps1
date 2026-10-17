# Runs the tests with coverage and writes coverage.out, coverage.html and
# coverage.xml (Cobertura) into -Dir, creating it if needed.
param(
    [Parameter(Position = 0)][string]$Gotestsum,
    [Parameter(Position = 1)][string]$Cobertura,
    [Parameter(Position = 2)][string]$Dir
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $Gotestsum -or -not $Cobertura -or -not $Dir) {
    [Console]::Error.WriteLine('usage: test-coverage.ps1 <gotestsum> <gocover-cobertura> <dir>')
    exit 2
}

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
& $Gotestsum --format testdox -- -race "-coverprofile=$Dir/coverage.out" -covermode=atomic ./...
go tool cover "-func=$Dir/coverage.out" | Select-Object -Last 1
go tool cover "-html=$Dir/coverage.out" -o "$Dir/coverage.html"
Get-Content -LiteralPath "$Dir/coverage.out" | & $Cobertura > "$Dir/coverage.xml"
Write-Host "Reports: $Dir/coverage.html | $Dir/coverage.xml"
