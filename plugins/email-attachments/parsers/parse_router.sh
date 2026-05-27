#!/usr/bin/env bash
# parse_router.sh — Complexity-based router for attachment parsing
#
# USAGE:
#   parse_router.sh <attachment-file> <normalized-out-dir> <preview-out-dir>
#
# Rules:
#   - Simple text files (< 50 KB, plain): bash (parse_text.sh)
#   - EML / multipart MIME: python3 (text_parser.py → recursive)
#   - Office docs (DOCX, XLSX, PPTX): python3
#   - PDF: bash orchestrates CLI tools (parse_pdf.sh)
#   - Images: bash + exiftool/tesseract (parse_image.sh)
#   - Archives: python3 for safe traversal
#   - Unknown: generic fallback (parse_generic.sh)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
EXT="${BASENAME##*.}"
[[ "$EXT" == "$BASENAME" ]] && EXT=""
EXT_LC=$(printf '%s' "$EXT" | tr '[:upper:]' '[:lower:]')

# ── Detect MIME type ──────────────────────────────────────────────────
MIME="application/octet-stream"
if command -v file &>/dev/null; then
    MIME=$(file --brief --mime-type "$FILE" 2>/dev/null || echo "application/octet-stream")
fi

# ── Detect file size ──────────────────────────────────────────────────
FILE_SIZE=0
if [[ -f "$FILE" ]]; then
    FILE_SIZE=$(stat -f%z "$FILE" 2>/dev/null || stat -c%s "$FILE" 2>/dev/null || echo 0)
fi

# ── Routing function ──────────────────────────────────────────────────
route() {
    local ext="$1"
    local mime="$2"
    local size="$3"

    # 1. PDF → bash (orchestrates CLI tools)
    if [[ "$mime" == "application/pdf" ]]; then
        echo "pdf"
        return
    fi

    # 2. Office Open XML (DOCX, XLSX, PPTX) → python3
    if [[ "$mime" == "application/vnd.openxmlformats-officedocument."* ]]; then
        echo "python3"
        return
    fi

    # 3. Legacy Office (DOC, XLS, PPT) → python3
    if [[ "$mime" == "application/msword" || "$mime" == "application/vnd.ms-excel" || "$mime" == "application/vnd.ms-powerpoint" ]]; then
        echo "python3"
        return
    fi

    # 4. Images → bash (exiftool, tesseract)
    if [[ "$mime" == image/* ]]; then
        echo "image"
        return
    fi

    # 5. Archives → python3 (safe traversal)
    if [[ "$mime" == "application/zip" || "$mime" == "application/x-7z-compressed" || "$mime" == "application/x-rar-compressed" || "$mime" == "application/gzip" || "$mime" == "application/x-tar" || "$mime" == "application/x-bzip2" ]]; then
        echo "python3"
        return
    fi

    # 6. EML / RFC-822 → python3 (recursive MIME parsing)
    if [[ "$mime" == "message/rfc822" || "$ext" == "eml" || "$ext" == "msg" ]]; then
        echo "python3"
        return
    fi

    # 7. ICS (iCalendar) → python3 inline
    if [[ "$mime" == "text/calendar" || "$ext" == "ics" ]]; then
        echo "python3"
        return
    fi

    # 8. Plain text → bash (fast raw copy)
    if [[ "$mime" == text/* || "$size" -lt 51200 ]]; then
        echo "bash"
        return
    fi

    # 9. Video / Audio → bash (metadata extraction)
    if [[ "$mime" == video/* || "$mime" == audio/* ]]; then
        echo "bash"
        return
    fi

    # 10. Fallback → generic
    echo "generic"
}

# Ensure output dirs exist before dispatching
mkdir -p "$NORM" "$PREVIEW"

# ── Execute routing ───────────────────────────────────────────────────
DEST=$(route "$EXT_LC" "$MIME" "$FILE_SIZE")

echo "  [ROUTER] $BASENAME → $DEST (mime=$MIME, size=${FILE_SIZE}B)"

case "$DEST" in
    bash)
        bash "$SCRIPT_DIR/parse_text.sh" "$FILE" "$NORM" "$PREVIEW"
        ;;
    python3)
        # Use the same -m invocation as parse_attachment.sh (handles relative imports)
        export PYTHONPATH="$SCRIPT_DIR:${PYTHONPATH:-}"
        python3 -m py.attachment_processor \
            "$FILE" \
            "$(dirname "$NORM")" \
            2>&1 || {
            echo "  [ROUTER] python3 failed, falling back to generic"
            bash "$SCRIPT_DIR/parse_generic.sh" "$FILE" "$NORM" "$PREVIEW"
        }
        ;;
    pdf)
        bash "$SCRIPT_DIR/parse_pdf.sh" "$FILE" "$NORM" "$PREVIEW"
        ;;
    image)
        bash "$SCRIPT_DIR/parse_image.sh" "$FILE" "$NORM" "$PREVIEW"
        ;;
    generic|*)
        bash "$SCRIPT_DIR/parse_generic.sh" "$FILE" "$NORM" "$PREVIEW"
        ;;
esac

echo "  [ROUTER] Done: $BASENAME"
