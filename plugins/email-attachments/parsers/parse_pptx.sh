#!/usr/bin/env bash
# parse_pptx.sh — Extract slides and text from PPTX presentations (OOXML)
# USAGE: parse_pptx.sh <pptx-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
echo "  [PPTX] Processing: $BASENAME"

TEXT_EXTRACTED=false

# ── Primary: python-pptx → text per slide ─────────────────────────────
if python3 -c "import pptx" 2>/dev/null; then
    echo "    → python-pptx slide text extraction"
    if python3 <<PYEOF > "$NORM/content.txt" 2>&1; then
from pptx import Presentation
try:
    prs = Presentation(r"$FILE")
    for i, slide in enumerate(prs.slides, 1):
        print(f"--- Slide {i} ---")
        for shape in slide.shapes:
            if hasattr(shape, "text") and shape.text:
                print(shape.text)
        print()
except Exception as e:
    print(f"[python-pptx error: {e}]")
PYEOF
    TEXT_EXTRACTED=true
    fi
else
    echo "[python-pptx not available; install: pip install python-pptx]" > "$NORM/content.txt"
fi

# ── Fallback: LibreOffice → PDF → text + previews ────────────────────
if [[ "$TEXT_EXTRACTED" == false ]]; then
    if command -v soffice &>/dev/null || command -v libreoffice &>/dev/null; then
        SOFFICE="$(command -v soffice 2>/dev/null || command -v libreoffice 2>/dev/null)"
        echo "    → libreoffice fallback"
        TMPDIR="$(mktemp -d)"
        "$SOFFICE" --headless --convert-to pdf --outdir "$TMPDIR" "$FILE" >/dev/null 2>/dev/null || true
        RENDERED="$TMPDIR/$(basename "$BASENAME" | sed 's/\.[^.]*$/.pdf/')"
        if [[ -f "$RENDERED" ]] && command -v pdftotext &>/dev/null; then
            pdftotext -layout "$RENDERED" "$NORM/content.txt" 2>/dev/null || true
            if [[ -s "$NORM/content.txt" ]]; then
                TEXT_EXTRACTED=true
            fi
            if command -v pdftoppm &>/dev/null; then
                mkdir -p "$PREVIEW"
                pdftoppm -png -r 180 "$RENDERED" "$PREVIEW/slide" 2>/dev/null || true
            fi
        fi
        rm -rf "$TMPDIR"
    fi
fi

# ── Markdown wrapper ──────────────────────────────────────────────────
{
    printf '%s\n' "# Presentation: $BASENAME"
    printf '%s\n' ""
    printf '%s\n' "*Slides extracted as text below*"
    printf '%s\n' ""
    cat "$NORM/content.txt"
} > "$NORM/content.md"

echo "    ✓ PPTX parsed: content.txt, content.md"
