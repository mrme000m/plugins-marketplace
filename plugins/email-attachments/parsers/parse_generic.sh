#!/usr/bin/env bash
# parse_generic.sh — Fallback for unknown or binary attachments
# USAGE: parse_generic.sh <file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

mkdir -p "$NORM" "$PREVIEW"

BASENAME="$(basename "$FILE")"
echo "  [GENERIC] Processing: $BASENAME"

# ── File type info ────────────────────────────────────────────────────
file "$FILE" > "$NORM/fileinfo.txt" 2>/dev/null || true

# ── Head/tail hex dump ────────────────────────────────────────────────
if command -v xxd &>/dev/null; then
    xxd -l 4096 "$FILE" > "$NORM/hexdump.txt" 2>/dev/null || true
elif command -v hexdump &>/dev/null; then
    hexdump -C -n 4096 "$FILE" > "$NORM/hexdump.txt" 2>/dev/null || true
fi

# ── Strings ───────────────────────────────────────────────────────────
if command -v strings &>/dev/null; then
    strings -n 8 "$FILE" > "$NORM/strings.txt" 2>/dev/null || true
else
    echo "[strings not available]" > "$NORM/strings.txt"
fi

# ── Try mime-derived heuristics ───────────────────────────────────────
MIME="$(file --brief --mime-type "$FILE" 2>/dev/null || echo "application/octet-stream")"

# If it looks like text-like, parse as text
if [[ "$MIME" == text/html* || "$MIME" == text/calendar* || "$MIME" == text/plain* ]]; then
    cp "$FILE" "$NORM/content.txt"
    {
        printf '%s\n' "# HTML Content"
        printf '%s\n' ""
        printf '%s\n' '```html'
        cat "$NORM/content.txt"
        printf '%s\n' '```'
    } > "$NORM/content.md"
else
    {
        printf '%s\n' "# Unknown/Binary File: $BASENAME"
        printf '%s\n' ""
        printf '%s\n' "- MIME type: $MIME"
        printf '%s\n' "- Full file output saved in original/"
        printf '%s\n' "- See hexdump.txt and strings.txt for manual analysis"
        printf '%s\n' ""
        printf '%s\n' "## Extracted Strings (first 100)"
        printf '%s\n' '```'
        head -100 "$NORM/strings.txt" 2>/dev/null || true
        printf '%s\n' '```'
    } > "$NORM/content.md"

    {
        printf '%s\n' "[Binary/unknown file: $MIME]"
        printf '%s\n' ""
        printf '%s\n' "Extracted strings (first 50):"
        head -50 "$NORM/strings.txt" 2>/dev/null || true
    } > "$NORM/content.txt"
fi

echo "    ✓ Generic parsed: content.txt, content.md, fileinfo.txt, strings.txt"
