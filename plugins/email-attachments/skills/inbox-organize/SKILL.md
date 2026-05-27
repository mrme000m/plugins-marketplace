---
name: inbox-organize
description: |
  Classify and route email attachments after parsing and security audit.
  Determines inbox folder, priority, action items, and metadata enrichment
  for safe attachments. Companion skill to email-attachments and phishing-audit.

  **Trigger phrases:** "organize this email", "route attachment", "inbox triage",
  "classify attachment", "what folder", "priority", "action items", "email workflow",
  "sort attachments", "archive", "label email", "categorize"
---

# Inbox Organization & Classification Skill

## When to Use

Use this skill after the attachment has been parsed (via `email-attachments` skill) and cleared by security audit (via `phishing-audit` skill with verdict `safe-organize` or low-risk `needs-review`). This skill answers: *where does this email/attachment belong?* and *what action is required?*

## Input

Use the same normalized artifacts:
- `content.txt` / `content.md` — extracted document content
- `meta.json` — file metadata (sender, filename, MIME, hash)
- `risk.json` — security signals
- (Optional) `preview/` — visual thumbnails

## Classification Dimensions

### 1. Document Category
Determine which category best fits the attachment:

| Category | Examples |
|----------|----------|
| `invoice` | Payment request, billing statement, receipt |
| `contract` | Legal document, terms, SLA, NDA |
| `receipt` | Purchase confirmation, order summary |
| `report` | Analytics, metrics, status report |
| `resume` | CV, cover letter, candidate profile |
| `newsletter` | Marketing, promotional, update |
| `it-alert` | Security notice, system update, password change |
| `personal` | Personal communication, non-work |
| `spam` | Unsolicited, irrelevant |
| `other` | Does not fit above categories |

### 2. Priority Level

| Level | Criteria |
|-------|----------|
| `critical` | Payment due, contract deadline, security incident |
| `high` | Action required within 48 hours, escalated request |
| `medium` | Routine business, review requested |
| `low` | FYI, newsletter, no action needed |
| `archive` | Already handled, reference only |

### 3. Action Items
Extract specific actions implied by the document:
- `pay` — payment required
- `sign` — signature or approval needed
- `review` — document needs reading/feedback
- `reply` — response requested
- `schedule` — meeting or call scheduling
- `none` — no action required

### 4. Folder/Label Suggestion
Suggest an inbox folder or tag based on category + sender:
- `Invoices/` — all payment/billing documents
- `Contracts/` — legal and agreements
- `IT-Alerts/` — security and system notices
- `Candidates/` — resumes and HR docs
- `Newsletters/` — marketing, optional reads
- `Archive/` — handled and reference-only
- `Quarantine/` — (from phishing-audit, not from this skill)

### 5. Entity Summary
Extract useful entities for filing/search:
- Invoice number, PO number, contract ID
- Vendor/company name
- Amount (if financial)
- Due date
- Key contact person

## Output Format

```
## Inbox Classification Report
**File:** <filename>
**Sender:** <sender>
**Category:** <category>
**Priority:** <critical|high|medium|low|archive>

### Suggested Folder
<folder name>

### Action Items
1. [action] — description
2. [action] — description

### Key Entities
- Invoice #:
- Amount:
- Due Date:
- Vendor:
- Contact:

### Summary
<2-3 sentence natural language summary of what the document is and why it was classified here>

### Tags
#category #priority #sender-domain #action-type
```

## Workflow

### 1. Single Attachment Classification

After parsing and security audit:
```bash
# Read the normalized content
cat attachments/<msgid>/normalized/content.txt
cat attachments/<msgid>/normalized/meta.json

# Run classification
category=$(determine_category)
priority=$(determine_priority)
folder=$(suggest_folder)
actions=$(extract_actions)
entities=$(extract_entities)
```

### 2. Batch Triage

For multiple attachments from the same email:
```bash
# Process all attachments for a message
for dir in attachments/<msgid-prefix>*/; do
  cat "$dir/normalized/meta.json"
  cat "$dir/normalized/content.txt"
  # Classify each and produce unified report
done
```

### 3. Integration with Security Audit

The inbox-organize skill must only be invoked after phishing-audit clears the attachment:
- If verdict is `safe-organize` (0–2): classify normally
- If verdict is `needs-review` (3–6): classify but add `[REVIEW REQUIRED]` flag and place in `Review/` folder
- If verdict is `high-risk-quarantine` (7–10): **DO NOT classify**. Route to `Quarantine/` and alert.

## Examples

**User:** "Where should this invoice go?"
→ Read parsed content. Identify as `invoice` category.
→ Check for due date and amount. Set priority (critical if past due, high if within 7 days).
→ Suggest `Invoices/` folder. Extract invoice #, amount, due date as entities.
→ Action items: `pay`, `archive`.

**User:** "Organize these 3 email attachments"
→ Loop through each attachment's normalized output.
→ Classify each independently.
→ Produce a unified report showing all 3 with their folders and priorities.

**User:** "What action do I need to take on this?"
→ Read content for explicit action requests ("please review and sign", "payment due by X").
→ If no explicit request, check if it's FYI or reference material.
→ Output action items with deadlines if stated.

## Guidelines

- **Never classify before security audit.** Always check `risk.json` or confirm the phishing-audit verdict is `safe-organize` before routing to regular folders.
- **Preserve evidence.** Keep the full `attachments/<msgid>/` directory tree intact. Classification reports reference these paths, not the original email.
- **Use extracted entities for searchability.** Include invoice numbers, amounts, and vendor names in the classification output so they are searchable later.
- **Multi-attachment emails.** If an email has multiple attachments, classify each independently, then produce a unified summary. Quarantine the entire message if ANY attachment fails security audit.
- **Respect user conventions.** If the user has established folder names, use those. Default to the suggested folders above only when the user has no existing system.
