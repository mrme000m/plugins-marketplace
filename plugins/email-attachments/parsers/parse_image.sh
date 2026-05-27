#!/usr/bin/env bash
# parse_image.sh — OCR, EXIF metadata, and cleaned preview from image files
# USAGE: parse_image.sh <image-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
echo "  [IMAGE] Processing: $BASENAME"

mkdir -p "$PREVIEW"

# ── EXIF metadata ─────────────────────────────────────────────────────
if command -v exiftool &>/dev/null; then
    exiftool "$FILE" > "$NORM/exif.txt" 2>/dev/null || true
fi

# ── Create cleaned preview ────────────────────────────────────────────
CLEANED="$PREVIEW/cleaned.png"
IM_OK=false
if command -v magick &>/dev/null; then
    echo "    → imagemagick preprocessing"
    if magick "$FILE" \
        -colorspace Gray \
        -density 300 \
        -deskew 40% \
        -contrast-stretch 2%x2% \
        -normalize \
        "$CLEANED" 2>/dev/null; then
        IM_OK=true
    else
        magick "$FILE" "$CLEANED" 2>/dev/null || true
    fi
elif command -v convert &>/dev/null; then
    convert "$FILE" "$CLEANED" 2>/dev/null || true
fi

# ── OCR (tesseract) ───────────────────────────────────────────────────
if command -v tesseract &>/dev/null; then
    echo "    → tesseract OCR"
    IMG_FOR_OCR="$FILE"
    [ -f "$CLEANED" ] && IMG_FOR_OCR="$CLEANED"
    OCRES="$NORM/ocr"
    if tesseract "$IMG_FOR_OCR" "$OCRES" --psm 6 2>/dev/null; then
        if [ -f "$OCRES.txt" ]; then
            mv "$OCRES.txt" "$NORM/content.txt"
        else
            echo "[OCR produced no output file]" > "$NORM/content.txt"
        fi
    else
        tesseract "$IMG_FOR_OCR" "$OCRES" 2>/dev/null || true
        if [ -f "$OCRES.txt" ]; then
            mv "$OCRES.txt" "$NORM/content.txt"
        else
            echo "[OCR failed or no text detected]" > "$NORM/content.txt"
        fi
    fi
else
    echo "[tesseract not installed]" > "$NORM/content.txt"
fi

# ── Copy original to preview ──────────────────────────────────────────
EXT="${BASENAME##*.}"
[ "$EXT" = "$BASENAME" ] && EXT="png"
cp "$FILE" "$PREVIEW/original.$EXT" 2>/dev/null || true

# ── Markdown wrapper ──────────────────────────────────────────────────
{
    printf '%s\n' "# Image: $BASENAME"
    printf '%s\n' ""
    if [ -f "$NORM/exif.txt" ]; then
        printf '%s\n' "## EXIF Metadata"
        printf '%s\n' '```'
        cat "$NORM/exif.txt"
        printf '%s\n' '```'
        printf '%s\n' ""
    fi
    printf '%s\n' "## OCR Text"
    printf '%s\n' ""
    cat "$NORM/content.txt"
} > "$NORM/content.md"

echo "    ✓ Image parsed: content.txt, content.md, exif.txt"
