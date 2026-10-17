#!/usr/bin/env bash
# Fails if a script in <bash-dir> has no twin of the same name in <pwsh-dir>,
# or the other way round.
set -euo pipefail

if [ $# -ne 2 ]; then
	echo "usage: check-parity.sh <bash-dir> <pwsh-dir>" >&2
	exit 2
fi

bash_dir=$1
pwsh_dir=$2
status=0

for f in "$bash_dir"/*.sh; do
	[ -e "$f" ] || continue
	name=$(basename "$f" .sh)
	if [ ! -e "$pwsh_dir/$name.ps1" ]; then
		echo "missing: $pwsh_dir/$name.ps1" >&2
		status=1
	fi
done

for f in "$pwsh_dir"/*.ps1; do
	[ -e "$f" ] || continue
	name=$(basename "$f" .ps1)
	if [ ! -e "$bash_dir/$name.sh" ]; then
		echo "missing: $bash_dir/$name.sh" >&2
		status=1
	fi
done

if [ $status -eq 0 ]; then
	echo "Every script has a twin in both $bash_dir and $pwsh_dir."
fi
exit $status
