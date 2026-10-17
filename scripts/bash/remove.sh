#!/usr/bin/env bash
# Removes every path passed as an argument, file or directory, if it exists.
set -euo pipefail

if [ $# -eq 0 ]; then
	echo "usage: remove.sh <path>..." >&2
	exit 2
fi

for path in "$@"; do
	echo "Removing $path..."
	rm -rf -- "$path"
done
