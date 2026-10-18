#!/usr/bin/env bash
# Installs the shellcheck binary of release <version> (e.g. v0.11.0) into <dir>.
# Written in Haskell, so it comes from its GitHub release, not go install.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: install-shellcheck.sh <version> <dir>" >&2
	exit 2
fi

version=$1
dir=$2

case "$(uname -s)" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*)
		echo "install-shellcheck.sh: unsupported OS $(uname -s)" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64 | amd64) arch=x86_64 ;;
	aarch64 | arm64) arch=aarch64 ;;
	*)
		echo "install-shellcheck.sh: unsupported architecture $(uname -m)" >&2
		exit 1
		;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/shellcheck.tar.gz" \
	"https://github.com/koalaman/shellcheck/releases/download/$version/shellcheck-$version.$os.$arch.tar.gz"
tar -xzf "$tmp/shellcheck.tar.gz" -C "$tmp"
mkdir -p "$dir"
cp "$tmp/shellcheck-$version/shellcheck" "$dir/shellcheck"
chmod +x "$dir/shellcheck"
