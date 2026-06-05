#!/usr/bin/env bash
# pplx-health.sh — diagnose pplx-plugin CLI, BWS auth, cookies, and Perplexity connectivity
# Usage: bash pplx-health.sh [--verbose] [--no-search]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=./pplx-common.sh
source "$SCRIPT_DIR/pplx-common.sh"

VERBOSE=false
NO_SEARCH=false
for arg in "$@"; do
  case "$arg" in
    --verbose|-v) VERBOSE=true ;;
    --no-search) NO_SEARCH=true ;;
    --help|-h)
      sed -n '1,52p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
  esac
done

PASS=0
FAIL=0
WARN=0
check() { if [ "$2" = "0" ]; then pplx_ok "$1"; PASS=$((PASS+1)); else pplx_fail "$1"; FAIL=$((FAIL+1)); fi; }
warn_count() { pplx_warn "$1"; WARN=$((WARN+1)); }

printf 'PPLX Plugin Health Check\n========================\n\n'
pplx_info "Plugin: $PPLX_PLUGIN_DIR"
pplx_info "Client repo: $PPLX_REPO_DIR"

if "$PPLX_PYTHON" --version >/dev/null 2>&1; then check "Python available ($($PPLX_PYTHON --version 2>&1))" 0; else check "Python available" 1; fi

if pplx_cli_available; then
  check "pplx executable in PATH" 0
  [ "$VERBOSE" = true ] && pplx_run --version || true
elif pplx_python_importable; then
  check "pplx Python package importable" 0
  warn_count "pplx executable is not in PATH; helpers will call pplx_sdk.cli through Python."
else
  check "pplx executable or Python package" 1
fi

for module in curl_cffi dotenv bitwarden_sdk; do
  if "$PPLX_PYTHON" - <<PY >/dev/null 2>&1
import $module
PY
  then check "Python dependency: $module" 0; else check "Python dependency: $module" 1; fi
done

# Cookie resolution order used by pplx_sdk.auth.BitwardenAuth:
# 1. PERPLEXITY_COOKIES_PATH file
# 2. Bitwarden Secrets Manager via bitwarden-sdk, BWS_ACCESS_TOKEN, project 'pplx', secret 'perplexity-cookies'
# 3. Legacy bw CLI Secure Note named 'perplexity.ai'
if [ -n "${PERPLEXITY_COOKIES_PATH:-}" ]; then
  if [ -f "$PERPLEXITY_COOKIES_PATH" ]; then
    if "$PPLX_PYTHON" -m json.tool "$PERPLEXITY_COOKIES_PATH" >/dev/null 2>&1; then
      check "PERPLEXITY_COOKIES_PATH JSON" 0
    else
      check "PERPLEXITY_COOKIES_PATH JSON" 1
    fi
  else
    check "PERPLEXITY_COOKIES_PATH file exists" 1
  fi
fi

if [ -n "${BWS_ACCESS_TOKEN:-}" ]; then
  if PYTHONPATH="$PPLX_REPO_DIR/src${PYTHONPATH:+:$PYTHONPATH}" "$PPLX_PYTHON" - <<'PY' >/tmp/pplx_bws_check.$$ 2>/tmp/pplx_bws_err.$$
import json
from pplx_sdk.auth.bitwarden import BitwardenAuth
auth = BitwardenAuth()
client = auth._load_from_bws()
print(f"cookies_keys={len(client)}")
PY
  then
    check "BWS secret 'perplexity-cookies'" 0
    [ "$VERBOSE" = true ] && cat /tmp/pplx_bws_check.$$
  else
    check "BWS secret 'perplexity-cookies'" 1
    [ "$VERBOSE" = true ] && cat /tmp/pplx_bws_err.$$ >&2 || true
  fi
  rm -f /tmp/pplx_bws_check.$$ /tmp/pplx_bws_err.$$
else
  warn_count "BWS_ACCESS_TOKEN is not set; BWS SDK cookie loading is not configured. Run: python scripts/setup_bws_secret.py create-token"
fi

if PYTHONPATH="$PPLX_REPO_DIR/src${PYTHONPATH:+:$PYTHONPATH}" "$PPLX_PYTHON" - <<'PY' >/tmp/pplx_cookie_loader.$$ 2>/tmp/pplx_cookie_loader_err.$$
from pplx_sdk.auth.bitwarden import BitwardenAuth
auth = BitwardenAuth()
cookies = auth.get_cookies()
if not isinstance(cookies, dict) or not cookies:
    raise SystemExit("cookie loader returned no cookies")
print(f"cookie_keys={len(cookies)}")
PY
then
  check "Perplexity cookie loader resolved cookies" 0
  [ "$VERBOSE" = true ] && cat /tmp/pplx_cookie_loader.$$
else
  check "Perplexity cookie loader resolved cookies" 1
  [ "$VERBOSE" = true ] && cat /tmp/pplx_cookie_loader_err.$$ >&2 || true
fi
rm -f /tmp/pplx_cookie_loader.$$ /tmp/pplx_cookie_loader_err.$$

if [ -z "${BWS_ACCESS_TOKEN:-}" ]; then
  if pplx_has bw; then
    check "Legacy Bitwarden CLI fallback (bw)" 0
    BW_STATUS="$(bw status 2>/dev/null | jq -r '.status' 2>/dev/null || echo unknown)"
    if [ "$BW_STATUS" != "unlocked" ]; then
      warn_count "Legacy bw vault status is '$BW_STATUS'. BWS is preferred; unlock bw only if using legacy Secure Note fallback."
    fi
  else
    warn_count "Legacy bw CLI not found; this is fine when BWS_ACCESS_TOKEN or PERPLEXITY_COOKIES_PATH is configured."
  fi
fi

if pplx_run models >/tmp/pplx_models_check.$$ 2>/tmp/pplx_models_err.$$; then
  if grep -Eq 'AUTO|PRO|REASONING|DEEP' /tmp/pplx_models_check.$$; then
    check "Dynamic model discovery" 0
    [ "$VERBOSE" = true ] && sed -n '1,40p' /tmp/pplx_models_check.$$
  else
    warn_count "Model command returned an unexpected shape."
    [ "$VERBOSE" = true ] && cat /tmp/pplx_models_check.$$
  fi
else
  check "Dynamic model discovery" 1
  [ "$VERBOSE" = true ] && cat /tmp/pplx_models_err.$$ >&2 || true
fi
rm -f /tmp/pplx_models_check.$$ /tmp/pplx_models_err.$$

if [ "$NO_SEARCH" = false ]; then
  if pplx_run --raw search "pplx health check: respond with ok" >/tmp/pplx_search_check.$$ 2>/tmp/pplx_search_err.$$; then
    if jq -e '.backend_uuid or .text or .answer' /tmp/pplx_search_check.$$ >/dev/null 2>&1; then
      check "Search connectivity" 0
    else
      warn_count "Search returned non-JSON or unexpected JSON."
      [ "$VERBOSE" = true ] && sed -n '1,20p' /tmp/pplx_search_check.$$
    fi
  else
    check "Search connectivity" 1
    [ "$VERBOSE" = true ] && cat /tmp/pplx_search_err.$$ >&2 || true
  fi
  rm -f /tmp/pplx_search_check.$$ /tmp/pplx_search_err.$$
else
  warn_count "Skipped live search connectivity check (--no-search)."
fi

printf '\nResults: %s passed, %s failed, %s warnings\n' "$PASS" "$FAIL" "$WARN"
[ "$FAIL" -eq 0 ]
