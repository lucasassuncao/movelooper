# Assembles the docs site into -Out: -Docs/pages as the root, plus the markdown,
# the icons, movelooper and dev folders and the gifs in -Demo, which the pages
# load from examples/demo/. GitHub Pages publishes this same directory.
param(
    [Parameter(Position = 0)][string]$Docs,
    [Parameter(Position = 1)][string]$Demo,
    [Parameter(Position = 2)][string]$Out
)
$ErrorActionPreference = 'Stop'

if (-not $Docs -or -not $Demo -or -not $Out) {
    [Console]::Error.WriteLine('usage: stage-docs.ps1 <docs-dir> <demo-dir> <out-dir>')
    exit 2
}

if (Test-Path -LiteralPath $Out) {
    Remove-Item -LiteralPath $Out -Recurse -Force
}
New-Item -ItemType Directory -Force -Path "$Out/examples/demo" | Out-Null
Copy-Item -Path "$Docs/pages/*" -Destination $Out -Recurse
Copy-Item -Path "$Docs/*.md" -Destination $Out
Copy-Item -LiteralPath "$Docs/icons", "$Docs/movelooper", "$Docs/dev" -Destination $Out -Recurse
Copy-Item -Path "$Demo/*.gif" -Destination "$Out/examples/demo/"
Write-Host "Docs site staged in $Out/"
