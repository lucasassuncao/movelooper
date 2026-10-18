# Lints the scripts in <bash-dir> with shellcheck and the ones in <pwsh-dir> with
# PSScriptAnalyzer, which is a pwsh module and so needs pwsh on any OS.
param(
    [Parameter(Position = 0)][string]$ShellCheck,
    [Parameter(Position = 1)][string]$PSScriptAnalyzer,
    [Parameter(Position = 2)][string]$BashDir,
    [Parameter(Position = 3)][string]$PwshDir
)
$ErrorActionPreference = 'Stop'

if (-not $ShellCheck -or -not $PSScriptAnalyzer -or -not $BashDir -or -not $PwshDir) {
    [Console]::Error.WriteLine('usage: lint-scripts.ps1 <shellcheck> <psscriptanalyzer-manifest> <bash-dir> <pwsh-dir>')
    exit 2
}

$status = 0

& $ShellCheck (Get-ChildItem -Path $BashDir -Filter '*.sh' -File).FullName
if ($LASTEXITCODE -ne 0) { $status = 1 }

# Resolved first: Import-Module takes a relative path without ./ as a module name.
Import-Module (Resolve-Path -LiteralPath $PSScriptAnalyzer).Path
# Write-Host is excluded because it is how these scripts print, as echo is in bash.
$issues = Invoke-ScriptAnalyzer -Path $PwshDir -ExcludeRule PSAvoidUsingWriteHost
foreach ($i in $issues) {
    [Console]::Error.WriteLine("$PwshDir/$($i.ScriptName):$($i.Line): $($i.RuleName): $($i.Message)")
}
if ($issues) { $status = 1 }

if ($status -eq 0) {
    Write-Host "shellcheck and PSScriptAnalyzer found nothing in $BashDir and $PwshDir."
}
exit $status
