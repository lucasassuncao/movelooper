#!/usr/bin/env bash
# Lints the scripts in <bash-dir> with shellcheck and the ones in <pwsh-dir> with
# PSScriptAnalyzer, which is a pwsh module and so needs pwsh on any OS.
set -euo pipefail

if [ $# -ne 4 ]; then
	echo "usage: lint-scripts.sh <shellcheck> <psscriptanalyzer-manifest> <bash-dir> <pwsh-dir>" >&2
	exit 2
fi

shellcheck=$1
pssa=$2
bash_dir=$3
pwsh_dir=$4
status=0

"$shellcheck" "$bash_dir"/*.sh || status=1

# Single quotes on purpose: pwsh, not bash, expands the $ expressions. Write-Host
# is excluded because it is how these scripts print, as echo is in bash.
# shellcheck disable=SC2016
PSSA=$pssa PWSH_DIR=$pwsh_dir pwsh -NoProfile -Command '
	$ErrorActionPreference = "Stop"
	Import-Module (Resolve-Path -LiteralPath $env:PSSA).Path
	$issues = Invoke-ScriptAnalyzer -Path $env:PWSH_DIR -ExcludeRule PSAvoidUsingWriteHost
	foreach ($i in $issues) {
		[Console]::Error.WriteLine("$env:PWSH_DIR/$($i.ScriptName):$($i.Line): $($i.RuleName): $($i.Message)")
	}
	if ($issues) { exit 1 }
' || status=1

if [ $status -eq 0 ]; then
	echo "shellcheck and PSScriptAnalyzer found nothing in $bash_dir and $pwsh_dir."
fi
exit $status
