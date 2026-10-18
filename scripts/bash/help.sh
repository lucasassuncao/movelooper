#!/usr/bin/env bash
# Prints the help of every makefile passed as an argument, in order: a `##@ text`
# line starts a section, and a `## text` line documents the target right below it.
set -euo pipefail

if [ $# -eq 0 ]; then
	echo "usage: help.sh <makefile>..." >&2
	exit 2
fi

awk '
	/^##@ / {
		if (sections++) print ""
		printf "\033[1m%s\033[0m\n", substr($0, 5)
		next
	}
	/^## / { desc = substr($0, 4); next }
	desc != "" && /^[a-zA-Z_-]+:/ {
		target = $0
		sub(/:.*/, "", target)
		printf "  \033[36m%-20s\033[0m %s\n", target, desc
	}
	{ desc = "" }
' "$@"
