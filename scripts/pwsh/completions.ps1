# Writes the bash, zsh, fish and powershell completion scripts of a Cobra CLI
# into -Dir, creating it if needed. Needs pwsh 7.4+, whose `>` writes a native
# command's bytes untouched; older versions re-encode them with CRLF endings.
param(
    [Parameter(Position = 0)][string]$Main,
    [Parameter(Position = 1)][string]$Name,
    [Parameter(Position = 2)][string]$Dir
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $Main -or -not $Name -or -not $Dir) {
    [Console]::Error.WriteLine('usage: completions.ps1 <main-path> <binary-name> <dir>')
    exit 2
}

New-Item -ItemType Directory -Force -Path $Dir | Out-Null
go run $Main completion bash > "$Dir/${Name}_completion.bash"
go run $Main completion zsh > "$Dir/${Name}_completion.zsh"
go run $Main completion fish > "$Dir/${Name}_completion.fish"
go run $Main completion powershell > "$Dir/${Name}_completion.ps1"
Write-Host "Completion scripts written to $Dir/"
