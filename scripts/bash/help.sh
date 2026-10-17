#!/usr/bin/env bash
# Prints the help of every makefile passed as an argument: a `## text` line
# documents the target on the line right below it, and nothing else does.
set -euo pipefail

if [ $# -eq 0 ]; then
	echo "usage: help.sh <makefile>..." >&2
	exit 2
fi

awk '
	/^## / { desc = substr($0, 4); next }
	desc != "" && /^[a-zA-Z_-]+:/ {
		target = $0
		sub(/:.*/, "", target)
		printf "\033[36m%-20s\033[0m %s\n", target, desc
	}
	{ desc = "" }
' "$@" | LC_ALL=C sort
