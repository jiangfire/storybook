#!/usr/bin/env bash
set -euo pipefail

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

require_command git
require_command go
require_command node
require_command pnpm

echo "git:  $(git --version)"
echo "go:   $(go version)"
echo "node: $(node --version)"
echo "pnpm: $(pnpm --version)"

echo "GOPROXY=${GOPROXY:-}"
echo "GOSUMDB=${GOSUMDB:-}"
echo "NPM registry=${NPM_CONFIG_REGISTRY:-}"
