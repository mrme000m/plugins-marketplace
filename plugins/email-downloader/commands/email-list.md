---
name: email-list
description: List mailboxes or email summaries.
---

# /email-list Command

List available IMAP folders or show a summary of emails in a specific folder.

## Usage

```
/email-list [--mailbox MAILBOX] [--limit N] [--unread] [--sender SENDER] [--folders]
```

## Options

- `--folders`: List all available mailboxes/folders on the server.
- `--mailbox`: Specify which folder to list emails from (default: INBOX).
- `--limit`: Number of emails to summarize.
- `--unread`: Only show unread emails.
- `--sender`: Filter by sender.

## Example

### List all folders
`/email-list --folders`

### List last 10 emails in INBOX
`/email-list --limit 10`
