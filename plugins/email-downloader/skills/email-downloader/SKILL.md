---
name: email-downloader
description: Download and filter emails via IMAP, save to markdown/raw EML, and optionally parse attachments.
---

# email-downloader Skill

The `email-downloader` skill allows you to interact with email servers via IMAP. You can search for emails, download them as Markdown (for easy reading) and raw EML files, and extract attachments.

## Usage Guidelines

- **Search first:** If you are looking for specific emails, use filters like `--sender`, `--subject`, or `--unread`.
- **Limit results:** When downloading, use `--limit` to avoid overwhelming your context with too many emails.
- **Parse attachments:** Use the `--parse` flag if you want to automatically run the `email-attachments` parser on any downloaded files.
- **Credentials:** Ensure `EMAIL_USERNAME`, `EMAIL_PASSWORD`, and `EMAIL_IMAP_SERVER` are set in the environment or a `.env` file.

## Examples

### Download unread emails
```bash
python3 plugins/email-downloader/src/email_downloader.py --unread --limit 5 --parse
```

### Search for emails from a specific sender
```bash
python3 plugins/email-downloader/src/email_downloader.py --sender "billing@service.com" --limit 10
```

### Search by subject keyword
```bash
python3 plugins/email-downloader/src/email_downloader.py --subject "invoice" --output ./invoices
```

## Tool Interface

The core logic is in `plugins/email-downloader/src/email_downloader.py`.

| Argument | Description |
|----------|-------------|
| `--username` | IMAP username |
| `--password` | IMAP password |
| `--server` | IMAP server address |
| `--unread` | Filter for only unread emails |
| `--sender` | Filter by From address |
| `--subject` | Filter by subject keyword |
| `--since` | Filter since date (DD-Mon-YYYY) |
| `--limit` | Max number of emails to fetch |
| `--parse` | Trigger attachment parsing |
| `--json` | Output machine-readable JSON |
| `--output` | Directory to save files (default: attachments) |
