#!/usr/bin/env bash
set -euo pipefail

go run "$(go env GOPATH)/pkg/mod/github.com/wailsapp/wails/v3@v3.0.0-alpha.95/cmd/wails3" generate bindings ./internal/app
