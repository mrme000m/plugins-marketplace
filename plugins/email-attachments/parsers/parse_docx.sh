#!/usr/bin/env bash
# parse_docx.sh — Extract text, media, and preview from DOCX files (OOXML, not legacy .doc)
# USAGE: parse_docx.sh <docx-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
echo "  [DOCX] Processing: $BASENAME"

TEXT_EXTRACTED=false

# ── Primary: Pandoc to markdown (with media extraction) ───────────────
if command -v pandoc &>/dev/null; then
    echo "    → pandoc markdown extraction"
    MEDIA_DIR="$NORM/media"
    mkdir -p "$MEDIA_DIR"
    if pandoc --extract-media="$MEDIA_DIR" "$FILE" -t gfm -o "$NORM/content.md" 2>/dev/null; then
        pandoc "$NORM/content.md" -t plain -o "$NORM/content.txt" 2>/dev/null || \
            pandoc "$FILE" -t plain -o "$NORM/content.txt" 2>/dev/null || true
        TEXT_EXTRACTED=true
    else
        echo "[pandoc extraction failed]" > "$NORM/content.md"
    fi
fi

# ── Fallback 1: LibreOffice headless → PDF → text ─────────────────────
if [[ "$TEXT_EXTRACTED" == false ]]; then
    if command -v soffice &>/dev/null || command -v libreoffice &>/dev/null; then
        SOFFICE="$(command -v soffice 2>/dev/null || command -v libreoffice 2>/dev/null)"
        echo "    → libreoffice fallback (render to PDF)"
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
                pdftoppm -png -r 180 -l 10 "$RENDERED" "$PREVIEW/page" 2>/dev/null || true
            fi
        fi
        rm -rf "$TMPDIR"
    fi
fi

# ── Fallback 2: python3 XML extraction (no external tools needed) ──────
if [[ "$TEXT_EXTRACTED" == false ]]; then
    if python3 -c "import zipfile,xml.etree.ElementTree" 2>/dev/null; then
        echo "    → python3 XML text extraction"
        PYTMP="$(mktemp)"
        cat > "$PYTMP" <<'PYEOF'
import zipfile, xml.etree.ElementTree as ET, sys
infile = sys.argv[1]
try:
    with zipfile.ZipFile(infile, 'r') as z:
        with z.open('word/document.xml') as f:
            tree = ET.parse(f)
    texts = []
    ns_para = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}p'
    ns_text = '{http://schemas.openxmlformats.org/wordprocessingml/2006/main}t'
    for para in tree.iter(ns_para):
        line = ''.join(t.text or '' for t in para.iter(ns_text))
        if line:
            texts.append(line)
    sys.stdout.write('\n'.join(texts))
except Exception as e:
    sys.stderr.write(f"[python3 XML extraction error: {e}]\n")
    sys.exit(1)
PYEOF
        if python3 "$PYTMP" "$FILE" > "$NORM/content.txt" 2>&1; then
            TEXT_EXTRACTED=true
        fi
        rm -f "$PYTMP"
    fi
fi

# ── Fallback 3: strings ───────────────────────────────────────────────
if [[ "$TEXT_EXTRACTED" == false ]] && command -v strings &>/dev/null; then
    echo "    → strings fallback"
    strings -n 8 "$FILE" > "$NORM/content.txt" 2>/dev/null || true
fi

# ── ZIP-based media extraction ────────────────────────────────────────
if [[ "$BASENAME" == *.docx || "$BASENAME" == *.DOCX ]]; then
    UNZIP_DIR="$NORM/unzipped"
    mkdir -p "$UNZIP_DIR"
    if unzip -q "$FILE" -d "$UNZIP_DIR" 2>/dev/null; then
        if [[ -d "$UNZIP_DIR/word/media" ]]; then
            mkdir -p "$PREVIEW"
            cp "$UNZIP_DIR/word/media"/* "$PREVIEW/" 2>/dev/null || true
        fi
        if [[ -f "$UNZIP_DIR/word/document.xml" ]]; then
            cp "$UNZIP_DIR/word/document.xml" "$NORM/document.xml" 2>/dev/null || true
        fi
    fi
fi

# Ensure content.txt exists
if [[ ! -f "$NORM/content.txt" ]]; then
    echo "[No text extracted for $BASENAME]" > "$NORM/content.txt"
fi

# Build markdown from text if pandoc didn't produce it
if [[ ! -f "$NORM/content.md" ]] || [[ "$NORM/content.md" == *"[pandoc"* ]]; then
    {
        printf '%s\n' "# Word Document: $BASENAME"
        printf '%s\n' ""
        printf '%s\n' "*Text extracted via XML/parser fallback*"
        printf '%s\n' ""
        cat "$NORM/content.txt"
    } > "$NORM/content.md"
fi

# Count media files safely
MEDIA_COUNT=0
for f in "$PREVIEW"/*; do
    [[ -f "$f" ]] || continue
    MEDIA_COUNT=$((MEDIA_COUNT + 1))
done

echo "    ✓ DOCX parsed: content.md, content.txt, $MEDIA_COUNT media/previews"
