#!/usr/bin/env bash
# pplx-summarize.sh — export a Perplexity thread as markdown-ish text
# Usage: bash pplx-summarize.sh <thread-slug> [--full]

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=./pplx-common.sh
source "$SCRIPT_DIR/pplx-common.sh"

if [ $# -lt 1 ] || [[ "${1:-}" =~ ^--help|-h$ ]]; then
  sed -n '1,28p' "$0" | sed 's/^# \{0,1\}//'
  exit 1
fi

THREAD_SLUG="$1"
FULL_MODE=false
[[ "${2:-}" == "--full" ]] && FULL_MODE=true

THREAD_JSON="$(pplx_run --raw threads get "$THREAD_SLUG" 2>/dev/null || true)"
if [ -z "$THREAD_JSON" ]; then
  pplx_fail "Could not fetch thread: $THREAD_SLUG"
  pplx_warn "List threads with: pplx threads list --search <topic>"
  exit 1
fi

THREAD_JSON="$THREAD_JSON" "$PPLX_PYTHON" - "$FULL_MODE" <<'PY'
import json
import os
import sys

full = sys.argv[1].lower() == "true" if len(sys.argv) > 1 else False
raw = os.environ.get("THREAD_JSON", "")
try:
    data = json.loads(raw)
except json.JSONDecodeError:
    print(raw)
    raise SystemExit(0)

# Try multiple payload shapes
entries = (
    data.get("entries")
    or data.get("messages")
    or data.get("thread", {}).get("entries")
    or []
)
title = (
    data.get("thread_title")
    or data.get("title")
    or data.get("name")
    or data.get("thread", {}).get("title")
    or "Untitled Thread"
)
print(f"# Thread: {title}\n")
if not entries:
    print("No entries found in thread payload.")
    raise SystemExit(0)

for i, entry in enumerate(entries, 1):
    query = (
        entry.get("query_str")
        or entry.get("query")
        or entry.get("user_message")
        or ""
    )
    answer_parts = []
    # Try schematized response blocks first
    for block in entry.get("blocks", []) or []:
        intended = block.get("intended_usage", "")
        if intended.endswith("_markdown") or "markdown_block" in block:
            md = block.get("markdown_block") or {}
            chunks = md.get("chunks") or []
            answer_parts.extend(chunks)
    answer = "".join(answer_parts).strip() or entry.get("answer", "") or entry.get("text", "")

    # Fallback: try schematized_response -> answer
    if not answer:
        schema = entry.get("schematized_response") or {}
        if isinstance(schema, dict):
            answer = schema.get("answer", "") or schema.get("text", "")

    if query:
        print(f"## User {i}\n{query}\n")
    if answer:
        label = "Assistant" if full or i == len(entries) else "Assistant (excerpt)"
        limit = None if full else (3000 if i == len(entries) else 500)
        rendered = answer if limit is None else answer[:limit]
        print(f"## {label}\n{rendered}")
        if limit is not None and len(answer) > limit:
            print("...(truncated)")
        print()
PY
