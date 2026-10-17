# Prints the help of every makefile passed as an argument: a `## text` line
# documents the target on the line right below it, and nothing else does.
param([Parameter(ValueFromRemainingArguments)][string[]]$Files)
$ErrorActionPreference = 'Stop'

if (-not $Files) {
    [Console]::Error.WriteLine('usage: help.ps1 <makefile>...')
    exit 2
}

$entries = foreach ($file in $Files) {
    $desc = $null
    foreach ($line in Get-Content -LiteralPath $file) {
        if ($line -match '^## (.+)') {
            $desc = $Matches[1]
            continue
        }
        if ($desc -and $line -match '^([a-zA-Z_-]+):') {
            [pscustomobject]@{ Target = $Matches[1]; Description = $desc }
        }
        $desc = $null
    }
}

foreach ($entry in $entries | Sort-Object Target) {
    Write-Host -NoNewline -ForegroundColor Cyan ('{0,-20}' -f $entry.Target)
    Write-Host (' ' + $entry.Description)
}
