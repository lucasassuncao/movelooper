#!/usr/bin/env bash
# Creates the annotated tag <version>: "Release <version>" as the subject and the
# git-cliff notes of the commits since the previous tag as the body. Whitespace
# cleanup, because the default would strip the notes' `### ` headings as comments.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: tag.sh <git-cliff> <version>" >&2
	exit 2
fi

gitcliff=$1
version=$2

notes=$("$gitcliff" --unreleased --tag "$version" --strip all)
printf 'Release %s\n\n%s\n' "$version" "$notes" | git tag -a "$version" --cleanup=whitespace -F -
