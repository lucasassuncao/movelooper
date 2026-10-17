#!/usr/bin/env bash
# Runs the tests with coverage and writes coverage.out, coverage.html and
# coverage.xml (Cobertura) into <dir>, creating it if needed.
set -euo pipefail

if [ $# -ne 3 ]; then
	echo "usage: test-coverage.sh <gotestsum> <gocover-cobertura> <dir>" >&2
	exit 2
fi

gotestsum=$1
cobertura=$2
dir=$3

mkdir -p "$dir"
"$gotestsum" --format testdox -- -race -coverprofile="$dir/coverage.out" -covermode=atomic ./...
go tool cover -func="$dir/coverage.out" | tail -n 1
go tool cover -html="$dir/coverage.out" -o "$dir/coverage.html"
"$cobertura" < "$dir/coverage.out" > "$dir/coverage.xml"
echo "Reports: $dir/coverage.html | $dir/coverage.xml"
