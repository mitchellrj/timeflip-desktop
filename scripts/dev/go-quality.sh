#!/usr/bin/env bash
set -euo pipefail

gofmt -w $(find . -name '*.go' -not -path './build/*' -not -path './frontend/*')
go mod tidy
go vet ./...
go test ./...

