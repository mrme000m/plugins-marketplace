---
name: email-otp
description: Fetch OTP codes from email inboxes via IMAP with pluggable providers and service-specific templates.
---

# email-otp Command

When the user needs to fetch a verification code or OTP from their email, use this command.

## Usage

```bash
python3 plugins/email-otp/src/cli.py [options]
```

## Options

| Option | Description |
|--------|-------------|
| `--provider` | Email provider: `gmail`, `icloud`, `outlook`, `custom` (default: `gmail`) |
| `--email` | Email address |
| `--app-password` | App-specific password |
| `--server` | Custom IMAP server (for `--provider custom`) |
| `--port` | IMAP port (default: 993) |
| `--template` | OTP template: `bitwarden`, `github`, `google`, `aws`, `generic` (default: auto-detect) |
| `--sender` | Filter by sender email address |
| `--subject` | Filter by subject keyword |
| `--unseen` | Only search unread emails |
| `--max-emails` | Maximum emails to check (default: 10) |
| `--no-mark-seen` | Don't mark processed emails as read |
| `--json` | Output JSON instead of plain text |
| `--list-providers` | List available email providers |
| `--list-templates` | List available OTP templates |

## Examples

### Fetch Bitwarden OTP from Gmail
```bash
python3 plugins/email-otp/src/cli.py --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx" --template bitwarden
```

### Fetch GitHub 2FA code from iCloud
```bash
python3 plugins/email-otp/src/cli.py --provider icloud --email user@icloud.com --app-password "xxxx-xxxx-xxxx-xxxx" --template github
```

### Auto-detect OTP from any service
```bash
python3 plugins/email-otp/src/cli.py --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx"
```

### JSON output for programmatic use
```bash
python3 plugins/email-otp/src/cli.py --provider gmail --email user@gmail.com --app-password "$GMAIL_APP_PASSWORD" --template bitwarden --json
```

## Credential Resolution

1. CLI arguments (`--email`, `--app-password`)
2. Environment variables (`EMAIL_USERNAME`, `EMAIL_PASSWORD`, `EMAIL_PROVIDER`)
3. `.env` file (`~/.config/email-otp/.env`)
4. macOS Keychain (service: `email-otp.<provider>.password`)

## App Password Setup

- **Gmail:** https://myaccount.google.com/apppasswords
- **iCloud:** Apple ID → Sign-In and Security → App-Specific Passwords
- **Outlook:** https://account.microsoft.com/security → App passwords
