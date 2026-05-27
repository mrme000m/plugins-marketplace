# Email Downloader Plugin

A modular, extendable, and versatile email downloader optimized for agentic usage.

## Features

- **IMAP Support**: Works with any standard IMAP server.
- **Robust Filtering**: Filter by sender, subject, unread status, date, and limit.
- **Markdown Normalization**: Automatically converts email bodies to Markdown for easy reading by agents.
- **Attachment Extraction**: Saves attachments into structured directories.
- **Attachment Parsing**: Optional integration with `email-attachments` plugin to normalize and audit attachments.
- **JSON Output**: Machine-readable output for easy integration with other tools.

## Structure

- `src/email_downloader.py`: Core logic and CLI entry point.
- `commands/`: Slash commands for Claude Code.
- `skills/`: Agentic skill definition.

## Setup

Set the following environment variables (or use a `.env` file):

```bash
EMAIL_USERNAME="your@email.com"
EMAIL_PASSWORD="your_password"
EMAIL_IMAP_SERVER="mail.yourserver.com"
```

## Usage

### Slash Commands

- `/email-download`: Download emails with filters.
- `/email-list`: List folders or email summaries.

### CLI

```bash
python3 plugins/email-downloader/src/email_downloader.py --unread --limit 5 --parse
```

## Integration with Email Attachments

If the `email-attachments` plugin is present in the workspace, using the `--parse` flag will automatically trigger the attachment parsing pipeline for every downloaded file.
