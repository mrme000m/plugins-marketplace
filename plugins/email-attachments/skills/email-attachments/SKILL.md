---
name: email-attachments
description: |
  Normalize and process email attachments using a hybrid CLI + Claude pipeline.
  Parses PDF, DOCX, XLSX, PPTX, images, text, and unknown binary files into
  extracted text, metadata, preview images, and security risk signals.

  **Trigger phrases:** "parse attachment", "email attachment", "process PDF",
  "extract from docx", "OCR image", "attachment pipeline", "email security",
  "normalize attachment", "extract text from file", "file hash", "MIME type"
---

# Email Attachment Processing Pipeline

## Overview

Use this skill when the user needs to extract, classify, or secure-assess email attachments or any file-based document. The pipeline follows this flow:

1. **Ingest**: Save with original filename, compute SHA-256, detect MIME type via magic bytes.
2. **Normalize**: Run deterministic extractors (Python-based with CLI fallfalls) to produce `content.txt`, `content.md`, `meta.json`, `risk.json`, and `preview/` images.
3. **Audit**: Pass normalized artifacts to Claude for phishing analysis, entity extraction, urgency detection, and policy routing.

## Architecture

```
attachment.file
    │
    ▼
parse_router.sh (complexity-based router)
    ├── Simple files (< 50 KB, plain text) → parse_text.sh (bash)
    ├── EML / multipart MIME → python3 (text_parser.py — recursive embedded handling)
    ├── PDF, DOCX, XLSX, PPTX → python3 (specialized parsers)
    │   ├── PDF → bash orchestrates CLI tools (pdftotext, pdfinfo, pdftoppm)
    │   ├── DOCX → python3 (Apryse docpub | pandoc | LibreOffice | XML parse)
    │   ├── XLSX → python3 (Apryse docpub | openpyxl | LibreOffice)
    │   └── PPTX → python3 (Apryse docpub | python-pptx | LibreOffice)
    ├── Images → bash (exiftool + imagemagick + tesseract OCR)
    ├── Archives → python3 (safe traversal)
    └── Unknown binaries → generic_parser (strings + hex dump fallback)
    │
    ▼
normalized/
    content.txt         — plain text extraction
    content.md          — markdown extraction (with previews)
    meta.json           — file metadata, sender, msgid, hashes
    risk.json           — security signals (yara, oleid, strings, spoofing)
    embedded/           — extracted nested attachments from EML
    embedded_index.json — summary of all embedded attachments
    preview/            — page/slide/thumbnail images
```

## Prerequisites

Install the tool stack before using the pipeline:

**High-Performance Option (Recommended):**
- **Apryse CLI**: Install `pdf2text` and `docpub` for best-in-class Office-to-PDF and PDF-to-Text conversion.
- Set `APRYSE_LICENSE_KEY` in your environment.

**Standard Option (Open Source):**
**macOS (Homebrew):**
```bash
brew install poppler pandoc tesseract imagemagick exiftool file yara oletools libreoffice
pip install openpyxl python-pptx oletools
```

**Debian/Ubuntu:**
```bash
sudo apt-get install poppler-utils pandoc tesseract-ocr imagemagick exiftool file yara libreoffice
pip install openpyxl python-pptx oletools
```

## Quick Reference

| Task | Command |
|------|---------|
| Parse a single file (via router) | `bash parsers/parse_router.sh ./file.eml ./out ./preview` |
| Parse via processor | `python3 -m parsers.py.attachment_processor ./invoice.pdf` |
| Parse with email context | `python3 -m parsers.py.attachment_processor ./file.docx --sender "user@example.com"` |
| Show capabilities | `/attachment-help` |
| Run security audit | `/attachment-audit <msgid>` |
| Inspect text | `cat attachments/<msgid>/normalized/content.txt` |
| View signals | `cat attachments/<msgid>/normalized/risk.json` |
| View embedded attachments | `cat attachments/<msgid>/normalized/embedded_index.json` |

## Routing Rules

The pipeline routes files based on complexity:

| File Type | Routed To | Complexity |
|-----------|-----------|-----------|
| `.txt`, `.log`, `.md`, `.csv` | `parse_text.sh` (bash) | Simple — raw copy, no decoding |
| `.eml`, `.msg` | `text_parser.py` (python3) | Complex — MIME multipart, recursion, base64 |
| `.html`, `.htm` | `parse_text.sh` (bash) | Medium — strip tags with `sed` or raw copy |
| `.pdf` | `parse_pdf.sh` (bash + tools) | Complex — orchestrates pdftotext/pdfinfo/pdftoppm |
| `.docx`, `.xlsx`, `.pptx` | Python3 modules | Complex — ZIP-based XML, zipfile + lxml |
| `.doc`, `.xls`, `.ppt` | Python3 modules | Complex — OLE2 binary, olefile / textract |
| `.jpg`, `.png`, `.gif` | `parse_image.sh` (bash) | Medium — exiftool + tesseract OCR |
| `.zip`, `.7z`, `.rar`, `.gz` | Python3 | Complex — archive traversal, safe extraction |
| Unknown / binary | `parse_generic.sh` (bash) | Fallback — file, strings, hexdump |

## Workflow

### 1. Single File Parse

```bash
# Process an attachment using the Python-based pipeline
python3 -m parsers.py.attachment_processor ~/Downloads/invoice.pdf \
  --sender "billing@vendor.com" \
  --msgid "abc123@mail.example.com"
```

### 2. Inspect Results

```bash
# Check for red flags immediately
cat attachments/<msgid>/normalized/risk.json
# Extract entities or verify content
cat attachments/<msgid>/normalized/content.md
```

## Guidelines

- **Always include sender and msgid.** The orchestrator context is vital for downstream analysis.
- **Apryse CLI Support.** The pipeline automatically uses `pdf2text` and `docpub` if available for superior quality.
- **OCR is an automatic fallback.** Scanned documents are processed via Tesseract if text yield is low.
- **Security First.** Check `risk.json` for spoofed extensions or YARA hits before processing content.
- **Binary Safety.** The generic parser ensures every file yields at least strings and hex data for analysis.
- **Do not pass raw binaries to Claude.** Always use the normalized text and metadata.

## Examples

**User:** Parse this PDF invoice attachment for me.
```bash
python3 -m parsers.py.attachment_processor ~/Downloads/invoice.pdf --sender "billing@vendor.com"
cat attachments/invoice-pdf-*/normalized/content.md
```

**User:** Extract text from this Word document and check for security issues.
```bash
python3 -m parsers.py.attachment_processor ~/Documents/contract.docx --sender "partner@example.com"
cat attachments/contract-docx-*/normalized/content.txt
cat attachments/contract-docx-*/normalized/risk.json
```

**User:** Process this email file with all its nested attachments.
```bash
python3 -m parsers.py.attachment_processor ~/Downloads/message.eml --sender "sender@domain.com"
cat attachments/message-eml-*/normalized/embedded_index.json
```
