#!/usr/bin/env bash
# Writes the bash, zsh, fish and powershell completion scripts of a Cobra CLI
# into <dir>, creating it if needed.
set -euo pipefail

if [ $# -ne 3 ]; then
	echo "usage: completions.sh <main-path> <binary-name> <dir>" >&2
	exit 2
fi

main=$1
name=$2
dir=$3

mkdir -p "$dir"
go run "$main" completion bash > "$dir/${name}_completion.bash"
go run "$main" completion zsh > "$dir/${name}_completion.zsh"
go run "$main" completion fish > "$dir/${name}_completion.fish"
go run "$main" completion powershell > "$dir/${name}_completion.ps1"
echo "Completion scripts written to $dir/"
