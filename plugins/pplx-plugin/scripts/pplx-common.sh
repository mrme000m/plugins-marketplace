#!/usr/bin/env bash
# pplx-common.sh — shared helpers for pplx-plugin scripts

set -euo pipefail

pplx_plugin_dir() {
  local src="${BASH_SOURCE[0]}"
  while [ -L "$src" ]; do
    local dir
    dir="$(cd -P "$(dirname "$src")" >/dev/null 2>&1 && pwd)"
    src="$(readlink "$src")"
    [[ "$src" != /* ]] && src="$dir/$src"
  done
  cd -P "$(dirname "$src")/.." >/dev/null 2>&1 && pwd
}

pplx_load_env_file() {
  local env_file="$1"
  [ -f "$env_file" ] || return 0

  local line key value
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line//$'\r'/}"
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" != *=* ]] && continue
    key="${line%%=*}"
    value="${line#*=}"
    key="${key#export }"
    key="${key//[[:space:]]/}"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    # Do not overwrite explicit environment values supplied by the harness/user.
    [ -n "${!key:-}" ] && continue
    # Strip one matching quote pair for common .env forms. Do not eval.
    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi
    export "$key=$value"
  done < "$env_file"
}

PPLX_PLUGIN_DIR="${PPLX_PLUGIN_DIR:-$(pplx_plugin_dir)}"
PPLX_REPO_DIR="${PPLX_REPO_DIR:-$(cd "$PPLX_PLUGIN_DIR/.." >/dev/null 2>&1 && pwd)}"
PPLX_PYTHON="${PPLX_PYTHON:-python3}"
pplx_load_env_file "$PPLX_REPO_DIR/.env"

pplx_info() { printf '[INFO] %s\n' "$*"; }
pplx_ok()   { printf '[OK] %s\n' "$*"; }
pplx_warn() { printf '[WARN] %s\n' "$*"; }
pplx_fail() { printf '[FAIL] %s\n' "$*"; }

pplx_has() { command -v "$1" >/dev/null 2>&1; }

pplx_cli_available() { pplx_has pplx; }

pplx_python_importable() {
  "$PPLX_PYTHON" - <<'PY' >/dev/null 2>&1
import pplx_sdk
PY
}

pplx_run() {
  if pplx_cli_available; then
    pplx "$@"
    return
  fi

  if pplx_python_importable; then
    PYTHONPATH="$PPLX_REPO_DIR${PYTHONPATH:+:$PYTHONPATH}" "$PPLX_PYTHON" -m pplx_sdk.cli "$@"
    return
  fi

  # Fallback: try to run the CLI from the repo src directory directly
  if [ -d "$PPLX_REPO_DIR/src" ]; then
    PYTHONPATH="$PPLX_REPO_DIR/src${PYTHONPATH:+:$PYTHONPATH}" "$PPLX_PYTHON" -m pplx_sdk.cli "$@"
    return
  fi

  pplx_fail "pplx CLI is not installed and the pplx_sdk Python package is not importable."
  pplx_warn "Install with: pip install -e \"$PPLX_REPO_DIR\""
  return 127
}

pplx_require_jq() {
  if ! pplx_has jq; then
    pplx_fail "jq is required for this helper. Install jq and retry."
    return 127
  fi
}

pplx_require_file() {
  local file="$1"
  if [ ! -f "$file" ]; then
    pplx_fail "File not found: $file"
    return 1
  fi
}
