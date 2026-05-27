# Attachment Parsing Routing Rules

## Rule: Prefer bash for simple files, python3 for complex ones

**Bash** handles fast, predictable, byte-level operations.  
**Python3** is used for structured data that requires RFC-compliant parsing, recursion, or MIME decoding.

## Routing Matrix

| File Type | Complexity | Routed To | Why |
|-----------|-----------|-----------|-----|
| `.txt`, `.log`, `.md`, `.csv` | Simple | `parse_text.sh` (bash) | Raw copy, no decoding needed |
| `.eml`, `.msg` | Complex | `parse_text.sh` or python3 | MIME multipart, recursive attachments, base64 encoding |
| `.ics` | Medium | `parse_text.sh` (python3 inline) | VEVENT parsing with structured key-value extraction |
| `.html`, `.htm` | Medium | `parse_text.sh` (bash) | Strip tags with `sed`, raw copy fallback |
| `.pdf` | Complex | `parse_pdf.sh` (bash + tools) | `pdftotext`/`pdfinfo` are CLI binaries, bash orchestrates |
| `.docx`, `.xlsx`, `.pptx` | Complex | Python3 | ZIP-based XML structure, need python `zipfile` + `lxml` |
| `.doc`, `.xls`, `.ppt` | Complex | Python3 | OLE2 binary format, need python `olefile` or `textract` |
| `.jpg`, `.png`, `.gif`, `.bmp` | Medium | `parse_image.sh` (bash) | `exiftool` + `tesseract` OCR, bash pipelines |
| `.zip`, `.7z`, `.rar`, `.gz`, `.tar` | Complex | Python3 | Archive traversal, nested extraction, safe path handling |
| Unknown / binary | Fallback | `parse_generic.sh` (bash) | `file`, `strings`, `hexdump`, `xdd` | |` |


## Multi-line File Type Detectors

When routing, always run `file --brief --mime-type` first, then use extension for `message/rfc822` → `.eml` routing (since `.eml` files may not have a unique MIME type across all systems).

## Embedded Attachment Handling

EML files may contain nested attachments. Rules:
1. Parse the EML envelope (headers, body) via python3
2. Extract and save each embedded attachment to `embedded/`
3. Recursively call `parse_router.sh` on each embedded attachment
4. Summarize all embedded attachments in `embedded_index.json`

## Performance Thresholds

| Threshold | Action |
|-----------|--------|
| File < 50 KB and plain text | Bash, immediate |
| File > 50 MB | Warn user, stream don't load to memory |
| EML with > 10 attachments | Process in batches, log progress |
| PDF with > 100 pages | Limit preview to first 10 pages |

## Error Fallbacks

| Condition | Fallback |
|-----------|----------|
| python3 unavailable | Use bash `sed`/`awk` or raw copy |
| CLI tool missing (pdftotext, etc.) | Use python library or generic fallback |
| Parser crash | `parse_generic.sh` on the raw file |
| Recursive depth > 5 | Stop recursion, log as "deeply nested" |

## Summary

```
                   parse_router.sh
                         │
        ┌────────────────┼────────────────┐
        │ plain text     │  MIME enc     │  structured binary
        │  (< 50 KB)     │  (EML, ICS)   │  (DOCX, XLSX, PDF)
        ▼                ▼               ▼
   parse_text.sh    text_parser.py   python3 modules
   (bash)           or               (docx_parser.py, etc.)
                    parse_text.sh
                    (python3 inline)
```
