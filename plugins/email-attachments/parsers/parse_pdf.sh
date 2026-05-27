#!/usr/bin/env bash
# parse_pdf.sh — Extract text, previews, and metadata from PDF files
# USAGE: parse_pdf.sh <pdf-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

PDF="$1"
NORM="$2"
PREVIEW="$3"

echo "  [PDF] Processing: $(basename "$PDF")"

# ── Text extraction ───────────────────────────────────────────────────
PDFTOTEXT_OK=false
if command -v pdftotext &>/dev/null; then
    echo "    → pdftotext (layout-preserving)"
    if pdftotext -layout "$PDF" "$NORM/content.txt" 2>/dev/null; then
        PDFTOTEXT_OK=true
    else
        if pdftotext "$PDF" "$NORM/content.txt" 2>/dev/null; then
            PDFTOTEXT_OK=true
        fi
    fi
fi

if [[ "$PDFTOTEXT_OK" == false ]]; then
    echo "[pdftotext not available — install poppler-utils]" > "$NORM/content.txt"
fi

# ── PDF metadata ──────────────────────────────────────────────────────
PDF_PAGES="0"
if command -v pdfinfo &>/dev/null; then
    pdfinfo "$PDF" > "$NORM/pdfinfo.txt" 2>/dev/null || true
    PDF_PAGES=$(pdfinfo "$PDF" 2>/dev/null | awk '/Pages:/{print $2}')
    PDF_PAGES="${PDF_PAGES:-0}"
fi

# ── Page preview images ───────────────────────────────────────────────
if command -v pdftoppm &>/dev/null && [[ "$PDF_PAGES" -gt 0 ]]; then
    echo "    → pdftoppm preview ($PDF_PAGES pages, first 10 only)"
    mkdir -p "$PREVIEW"
    pdftoppm -png -r 180 -l 10 "$PDF" "$PREVIEW/page" 2>/dev/null || true
else
    echo "    (!) No preview rendered (pdftoppm unavailable or 0 pages)"
fi

# ── OCR fallback for scanned/image-based PDFs ─────────────────────────
OCR_PERFORMED=false
if [[ "$PDFTOTEXT_OK" == true ]]; then
    TEXT_LEN=$(wc -c < "$NORM/content.txt" 2>/dev/null || echo 0)
    TEXT_LEN="${TEXT_LEN// /}"
    if [[ "$TEXT_LEN" -lt 100 && "$PDF_PAGES" -gt 0 ]]; then
        OCR_PERFORMED=true
    fi
fi

if [[ "$OCR_PERFORMED" == true ]] && command -v tesseract &>/dev/null; then
    echo "    → OCR fallback (low text yield: ${TEXT_LEN:-0} bytes)"
    printf '\n--- OCR FALLBACK ---\n' >> "$NORM/content.txt"
    for img in "$PREVIEW"/page-*.png; do
        [[ -f "$img" ]] || continue
        page_num=$(echo "$img" | sed -n 's/.*page-\([0-9]*\).*/\1/p')
        page_num="${page_num:-?}"
        printf '\n--- Page %s ---\n' "$page_num" >> "$NORM/content.txt"
        tesseract "$img" stdout --psm 6 2>/dev/null >> "$NORM/content.txt" || true
    done
fi

# ── Markdown variant ──────────────────────────────────────────────────
{
    printf '%s\n' "# PDF Content: $(basename "$PDF")"
    printf '%s\n' ""
    printf '%s\n' "*Extraction source: pdftotext (layout mode)*"
    if [[ -f "$NORM/pdfinfo.txt" ]]; then
        printf '%s\n' "*PDF metadata available in pdfinfo.txt*"
    fi
    printf '%s\n' "*Pages: $PDF_PAGES*"
    if [[ "$OCR_PERFORMED" == true ]]; then
        printf '%s\n' "*OCR fallback was performed*"
    fi
    printf '%s\n' ""
    cat "$NORM/content.txt"
} > "$NORM/content.md"

# Count previews safely (no ls glob crash)
PREVIEW_COUNT=0
for png in "$PREVIEW"/page-*.png; do
    [[ -f "$png" ]] || continue
    PREVIEW_COUNT=$((PREVIEW_COUNT + 1))
done

echo "    ✓ PDF parsed: content.txt, content.md, $PREVIEW_COUNT previews"
