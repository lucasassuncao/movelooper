#!/usr/bin/env bash
# Assembles the docs site into <out-dir>: <docs-dir>/pages as the root, plus the
# markdown, the icons, movelooper and dev folders and the demo gif, which the docs load
# from examples/demo/. GitHub Pages publishes this same directory.
set -euo pipefail

if [ $# -ne 3 ]; then
	echo "usage: stage-docs.sh <docs-dir> <demo-gif> <out-dir>" >&2
	exit 2
fi

docs=$1
gif=$2
out=$3

rm -rf -- "$out"
mkdir -p "$out/examples/demo"
cp -r "$docs/pages/." "$out/"
cp "$docs"/*.md "$out/"
cp -r "$docs/icons" "$docs/movelooper" "$docs/dev" "$out/"
cp "$gif" "$out/examples/demo/"
echo "Docs site staged in $out/"
