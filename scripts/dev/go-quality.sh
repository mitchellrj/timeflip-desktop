#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

export GOTRACEBACK="${GOTRACEBACK:-single}"
export OBJC_DISABLE_INITIALIZE_FORK_SAFETY="${OBJC_DISABLE_INITIALIZE_FORK_SAFETY:-YES}"
go_test_package_parallelism="${GO_TEST_PKG_PARALLELISM:-1}"

scripts/dev/pre-commit-go.sh

run_race="${RUN_GO_RACE:-auto}"
if [[ "$run_race" == "auto" && "$(go env GOOS)" != "darwin" ]]; then
  run_race=1
fi

if [[ "$run_race" == "1" ]]; then
  go test -p="$go_test_package_parallelism" -count=1 -race ./...
else
  printf 'Skipping Go race tests on %s; set RUN_GO_RACE=1 to force them.\n' "$(go env GOOS)"
fi

mkdir -p build
go test -p="$go_test_package_parallelism" -coverprofile=build/coverage.out ./...
go build -o build/timeflip-desktop ./cmd/timeflip-desktop
