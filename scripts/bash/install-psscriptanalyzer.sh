#!/usr/bin/env bash
# Saves the PSScriptAnalyzer module <version> (e.g. 1.25.0) from the PowerShell
# Gallery into <dir>. It is a pwsh module, so this needs pwsh on any OS.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: install-psscriptanalyzer.sh <version> <dir>" >&2
	exit 2
fi

mkdir -p "$2"
# Single quotes on purpose: pwsh, not bash, expands the $env: references.
# shellcheck disable=SC2016
VERSION=$1 DIR=$2 pwsh -NoProfile -Command \
	'Save-PSResource -Name PSScriptAnalyzer -Version $env:VERSION -Path $env:DIR -TrustRepository'
