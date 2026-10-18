# Prints the help of every makefile passed as an argument, in order: a `##@ text`
# line starts a section, and a `## text` line documents the target right below it.
param([Parameter(ValueFromRemainingArguments)][string[]]$Files)
$ErrorActionPreference = 'Stop'

if (-not $Files) {
    [Console]::Error.WriteLine('usage: help.ps1 <makefile>...')
    exit 2
}

$sections = 0
foreach ($file in $Files) {
    $desc = $null
    foreach ($line in Get-Content -LiteralPath $file) {
        if ($line -match '^##@ (.+)') {
            if ($sections++) { Write-Host '' }
            Write-Host "$($PSStyle.Bold)$($Matches[1])$($PSStyle.Reset)"
            continue
        }
        if ($line -match '^## (.+)') {
            $desc = $Matches[1]
            continue
        }
        if ($desc -and $line -match '^([a-zA-Z_-]+):') {
            Write-Host ('  {0}{1,-20}{2} {3}' -f $PSStyle.Foreground.Cyan, $Matches[1], $PSStyle.Reset, $desc)
        }
        $desc = $null
    }
}
