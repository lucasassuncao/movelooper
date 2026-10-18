#!/usr/bin/env bash
# Records the VHS <tape> with <bin-dir> first on PATH, from the tape's own
# directory, since a tape resolves its Output and every typed path from there.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: record-tape.sh <bin-dir> <tape>" >&2
	exit 2
fi

bin_dir=$(cd "$1" && pwd)
tape=$2

cd "$(dirname "$tape")"
PATH="$bin_dir:$PATH" vhs "$(basename "$tape")"
