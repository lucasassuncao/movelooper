# Records the VHS <tape> with <bin-dir> first on PATH, from the tape's own
# directory, since a tape resolves its Output and every typed path from there.
param(
    [Parameter(Position = 0)][string]$BinDir,
    [Parameter(Position = 1)][string]$Tape
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $BinDir -or -not $Tape) {
    [Console]::Error.WriteLine('usage: record-tape.ps1 <bin-dir> <tape>')
    exit 2
}

$env:PATH = (Resolve-Path -LiteralPath $BinDir).Path + [IO.Path]::PathSeparator + $env:PATH
Set-Location -LiteralPath (Split-Path -Parent $Tape)
vhs (Split-Path -Leaf $Tape)
