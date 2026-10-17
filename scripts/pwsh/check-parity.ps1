# Fails if a script in -BashDir has no twin of the same name in -PwshDir,
# or the other way round.
param(
    [Parameter(Position = 0)][string]$BashDir,
    [Parameter(Position = 1)][string]$PwshDir
)
$ErrorActionPreference = 'Stop'

if (-not $BashDir -or -not $PwshDir) {
    [Console]::Error.WriteLine('usage: check-parity.ps1 <bash-dir> <pwsh-dir>')
    exit 2
}

$status = 0

foreach ($f in Get-ChildItem -Path $BashDir -Filter '*.sh' -File) {
    if (-not (Test-Path -LiteralPath "$PwshDir/$($f.BaseName).ps1")) {
        [Console]::Error.WriteLine("missing: $PwshDir/$($f.BaseName).ps1")
        $status = 1
    }
}

foreach ($f in Get-ChildItem -Path $PwshDir -Filter '*.ps1' -File) {
    if (-not (Test-Path -LiteralPath "$BashDir/$($f.BaseName).sh")) {
        [Console]::Error.WriteLine("missing: $BashDir/$($f.BaseName).sh")
        $status = 1
    }
}

if ($status -eq 0) {
    Write-Host "Every script has a twin in both $BashDir and $PwshDir."
}
exit $status
