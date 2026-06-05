#!/usr/bin/env bash
# pplx-upload.sh — upload a local file to a Perplexity Space via the pplx client
# Usage: bash pplx-upload.sh <space-slug-or-uuid> <file-path> [--by-uuid]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=./pplx-common.sh
source "$SCRIPT_DIR/pplx-common.sh"

if [ $# -lt 2 ] || [[ "${1:-}" =~ ^--help|-h$ ]]; then
  sed -n '1,30p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
fi

SPACE_REF="$1"
FILE_PATH="$2"
BY_UUID=false
[[ "${3:-}" == "--by-uuid" ]] && BY_UUID=true
pplx_require_file "$FILE_PATH"

if [ "$BY_UUID" = true ] || [[ "$SPACE_REF" =~ ^[0-9a-fA-F-]{32,}$ ]]; then
  SPACE_UUID="$SPACE_REF"
else
  pplx_require_jq
  SPACE_INFO="$(pplx_run --raw spaces get "$SPACE_REF" 2>/dev/null || true)"
  if [ -z "$SPACE_INFO" ]; then
    pplx_fail "Could not find Space: $SPACE_REF"
    pplx_warn "List spaces with: pplx spaces list"
    exit 1
  fi
  SPACE_UUID="$(printf '%s' "$SPACE_INFO" | jq -r '.uuid // .collection_uuid // .id // empty')"
  if [ -z "$SPACE_UUID" ]; then
    pplx_fail "Could not extract Space UUID from response. Use --by-uuid with the UUID."
    printf '%s\n' "$SPACE_INFO"
    exit 1
  fi
fi

pplx_info "Uploading $(basename "$FILE_PATH") to Space $SPACE_UUID"
pplx_run spaces upload "$SPACE_UUID" "$FILE_PATH"
pplx_ok "Upload request submitted"
pplx_warn "Check processing status with: pplx spaces files $SPACE_UUID"
