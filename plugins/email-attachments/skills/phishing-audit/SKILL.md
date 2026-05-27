---
name: phishing-audit
description: |
  Analyze email attachment artifacts (content.txt, meta.json, risk.json, previews)
  for phishing indicators, social engineering cues, urgency manipulation, spoofed
  branding, credential harvesting attempts, and suspicious entities.

  **Trigger phrases:** "phishing audit", "check for phishing", "is this legit",
  "suspicious attachment", "security assessment", "email threat", "malicious file",
  "audit attachment", "phishing analysis", "social engineering", "fraud check",
  "verify this invoice", "check this email", "risk assessment"
---

# Phishing & Security Audit Skill

## When to Use

Use this skill when the user asks to audit, verify, or assess an email attachment for phishing, fraud, social engineering, or general security risk. Also use when `risk.json` from the attachment pipeline flags any red flags (spoofed extension, YARA hits, suspicious strings, ole risk, EXIF warnings).

## Audit Input

You receive these normalized artifacts from the CLI pipeline:

| Artifact | Purpose |
|----------|---------|
| `content.txt` | Extracted plain text from the document |
| `content.md` | Markdown version with formatting preserved |
| `meta.json` | File metadata: filename, MIME, size, hash, sender, msgid, spoof flags |
| `risk.json` | Security signals: yara, oleid, strings, magic mismatch, EXIF warnings |
| `preview/` | Page/slide/thumbnail images for visual inspection |

## Audit Dimensions

Analyze across these 7 dimensions and provide scored findings:

### 1. Extension Spoofing & File Type Mismatch
- Does `meta.json` show `extension_spoofed: true`?
- Is the file masquerading as a benign type?
- Score: HIGH if spoofed, LOW otherwise

### 2. Document Purpose vs. Sender
- What does the document claim to be (invoice, receipt, action required)?
- Does the sender domain match the claimed sender in the document?
- Is the document self-consistent ( logos, addresses, phone numbers, URLs)?
- Score: HIGH if branding mismatch, LOW if consistent

### 3. Urgency & Social Engineering
- Look for urgency triggers: "24 hours", "account suspended", "immediate action", "expires today"
- Look for authority impersonation: gov agencies, banks, IT departments, CEOs
- Look for fear/shame: "unauthorized activity detected", "legal action"
- Score: HIGH if multiple urgency cues present, LOW if absent

### 4. Suspicious URLs & Links
- Extract all URLs from content.txt
- Check for:
  - Typosquats (rn → m, 0 → o, l → 1)
  - Homograph attacks (punycode, unicode lookalikes)
  - IP-based URLs
  - Suspicious TLDs
  - URL shorteners
- Score: HIGH if credential-harvesting URLs present; MEDIUM for suspicious; LOW for none

### 5. Credential Requests
- Does the document ask for passwords, 2FA codes, SSN, bank details?
- Are form fields or QR codes present that lead to credential submission?
- Score: CRITICAL if credential collection is implied

### 6. Security Signals Integration
- Does `risk.json` contain:
  - `yara_hits` from phishing/malware rules?
  - `suspicious_strings` containing script references, encoded commands?
  - `ole_risk` from macro-bearing Office documents?
  - `exif_warnings` (GPS, embedded scripts, executable markers)?
- Score: CRITICAL for YARA hits, HIGH for OLE/macro risk, MEDIUM for suspicious strings

### 7. Entity Extraction & Verification
- Extract named entities: organizations, people, phone numbers, bank accounts, invoice numbers
- Flag any entities that do not align with the sender domain
- Flag unexpected payment requests, wire instructions, or cryptocurrency addresses

## Output Format

Provide findings in this structure:

```
## Attachment Audit Report
**File:** <filename>
**Sender:** <sender>
**Message-ID:** <msgid>
**SHA-256:** <hash>

### Summary Verdict
| Verdict | Score | Explanation |
|---------|-------|-------------|
| safe-organize | 0-2/10 | Clean attachment, routine business content |
| needs-review   | 3-6/10 | Some anomalies or unverified claims |
| high-risk-quarantine | 7-10/10 | Clear phishing indicators or active threat |

**Overall Score:** X/10

### Findings by Dimension
1. **Type Spoofing:** [PASS / FLAG / FAIL] — details
2. **Brand Consistency:** [PASS / FLAG / FAIL] — details
3. **Urgency/Social Engineering:** [PASS / FLAG / FAIL] — details
4. **URL Safety:** [PASS / FLAG / FAIL] — details
5. **Credential Risk:** [PASS / FLAG / FAIL] — details
6. **Security Signals:** [PASS / FLAG / FAIL] — details (reference risk.json)
7. **Entity Verification:** [PASS / FLAG / FAIL] — details

### Extracted Entities
- Organization: X
- Phone: X
- URL: X
- Invoice #: X
- Bank/Wire: X

### Recommended Action
- If safe-organize: summarize and route to inbox folder
- If needs-review: flag for manual review, do not action links
- If high-risk-quarantine: recommend quarantining, warn sender if internal, document hash
```

## Workflow

### 1. Read the Artifacts

```bash
# Let the user know which files to inspect
cat attachments/<msgid>/normalized/meta.json
cat attachments/<msgid>/normalized/risk.json
cat attachments/<msgid>/normalized/content.txt
```

### 2. Score and Classify

Apply the 7-dimension analysis above. For each dimension:
- **PASS**: No concerning findings
- **FLAG**: Anomaly detected but not conclusive
- **FAIL**: Clear indicator of phishing or threat

Calculate overall score (0–10) and assign verdict.

### 3. Recommend Action

Route into one of three outcomes:
- **safe-organize** (0–2): Summarize, extract entities, place in appropriate folder.
- **needs-review** (3–6): Flag for human review, preserve evidence, do not execute links.
- **high-risk-quarantine** (7–10): Quarantine file, log hash, alert, do not open links.

## Examples

**User:** "Audit this attachment for phishing" (after parsing)
→ Read `meta.json`, `risk.json`, `content.txt` from the parsed directory.
→ Run the 7-dimension analysis and produce the structured report.
→ Recommend `safe-organize`, `needs-review`, or `high-risk-quarantine`.

**User:** "This invoice looks suspicious"
→ Check sender domain vs. invoice issuer.
→ Check for urgency cues ("Pay within 24 hours or service terminates").
→ Check payment instructions against known vendor accounts.
→ Check URLs for typosquats.
→ Score and report.

**User:** "Is this PDF safe?"
→ Check `risk.json` first for spoofs, yara, strings.
→ Check `meta.json` for MIME/extension alignment.
→ Read content for social engineering, credential requests, suspicious URLs.
→ If visual analysis needed (fake invoices, spoofed logos), inspect preview images.
→ Score and report.
