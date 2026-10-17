# Writes a README.md into every package matched by -Packages with gomarkdoc,
# linking the source to -RepositoryUrl on -DefaultBranch.
param(
    [Parameter(Position = 0)][string]$Gomarkdoc,
    [Parameter(Position = 1)][string]$RepositoryUrl,
    [Parameter(Position = 2)][string]$DefaultBranch,
    [Parameter(Position = 3)][string]$Packages
)
$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true

if (-not $Gomarkdoc -or -not $RepositoryUrl -or -not $DefaultBranch -or -not $Packages) {
    [Console]::Error.WriteLine('usage: package-docs.ps1 <gomarkdoc> <repository-url> <default-branch> <packages>')
    exit 2
}

& $Gomarkdoc -e `
    --repository.url $RepositoryUrl `
    --repository.default-branch $DefaultBranch `
    --repository.path / `
    -o '{{.Dir}}/README.md' $Packages
