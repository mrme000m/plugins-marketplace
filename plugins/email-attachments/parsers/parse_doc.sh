#!/usr/bin/env bash
# parse_doc.sh — Legacy OLE Office documents (.doc, .xls, .ppt)
# These are NOT OOXML. They use OLE2 compound file format.
# USAGE: parse_doc.sh <file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
MIME="$(file --brief --mime-type "$FILE" 2>/dev/null || echo "application/octet-stream")"
echo "  [OLE] Processing: $BASENAME (type: $MIME)"

# ── OLE metadata via oletools (if available) ──────────────────────────
OLEMETA_OK=false
if command -v oleid &>/dev/null; then
    echo "    → oleid scan"
    oleid "$FILE" > "$NORM/oleid.txt" 2>/dev/null || true
    OLEMETA_OK=true
fi

if command -v olemeta &>/dev/null; then
    echo "    → olemeta extraction"
    olemeta "$FILE" > "$NORM/olemeta.txt" 2>/dev/null || true
fi

if command -v oleobj &>/dev/null; then
    echo "    → oleobj extraction"
    mkdir -p "$NORM/ole_objects"
    oleobj "$FILE" -d "$NORM/ole_objects" >/dev/null 2>/dev/null || true
fi

# ── Text extraction: antiword, then LibreOffice, then strings ─────────
TEXT_EXTRACTED=false

if command -v antiword &>/dev/null; then
    echo "    → antiword text extraction"
    if antiword "$FILE" > "$NORM/content.txt" 2>/dev/null; then
        TEXT_EXTRACTED=true
    fi
fi

if [[ "$TEXT_EXTRACTED" == false ]] && (command -v soffice &>/dev/null || command -v libreoffice &>/dev/null); then
    SOFFICE="$(command -v soffice 2>/dev/null || command -v libreoffice 2>/dev/null)"
    echo "    → libreoffice conversion fallback"
    TMPDIR="$(mktemp -d)"
    "$SOFFICE" --headless --convert-to txt --outdir "$TMPDIR" "$FILE" >/dev/null 2>/dev/null || true
    RENDERED="$TMPDIR/$(basename "$BASENAME" | sed 's/\.[^.]*$/.txt/')"
    if [[ -f "$RENDERED" ]] && [[ -s "$RENDERED" ]]; then
        mv "$RENDERED" "$NORM/content.txt"
        TEXT_EXTRACTED=true
    fi
    if [[ "$TEXT_EXTRACTED" == false ]]; then
        "$SOFFICE" --headless --convert-to pdf --outdir "$TMPDIR" "$FILE" >/dev/null 2>/dev/null || true
        RENDERED_PDF="$TMPDIR/$(basename "$BASENAME" | sed 's/\.[^.]*$/.pdf/')"
        if [[ -f "$RENDERED_PDF" ]] && command -v pdftotext &>/dev/null; then
            pdftotext -layout "$RENDERED_PDF" "$NORM/content.txt" 2>/dev/null || true
            if [[ -s "$NORM/content.txt" ]]; then
                TEXT_EXTRACTED=true
            fi
            if command -v pdftoppm &>/dev/null; then
                mkdir -p "$PREVIEW"
                pdftoppm -png -r 180 -l 10 "$RENDERED_PDF" "$PREVIEW/page" 2>/dev/null || true
            fi
        fi
    fi
    rm -rf "$TMPDIR"
fi

# Last resort: strings
if [[ "$TEXT_EXTRACTED" == false ]] && command -v strings &>/dev/null; then
    echo "    → strings fallback"
    strings -n 8 "$FILE" > "$NORM/content.txt" 2>/dev/null || true
fi

if [[ ! -f "$NORM/content.txt" ]]; then
    echo "[No text extraction available for $BASENAME]" > "$NORM/content.txt"
fi

# ── Markdown wrapper ──────────────────────────────────────────────────
{
    printf '%s\n' "# Legacy Office Document: $BASENAME"
    printf '%s\n' ""
    printf '%s\n' "- MIME type: $MIME"
    printf '%s\n' "- Format: OLE2 compound file (legacy, not OOXML)"
    printf '%s\n' ""

    if [[ "$OLEMETA_OK" == true ]]; then
        printf '%s\n' "## OLE Analysis"
        printf '%s\n' '```'
        cat "$NORM/oleid.txt" 2>/dev/null
        printf '%s\n' '```'
        printf '%s\n' ""
    fi

    if [[ -f "$NORM/olemeta.txt" ]]; then
        printf '%s\n' "## Metadata"
        printf '%s\n' '```'
        cat "$NORM/olemeta.txt" 2>/dev/null
        printf '%s\n' '```'
        printf '%s\n' ""
    fi

    printf '%s\n' "## Extracted Content"
    printf '%s\n' ""
    printf '%s\n' '```'
    cat "$NORM/content.txt"
    printf '%s\n' '```'
} > "$NORM/content.md"

OBJECT_COUNT=0
for f in "$NORM/ole_objects"/*; do
    [[ -f "$f" ]] || continue
    OBJECT_COUNT=$((OBJECT_COUNT + 1))
done

echo "    ✓ OLE parsed: content.txt, content.md, oleid.txt, olemeta.txt, $OBJECT_COUNT embedded objects"
