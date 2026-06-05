#!/usr/bin/env bash
# pplx-bw-env.sh — populate a pplx .env from a Bitwarden CLI item when local env is missing
# Usage: bash pplx-bw-env.sh [--item pplx-env] [--env /path/to/.env] [--dry-run] [--force]
#
# Expected Bitwarden item formats:
#   1. Secure Note notes containing KEY=VALUE lines, e.g.
#        BITWARDEN_CLIENT_ID=...
#        BITWARDEN_CLIENT_SECRET=...
#        BWS_ACCESS_TOKEN=...
#        PERPLEXITY_COOKIES_PATH=...
#   2. Custom fields with the same key names.
#
# Safety:
#   - Does not print secret values.
#   - By default only fills missing keys; use --force to overwrite existing keys.
#   - Creates .env with mode 600 when needed.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=./pplx-common.sh
source "$SCRIPT_DIR/pplx-common.sh"

ITEM_NAME="${PPLX_BW_ENV_ITEM:-pplx-env}"
ENV_FILE="${PPLX_ENV_FILE:-$PPLX_REPO_DIR/.env}"
DRY_RUN=false
FORCE=false

usage() {
  awk '
    NR == 1 { next }
    /^# ?/ { sub(/^# ?/, ""); print; next }
    /^$/ { print; next }
    { exit }
  ' "$0"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --item)
      ITEM_NAME="${2:-}"
      shift 2
      ;;
    --env)
      ENV_FILE="${2:-}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --force)
      FORCE=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      pplx_fail "Unknown argument: $1"
      usage
      exit 2
      ;;
  esac
done

if [ -z "$ITEM_NAME" ]; then
  pplx_fail "--item cannot be empty"
  exit 2
fi

if ! pplx_has bw; then
  pplx_fail "Bitwarden CLI 'bw' not found. Install/configure bw or use BWS_ACCESS_TOKEN directly."
  exit 127
fi

if ! pplx_has jq; then
  pplx_fail "jq is required to parse bw item JSON."
  exit 127
fi

BW_ARGS=()
if [ -n "${BW_SESSION:-}" ]; then
  BW_ARGS+=(--session "$BW_SESSION")
fi

BW_STATUS="$(bw status "${BW_ARGS[@]}" 2>/dev/null | jq -r '.status // "unknown"' 2>/dev/null || echo unknown)"
if [ "$BW_STATUS" != "unlocked" ]; then
  pplx_fail "Bitwarden vault is '$BW_STATUS'. Unlock first, e.g.: export BW_SESSION=\$(bw unlock --raw)"
  exit 1
fi

TMP_ITEM="$(mktemp "${TMPDIR:-/tmp}/pplx_bw_item.XXXXXX")"
TMP_ENV="$(mktemp "${TMPDIR:-/tmp}/pplx_bw_env.XXXXXX")"
cleanup() {
  rm -f "$TMP_ITEM" "$TMP_ENV"
}
trap cleanup EXIT

if ! bw get item "$ITEM_NAME" "${BW_ARGS[@]}" > "$TMP_ITEM" 2>/tmp/pplx_bw_env_err.$$; then
  pplx_fail "Could not fetch Bitwarden item '$ITEM_NAME'."
  pplx_warn "Create a Secure Note named '$ITEM_NAME' with .env KEY=VALUE lines or matching custom fields."
  [ "$DRY_RUN" = true ] && cat /tmp/pplx_bw_env_err.$$ >&2 || true
  rm -f /tmp/pplx_bw_env_err.$$
  exit 1
fi
rm -f /tmp/pplx_bw_env_err.$$

# Extract from notes KEY=VALUE lines and from custom fields.
jq -r '
  def wanted: test("^(BITWARDEN_CLIENT_ID|BITWARDEN_CLIENT_SECRET|BWS_ACCESS_TOKEN|PERPLEXITY_COOKIES_PATH)$");
  [
    ((.notes // "") | split("\n")[]? | select(test("^[A-Za-z_][A-Za-z0-9_]*="))),
    ((.fields // [])[]? | select((.name // "") | wanted) | "\(.name)=\(.value // "")")
  ][]
' "$TMP_ITEM" | awk -F= '
  BEGIN { allow["BITWARDEN_CLIENT_ID"]=1; allow["BITWARDEN_CLIENT_SECRET"]=1; allow["BWS_ACCESS_TOKEN"]=1; allow["PERPLEXITY_COOKIES_PATH"]=1 }
  allow[$1] && length($2) > 0 && !seen[$1]++ { print }
' > "$TMP_ENV"

if [ ! -s "$TMP_ENV" ]; then
  pplx_fail "Bitwarden item '$ITEM_NAME' did not contain supported .env keys."
  pplx_warn "Supported keys: BITWARDEN_CLIENT_ID, BITWARDEN_CLIENT_SECRET, BWS_ACCESS_TOKEN, PERPLEXITY_COOKIES_PATH"
  exit 1
fi

mkdir -p "$(dirname "$ENV_FILE")"
if [ ! -f "$ENV_FILE" ]; then
  if [ "$DRY_RUN" = false ]; then
    : > "$ENV_FILE"
    chmod 600 "$ENV_FILE" 2>/dev/null || true
  fi
  pplx_info "Will create env file: $ENV_FILE"
fi

existing_has_key() {
  local key="$1"
  [ -f "$ENV_FILE" ] && grep -Eq "^[[:space:]]*(export[[:space:]]+)?${key}=" "$ENV_FILE"
}

set_key() {
  local key="$1" value="$2"
  if [ "$DRY_RUN" = true ]; then
    if existing_has_key "$key"; then
      if [ "$FORCE" = true ]; then
        pplx_info "Would overwrite $key in $ENV_FILE"
      else
        pplx_info "Would keep existing $key in $ENV_FILE"
      fi
    else
      pplx_info "Would add $key to $ENV_FILE"
    fi
    return 0
  fi

  if existing_has_key "$key"; then
    if [ "$FORCE" = true ]; then
      local escaped
      escaped="$(printf '%s' "$value" | sed 's/[&\\]/\\&/g')"
      if sed -i.bak -E "s|^[[:space:]]*(export[[:space:]]+)?${key}=.*|${key}=${escaped}|" "$ENV_FILE"; then
        rm -f "$ENV_FILE.bak"
        pplx_ok "Updated $key in $ENV_FILE"
      else
        pplx_fail "Failed to update $key in $ENV_FILE"
        return 1
      fi
    else
      pplx_info "Kept existing $key in $ENV_FILE"
    fi
  else
    printf '%s=%s\n' "$key" "$value" >> "$ENV_FILE"
    pplx_ok "Added $key to $ENV_FILE"
  fi
}

while IFS='=' read -r key value; do
  [ -n "$key" ] || continue
  set_key "$key" "$value"
done < "$TMP_ENV"

if [ "$DRY_RUN" = false ]; then
  chmod 600 "$ENV_FILE" 2>/dev/null || true
fi

pplx_ok "Bitwarden env population completed for $ENV_FILE"
pplx_warn "Values were not printed. Verify with: bash \"$SCRIPT_DIR/pplx-health.sh\" --verbose --no-search"
