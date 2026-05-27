---
name: parse-attachment
description: |
  Parse an email attachment file using the deterministic CLI extraction pipeline.
  Detects file type, computes hash, produces content.txt, content.md, meta.json,
  risk.json, and preview images. Can process single files or entire directories.

  Triggered by: "parse attachment", "extract text", "process file",
  "normalize attachment", "parse this PDF/DOCX/image".
---

# /parse-attachment Command

When the user runs `/parse-attachment`, parse one or more files using the CLI pipeline and report results.

## Usage Patterns

### Single File
```
/parse-attachment <file> [--sender "..."] [--msgid "..."]
```

### Directory of Files
```
/parse-attachment <directory> [--sender "..."] [--msgid "..."]
```

### With Sender Context
```
/parse-attachment ./invoice.pdf --sender "billing@vendor.com" --msgid "msg-123@example.com"
```

## Implementation Steps

1. **Locate the file(s)**
   - If path is a file: process it via `parsers/parse_attachment.sh`
   - If path is a directory: iterate all files, process each
   - If relative path: resolve from cwd

2. **Run the orchestrator**
   ```bash
   ./parsers/parse_attachment.sh <file> \
     --sender "<sender>" \
     --msgid "<msgid>"
   ```
   The orchestrator handles type detection, hashing, security scanning, and routes to the correct typed parser.

3. **Report results to user**
   Show:
   - Message ID (automatically generated from filename if not provided)
   - Detected type (from magic bytes)
   - SHA-256 hash
   - Extension spoofed? (yes/no)
   - Security signals from risk.json (yara hits, suspicious strings, ole risk)
   - Output paths for inspection

## Example Interaction

**User:** `/parse-attachment ~/Downloads/invoice.pdf`

**Response:**
```
============================================================
► ATTACHMENT PROCESSED: invoice.pdf
  Detected Type: pdf (application/pdf)
  Message-ID:    invoice-pdf-a1b2c3d4
  SHA-256:       a1b2c3d4e5f6g7h8...

  SECURITY SIGNALS:
    Extension Match: PASS
    YARA Hits:       None
    Suspicious Str:  None

  ARTIFACTS SAVED:
    Text Content:  attachments/invoice-pdf-a1b2c3d4/normalized/content.txt
    Markdown:      attachments/invoice-pdf-a1b2c3d4/normalized/content.md
    Risk Report:   attachments/invoice-pdf-a1b2c3d4/normalized/risk.json
============================================================

  Next steps:
    /attachment-audit invoice-pdf-a1b2c3d4  → security assessment
    Read content: cat attachments/invoice-pdf-a1b2c3d4/normalized/content.txt
```

## Error Handling

- **File not found:** Prompt user to verify path.
- **Missing CLI tools:** Tell user which tool is missing and how to install (brew/apt).
- **Parser failure:** Report which parser failed and that fallback generic parser was used.
- **Directory contains no files:** Report empty directory.
