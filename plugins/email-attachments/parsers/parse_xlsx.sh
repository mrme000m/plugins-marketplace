#!/usr/bin/env bash
# parse_xlsx.sh — Extract content from XLSX spreadsheets (OOXML)
# USAGE: parse_xlsx.sh <xlsx-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

BASENAME="$(basename "$FILE")"
echo "  [XLSX] Processing: $BASENAME"

TEXT_EXTRACTED=false

# ── Primary: Python openpyxl → text ───────────────────────────────────
if python3 -c "import openpyxl" 2>/dev/null; then
    echo "    → openpyxl text extraction"
    if python3 <<PYEOF > "$NORM/content.txt" 2>&1; then
import openpyxl, sys
try:
    wb = openpyxl.load_workbook(r"$FILE", data_only=True)
    out = []
    for sheet_name in wb.sheetnames:
        out.append(f"=== Sheet: {sheet_name} ===")
        sheet = wb[sheet_name]
        for row in sheet.iter_rows(max_row=min(sheet.max_row, 500)):
            vals = [str(cell.value) if cell.value is not None else "" for cell in row]
            out.append("\t".join(vals))
        out.append("")
    sys.stdout.write("\n".join(out))
except Exception as e:
    print(f"[openpyxl error: {e}]")
PYEOF
    TEXT_EXTRACTED=true
    fi
else
    echo "[openpyxl not available; install: pip install openpyxl]" > "$NORM/content.txt"
fi

# ── Fallback: LibreOffice → PDF → text ───────────────────────────────
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
                pdftoppm -png -r 180 -l 10 "$RENDERED" "$PREVIEW/page" 2>/dev/null || true
            fi
        fi
        rm -rf "$TMPDIR"
    fi
fi

# ── CSV variant ───────────────────────────────────────────────────────
if command -v in2csv &>/dev/null; then
    in2csv "$FILE" > "$NORM/content.csv" 2>/dev/null || true
fi

# ── Markdown wrapper ──────────────────────────────────────────────────
{
    printf '%s\n' "# Spreadsheet: $BASENAME"
    printf '%s\n' ""
    printf '%s\n' "*Sheets extracted as tab-delimited tables below*"
    printf '%s\n' ""
    cat "$NORM/content.txt"
} > "$NORM/content.md"

echo "    ✓ XLSX parsed: content.txt, content.md"
