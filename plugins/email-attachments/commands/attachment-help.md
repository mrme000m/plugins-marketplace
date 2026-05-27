---
name: attachment-help
description: |
  Progressive discovery menu for the email-attachments plugin.
  Shows all parsers, skills, commands, and the attachment pipeline workflow.
  Triggered by: "help", "what can you do", "attachment capabilities",
  "show commands", "email plugin help".
---

# email-attachments Plugin — Capabilities

When the user runs `/attachment-help`, `/help email`, or asks "what can you do", show this progressive discovery menu.

## Output Format

```
Email Attachments Plugin — Capabilities
=======================================

Quick commands:
  /attachment-help    — Show this help menu
  /parse-attachment   — Parse an attachment: detect type, extract text, build metadata
  /attachment-audit   — Run phishing/security audit on parsed artifacts

[1] Attachment Pipeline (Python Engine)
    — High-fidelity parsing of PDF, DOCX, XLSX, PPTX, images, text, and binaries
    — Support for Apryse CLI (pdf2text, docpub) for best-in-class conversion
    — Smart identification using magic bytes + extension spoof detection
    — Automatic OCR fallback for scanned or image-based PDFs
    — Outputs: content.txt, content.md, meta.json, risk.json, preview/

[2] Security Audit (SKILL: phishing-audit)
    — Automated YARA rule scanning for phishing and malware signatures
    — Noise-filtered suspicious string detection (PowerShell, base64, URLs)
    — Rule-based risk scoring (0-10) and summary verdicts
    — 7-dimension analysis: spoofing, brand, urgency, URLs, credentials, signals, entities

[3] Inbox Organization (SKILL: inbox-organize)
    — Intelligent classification into categories (invoice, contract, resume, etc.)
    — Priority assessment and automated folder suggestions
    — Entity extraction (invoice #, amount, due date, vendor)

## Usage Examples

Parse a single file:
  /parse-attachment ./invoice.pdf --sender billing@vendor.com

Audit parsed results:
  /attachment-audit <msgid>

## Prerequisites

High-Performance Option:
  Apryse CLI (pdf2text, docpub) + APRYSE_LICENSE_KEY

Standard Option:
  poppler, pandoc, tesseract, imagemagick, exiftool, yara, libreoffice
  pip install -r plugins/email-attachments/requirements.txt

## Architecture

  attachment.file
      │
      ▼  attachment_processor.py (orchestrator)
      ├── Detect type (magic + MIME) + Hashing
      ├── Security Scanning (YARA + Strings)
      ├── Route to typed parser (pdf, docx, xlsx, pptx, image, text, generic)
      │   └── Falls back through Apryse -> Specialized Libs -> CLI Tools -> XML/Binary
      │
      ▼  Normalized artifacts (attachments/<msgid>/normalized/)
      ├── content.txt / content.md
      ├── meta.json / risk.json
      └── preview/ (images/thumbnails)
```
