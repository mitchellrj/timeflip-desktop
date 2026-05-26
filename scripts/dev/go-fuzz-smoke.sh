#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

export GOTRACEBACK="${GOTRACEBACK:-single}"
go test ./...
