#!/usr/bin/env bash
# Writes a README.md into every package matched by <packages> with gomarkdoc,
# linking the source to <repository-url> on <default-branch>.
set -euo pipefail

if [ $# -ne 4 ]; then
	echo "usage: package-docs.sh <gomarkdoc> <repository-url> <default-branch> <packages>" >&2
	exit 2
fi

"$1" -e \
	--repository.url "$2" \
	--repository.default-branch "$3" \
	--repository.path / \
	-o '{{.Dir}}/README.md' "$4"
