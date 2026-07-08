#!/usr/bin/env bash
set -euo pipefail

sudo chown -R "$(id -u):$(id -g)" /go /home/vscode

if [ -f go.mod ]; then
  go mod download
fi

profile_path=$(pwsh -NoProfile -Command 'Write-Output $PROFILE')
mkdir -p "$(dirname "$profile_path")"
cp .devcontainer/pwsh-profile.ps1 "$profile_path"
