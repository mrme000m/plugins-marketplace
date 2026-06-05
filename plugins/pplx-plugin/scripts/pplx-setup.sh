#!/usr/bin/env bash
# pplx-setup.sh — configure pplx-plugin for Claude Code, OpenCode, Codex, and shell agents
# Usage: bash pplx-setup.sh [--install-client] [--print-env]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=./pplx-common.sh
source "$SCRIPT_DIR/pplx-common.sh"

INSTALL_CLIENT=false
PRINT_ENV=false
for arg in "$@"; do
  case "$arg" in
    --install-client) INSTALL_CLIENT=true ;;
    --print-env) PRINT_ENV=true ;;
    --help|-h)
      sed -n '1,64p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
  esac
done

printf 'PPLX Plugin Setup\n=================\n\n'
pplx_info "Plugin directory: $PPLX_PLUGIN_DIR"
pplx_info "Client repository: $PPLX_REPO_DIR"

if ! "$PPLX_PYTHON" - <<'PY' >/dev/null 2>&1
import sys
raise SystemExit(0 if sys.version_info >= (3, 10) else 1)
PY
then
  pplx_fail "Python 3.10+ is required."
  exit 1
fi
pplx_ok "Python 3.10+ available"

if pplx_python_importable; then
  pplx_ok "pplx_sdk Python package importable"
elif [ -f "$PPLX_REPO_DIR/pyproject.toml" ]; then
  pplx_warn "pplx_sdk Python package is not importable yet."
  if [ "$INSTALL_CLIENT" = true ]; then
    pplx_info "Installing editable client from $PPLX_REPO_DIR"
    "$PPLX_PYTHON" -m pip install -e "$PPLX_REPO_DIR"
    pplx_ok "Installed pplx_sdk client"
  else
    pplx_warn "Run: $PPLX_PYTHON -m pip install -e \"$PPLX_REPO_DIR\""
  fi
else
  pplx_warn "Could not find client pyproject.toml next to plugin. Set PPLX_REPO_DIR to the pplx-sdk/client checkout."
fi

for module in bitwarden_sdk curl_cffi dotenv; do
  if "$PPLX_PYTHON" - <<PY >/dev/null 2>&1
import $module
PY
  then pplx_ok "Python dependency importable: $module"; else pplx_warn "Missing Python dependency: $module. Run: $PPLX_PYTHON -m pip install -e \"$PPLX_REPO_DIR\""; fi
done

if [ -n "${BWS_ACCESS_TOKEN:-}" ]; then
  pplx_ok "BWS_ACCESS_TOKEN is set; BWS SDK cookie loading is enabled."
else
  pplx_warn "BWS_ACCESS_TOKEN is not set. Preferred setup: python scripts/setup_bws_secret.py create-token"
fi

printf '\nCookie resolution order used by pplx_sdk:\n'
printf '  1. PERPLEXITY_COOKIES_PATH=/path/to/cookies.json\n'
printf '  2. BWS SDK: BWS_ACCESS_TOKEN + project "pplx" + secret "perplexity-cookies"\n'
printf '  3. Legacy bw CLI fallback: Secure Note named "perplexity.ai"\n'
printf '\nDev env bootstrap from configured bw CLI, if .env is missing/incomplete:\n'
printf '  bash "%s/pplx-bw-env.sh" --item pplx-env --env "%s/.env" --dry-run\n' "$SCRIPT_DIR" "$PPLX_REPO_DIR"
printf '  bash "%s/pplx-bw-env.sh" --item pplx-env --env "%s/.env"\n' "$SCRIPT_DIR" "$PPLX_REPO_DIR"
printf '\nBWS bootstrap commands from the pplx client checkout:\n'
printf '  python scripts/setup_bws_secret.py create-token\n'
printf '  python scripts/setup_bws_secret.py setup-cookies /path/to/cookies.json\n'
printf '  python scripts/setup_bws_secret.py show\n'

if pplx_cli_available; then
  pplx_ok "pplx executable found: $(command -v pplx)"
else
  pplx_warn "pplx executable not found in PATH; scripts will use python fallback when possible."
fi

printf '\nRecommended harness configuration:\n'
printf '  Claude Code: claude --plugin-dir "%s"\n' "$PPLX_PLUGIN_DIR"
printf '  OpenCode/Codex: add "%s/scripts" to trusted helper script paths\n' "$PPLX_PLUGIN_DIR"
printf '  Env override: export PPLX_REPO_DIR="%s"\n' "$PPLX_REPO_DIR"

if [ "$PRINT_ENV" = true ]; then
  printf '\n# Shell exports\n'
  printf 'export PPLX_PLUGIN_DIR=%q\n' "$PPLX_PLUGIN_DIR"
  printf 'export PPLX_REPO_DIR=%q\n' "$PPLX_REPO_DIR"
  printf 'export PPLX_PYTHON=%q\n' "$PPLX_PYTHON"
  printf '# export BWS_ACCESS_TOKEN=<bitwarden-secrets-manager-token>\n'
  printf '# export PERPLEXITY_COOKIES_PATH=/path/to/cookies.json\n'
fi

printf '\nRun diagnostics with:\n  bash "%s/pplx-health.sh" --verbose --no-search\n' "$SCRIPT_DIR"
