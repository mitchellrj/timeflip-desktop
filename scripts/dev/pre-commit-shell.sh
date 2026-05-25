#!/usr/bin/env bash
set -euo pipefail

for file in scripts/dev/*.sh; do
  bash -n "$file"
done

