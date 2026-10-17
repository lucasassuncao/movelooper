# Creates the annotated tag <version>: "Release <version>" as the subject and the
# git-cliff notes of the commits since the previous tag as the body. Whitespace
# cleanup, because the default would strip the notes' `### ` headings as comments.
param(
    [Parameter(Position = 0)][string]$GitCliff,
    [Parameter(Position = 1)][string]$Version
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $GitCliff -or -not $Version) {
    [Console]::Error.WriteLine('usage: tag.ps1 <git-cliff> <version>')
    exit 2
}

# A file rather than captured output, which pwsh would decode with the console
# code page and so mangle any non-ASCII character in the notes.
$file = New-TemporaryFile
try {
    & $GitCliff --unreleased --tag $Version --strip all -o $file.FullName
    $notes = Get-Content -Raw -LiteralPath $file.FullName
    Set-Content -NoNewline -LiteralPath $file.FullName -Value "Release $Version`n`n$notes"
    git tag -a $Version --cleanup=whitespace -F $file.FullName
} finally {
    Remove-Item -LiteralPath $file.FullName
}
