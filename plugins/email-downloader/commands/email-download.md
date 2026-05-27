---
name: email-download
description: Download emails via IMAP with filtering.
---

# /email-download Command

Download emails from an IMAP server and save them as Markdown and raw EML.

## Usage

```
/email-download [--unread] [--sender SENDER] [--subject KEYWORD] [--limit N] [--parse]
```

## Options

- `--unread`: Only fetch unread messages.
- `--sender`: Filter by sender email.
- `--subject`: Filter by subject keyword.
- `--limit`: Maximum number of emails to download.
- `--parse`: Automatically run attachment parsers on downloaded files.

## Example

`/email-download --unread --limit 5 --parse`
