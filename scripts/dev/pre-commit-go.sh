#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

export GOTRACEBACK="${GOTRACEBACK:-single}"
export OBJC_DISABLE_INITIALIZE_FORK_SAFETY="${OBJC_DISABLE_INITIALIZE_FORK_SAFETY:-YES}"
export GOCACHE="${TIMEFLIP_GO_BUILD_CACHE:-/private/tmp/timeflip-desktop-go-build-cache}"
mkdir -p "$GOCACHE"

check_gofmt() {
  local files
  files="$(find . -path ./build -prune -o -name '*.go' -print0 | xargs -0 gofmt -l)"
  if [[ -n "$files" ]]; then
    printf 'Go files need gofmt:\n%s\n' "$files" >&2
    return 1
  fi
}

check_go_mod_tidy() {
  local tmpdir
  local changed=0
  tmpdir="$(mktemp -d)"
  cp go.mod "$tmpdir/go.mod"
  cp go.sum "$tmpdir/go.sum"

  go mod tidy

  if ! cmp -s go.mod "$tmpdir/go.mod" || ! cmp -s go.sum "$tmpdir/go.sum"; then
    changed=1
  fi

  cp "$tmpdir/go.mod" go.mod
  cp "$tmpdir/go.sum" go.sum
  rm -rf "$tmpdir"

  if [[ "$changed" == "1" ]]; then
    printf 'go.mod or go.sum is not tidy; run go mod tidy.\n' >&2
    return 1
  fi
}

run_go_tests() {
  local module
  local pkg
  local services_pkg
  module="$(go list -m)"
  services_pkg="${module}/internal/services"

  go test -count=1 "$services_pkg"
  go list ./... | while read -r pkg; do
    if [[ "$pkg" == "$services_pkg" ]]; then
      continue
    fi
    go test -count=1 "$pkg"
  done
}

check_gofmt
check_go_mod_tidy
go vet ./...
run_go_tests
