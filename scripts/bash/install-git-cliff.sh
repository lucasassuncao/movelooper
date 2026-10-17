#!/usr/bin/env bash
# Installs the git-cliff binary of release <version> (e.g. v2.14.2) into <dir>.
# git-cliff is written in Rust, so it comes from its GitHub release, not go install.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: install-git-cliff.sh <version> <dir>" >&2
	exit 2
fi

number=${1#v}
dir=$2

# musl on Linux: statically linked, so the binary runs on any distribution.
case "$(uname -s)" in
	Linux) os=unknown-linux-musl ;;
	Darwin) os=apple-darwin ;;
	*)
		echo "install-git-cliff.sh: unsupported OS $(uname -s)" >&2
		exit 1
		;;
esac

case "$(uname -m)" in
	x86_64 | amd64) arch=x86_64 ;;
	aarch64 | arm64) arch=aarch64 ;;
	*)
		echo "install-git-cliff.sh: unsupported architecture $(uname -m)" >&2
		exit 1
		;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/git-cliff.tar.gz" \
	"https://github.com/orhun/git-cliff/releases/download/v$number/git-cliff-$number-$arch-$os.tar.gz"
tar -xzf "$tmp/git-cliff.tar.gz" -C "$tmp"
mkdir -p "$dir"
cp "$tmp/git-cliff-$number/git-cliff" "$dir/git-cliff"
chmod +x "$dir/git-cliff"
