#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TOOLCHAIN_ROOT="${RUNNER_TOOLCHAIN_DIR:-$HOME/.local/toolchain}"
NPM_GLOBAL_PREFIX="${RUNNER_NPM_PREFIX:-$HOME/.local/npm-global}"
GO_PROXY_URL="${GOPROXY:-https://goproxy.cn,direct}"
GO_SUM_URL="${GOSUMDB:-sum.golang.google.cn}"
NPM_REGISTRY_URL="${NPM_CONFIG_REGISTRY:-https://registry.npmmirror.com}"
GO_VERSION="${GO_VERSION_OVERRIDE:-$(sed -nE 's/^go ([0-9]+(\.[0-9]+){1,2}).*$/\1/p' "$REPO_ROOT/go.mod" | head -n 1)}"
NODE_VERSION="${NODE_VERSION_OVERRIDE:-20.19.0}"
PNPM_VERSION="${PNPM_VERSION_OVERRIDE:-10.11.0}"

mkdir -p "$TOOLCHAIN_ROOT" "$NPM_GLOBAL_PREFIX"

append_path() {
  local path_entry="$1"
  case ":$PATH:" in
    *":$path_entry:"*) ;;
    *) export PATH="$path_entry:$PATH" ;;
  esac
  if [[ -n "${GITHUB_PATH:-}" ]]; then
    printf '%s\n' "$path_entry" >> "$GITHUB_PATH"
  fi
}

ensure_downloader() {
  if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget"
    return
  fi
  echo "missing required command: curl or wget" >&2
  exit 1
}

download_to() {
  local url="$1"
  local output="$2"
  case "$DOWNLOADER" in
    curl)
      curl -fsSL "$url" -o "$output"
      ;;
    wget)
      wget -qO "$output" "$url"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      GO_ARCH="amd64"
      NODE_ARCH="x64"
      ;;
    aarch64|arm64)
      GO_ARCH="arm64"
      NODE_ARCH="arm64"
      ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

install_go_if_missing() {
  if command -v go >/dev/null 2>&1; then
    return
  fi

  detect_arch
  ensure_downloader

  local archive_name="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
  local archive_path="$TOOLCHAIN_ROOT/$archive_name"
  local install_parent="$TOOLCHAIN_ROOT/go-dist"
  local install_path="$install_parent/go"

  rm -rf "$install_parent"
  mkdir -p "$install_parent"

  download_to "https://golang.google.cn/dl/${archive_name}" "$archive_path"
  tar -C "$install_parent" -xzf "$archive_path"
  append_path "$install_path/bin"
}

install_node_if_missing() {
  if command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
    return
  fi

  detect_arch
  ensure_downloader

  local archive_name="node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.xz"
  local archive_path="$TOOLCHAIN_ROOT/$archive_name"
  local install_parent="$TOOLCHAIN_ROOT/node-dist"
  local install_path="$install_parent/node-v${NODE_VERSION}-linux-${NODE_ARCH}"

  rm -rf "$install_parent"
  mkdir -p "$install_parent"

  download_to "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/${archive_name}" "$archive_path"
  tar -C "$install_parent" -xJf "$archive_path"
  append_path "$install_path/bin"
}

install_pnpm_if_missing() {
  export NPM_CONFIG_PREFIX="$NPM_GLOBAL_PREFIX"
  export npm_config_prefix="$NPM_GLOBAL_PREFIX"
  append_path "$NPM_GLOBAL_PREFIX/bin"

  if command -v pnpm >/dev/null 2>&1; then
    return
  fi

  npm install --global "pnpm@${PNPM_VERSION}" --registry "$NPM_REGISTRY_URL"
}

install_go_if_missing
install_node_if_missing
install_pnpm_if_missing

if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    printf 'GOPROXY=%s\n' "$GO_PROXY_URL"
    printf 'GOSUMDB=%s\n' "$GO_SUM_URL"
    printf 'NPM_CONFIG_REGISTRY=%s\n' "$NPM_REGISTRY_URL"
    printf 'npm_config_registry=%s\n' "$NPM_REGISTRY_URL"
    printf 'NPM_CONFIG_PREFIX=%s\n' "$NPM_GLOBAL_PREFIX"
    printf 'npm_config_prefix=%s\n' "$NPM_GLOBAL_PREFIX"
  } >> "$GITHUB_ENV"
fi

echo "git:  $(git --version)"
echo "go:   $(go version)"
echo "node: $(node --version)"
echo "pnpm: $(pnpm --version)"
echo "GOPROXY=${GO_PROXY_URL}"
echo "GOSUMDB=${GO_SUM_URL}"
echo "NPM registry=${NPM_REGISTRY_URL}"
