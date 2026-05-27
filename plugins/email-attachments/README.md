# Email Attachments Plugin

A hybrid CLI + Claude Code pipeline for ingesting, normalizing, and securing email attachments. Powered by a Python-based processing engine with support for high-fidelity tools like **Apryse CLI**, **Pandoc**, and **Tesseract OCR**.

## Features

- **Multi-Format Support**: High-fidelity extraction for PDF, DOCX, XLSX, PPTX, Images, and EML/ICS.
- **High-Quality Extraction**: Integration with **Apryse CLI** (`pdf2text`, `docpub`) for superior results.
- **Security Audit**: Automated YARA rule scanning, suspicious string detection, and extension spoof checking.
- **Normalization**: Every attachment yields structured text, markdown, metadata, and preview images.
- **Claude-Native**: Designed to feed normalized artifacts directly into Claude for reasoning-heavy tasks like phishing audits and triage.

## Installation

```bash
# Install the plugin from this marketplace
/plugin marketplace add https://github.com/mrme000m/plugins-marketplace
/plugin install email-attachments@plugins-marketplace
```

## Prerequisites

### High-Performance Option (Recommended)
- **Apryse CLI**: Install `pdf2text` and `docpub` (formerly PDFTron).
- Set `APRYSE_LICENSE_KEY` in your environment.

### Standard Option (Open Source)
- **macOS**: `brew install poppler pandoc tesseract imagemagick exiftool file yara libreoffice`
- **Linux**: `apt-get install poppler-utils pandoc tesseract-ocr imagemagick exiftool file yara libreoffice`
- **Python**: `pip install openpyxl python-pptx oletools`

## Usage

| Command | Description |
|---------|-------------|
| `/attachment-help` | Show all capabilities and available parsers |
| `/parse-attachment <file>` | Parse a file into normalized artifacts |
| `/attachment-audit <msgid>` | Run a security audit using rule-based signals + Claude reasoning |

## Architecture

1.  **Ingestion**: File is hashed (SHA-256) and MIME-typed.
2.  **Orchestration**: `attachment_processor.py` identifies the best parser and runs security checks.
3.  **Parsing**: specialized parsers (PDF, Office, etc.) extract text and generate previews.
4.  **Security**: `risk.json` captures YARA hits, suspicious strings, and magic mismatches.
5.  **Audit/Triage**: Claude uses the normalized outputs to score risks and suggest routing.

## Output Structure

Results are stored in `./attachments/<msgid>/normalized/`:
- `content.txt` / `content.md`: Extracted text/markdown.
- `meta.json`: File metadata, sender info, and tool availability.
- `risk.json`: Security signals and risk assessment findings.
- `preview/`: Page/slide thumbnails and cleaned image versions.

## License

MIT
