---
name: attachment-audit
description: |
  Run a phishing and security audit on previously parsed attachment artifacts.
  Reads meta.json, risk.json, content.txt, and content.md, then produces
  a scored report across 7 security dimensions plus recommended action.

  Triggered by: "audit attachment", "check for phishing", "security check",
  "analyze this file", "is this safe", "risk assessment".
---

# /attachment-audit Command

When the user runs `/attachment-audit`, read the parsed artifacts and produce a full security assessment.

## Usage Patterns

### By Message ID
```
/attachment-audit <msgid>
```

### By File Path (auto-locate msgid directory)
```
/attachment-audit <file-or-directory>
```

### Multiple Attachments
```
/attachment-audit msgid-1 msgid-2 msgid-3
```

## Implementation Steps

1. **Locate parsed artifacts**
   - Check `./attachments/<msgid>/normalized/` for the expected files
   - Verify `meta.json`, `risk.json`, and `content.txt` exist

2. **Run the Audit Helper (Optional but recommended)**
   ```bash
   python3 -m parsers.py.phishing_audit <msgid>
   ```
   This script performs rule-based scoring and summarizes findings for your review.

3. **Read artifacts**
   ```bash
   cat ./attachments/<msgid>/normalized/meta.json
   cat ./attachments/<msgid>/normalized/risk.json
   cat ./attachments/<msgid>/normalized/content.txt
   ```

3. **Run the 7-dimension phishing audit** (from `phishing-audit` skill)
   - Extension spoofing & file type mismatch
   - Document purpose vs. sender consistency
   - Urgency & social engineering cues
   - Suspicious URLs & links
   - Credential requests
   - Security signals (yara, oleid, strings, exif)
   - Entity extraction & verification

4. **Produce scored report**
   Follow the output format defined in the `phishing-audit` SKILL.md:
   - Overall verdict: `safe-organize` | `needs-review` | `high-risk-quarantine`
   - Score: 0–10
   - Findings per dimension
   - Extracted entities
   - Recommended action

## Example Interaction

**User:** `/attachment-audit abc-123-uuid`

**Response:**
```
## Attachment Audit Report
**File:** invoice.pdf
**Sender:** billing@vendor.com
**Message-ID:** abc-123-uuid
**SHA-256:** a1b2c3d4...

### Summary Verdict
**Overall Score:** 1/10 — safe-organize

### Findings by Dimension
1. **Type Spoofing:** PASS — extension matches magic type (pdf)
2. **Brand Consistency:** PASS — sender domain matches invoice issuer
3. **Urgency/Social Engineering:** PASS — no urgency cues detected
4. **URL Safety:** PASS — no URLs in document
5. **Credential Risk:** PASS — no credential requests
6. **Security Signals:** PASS — no yara hits, no suspicious strings, no OLE risk
7. **Entity Verification:** PASS — vendor name consistent with known records

### Extracted Entities
- Organization: ACME Corp
- Invoice #: INV-2025-0042
- Amount: $1,240.00
- Due Date: 2025-06-15

### Recommended Action
✓ safe-organize — Route to Invoices/ folder and set payment reminder.

Next step:
  /inbox-organize abc-123-uuid  → classify and route
```

## Guidelines

- **Always read risk.json first.** If it shows `extension_spoofed: true` or YARA hits, escalate immediately.
- **Do not open URLs from the document.** Only report and analyze them textually.
- **Cross-reference sender domain.** If the sender is `billing@vendor.com` but the document claims to be from `vendor-support.com`, flag as brand inconsistency.
- **Handle multiple attachments.** If the user passes multiple msgids, audit each independently and return a unified summary table.
