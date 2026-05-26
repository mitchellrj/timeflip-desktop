#!/usr/bin/env bash
set -euo pipefail

npm --prefix frontend install
npm --prefix frontend run build
