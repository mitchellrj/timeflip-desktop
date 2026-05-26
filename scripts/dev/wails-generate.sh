#!/usr/bin/env bash
set -euo pipefail

wails_dir="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)"
go run "${wails_dir}/cmd/wails3" generate bindings ./internal/app
