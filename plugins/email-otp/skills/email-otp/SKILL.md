---
name: email-otp
description: |
  Fetch OTP codes from email inboxes via IMAP. Supports Gmail, iCloud,
  and Outlook providers with service-specific extraction templates for
  Bitwarden, GitHub, Google, AWS, and generic OTP emails.

  **Trigger phrases:** "fetch OTP", "get verification code", "email OTP",
  "2FA code from email", "device verification", "one-time password",
  "email code", "extract OTP".
---

# Email OTP Plugin

## Keywords

email, otp, verification code, 2fa, one-time password, imap, gmail, icloud, outlook, bitwarden, github, google, aws, device verification

## Overview

Use `email-otp` to fetch one-time passwords (OTPs) from email inboxes via IMAP. The plugin supports multiple email providers and service-specific extraction templates.

**Supported providers:**

| Provider | IMAP Server | Auth Method |
|----------|-------------|-------------|
| Gmail | `imap.gmail.com:993` | App password |
| iCloud | `imap.mail.me.com:993` | App-specific password |
| Outlook | `outlook.office365.com:993` | App password |

**Supported templates:**

| Template | Service | Sender Pattern |
|----------|---------|----------------|
| bitwarden | Bitwarden | `no-reply@bitwarden.com` |
| github | GitHub | `noreply@github.com` |
| google | Google | `no-reply@accounts.google.com` |
| aws | AWS | `no-reply@signin.aws` |
| generic | Any OTP email | Auto-detect |

**Prerequisites:**
- Python 3.10+ (stdlib only — no external dependencies)
- Email app password (not your regular password)
  - Gmail: https://myaccount.google.com/apppasswords
  - iCloud: Apple ID → Sign-In and Security → App-Specific Passwords
  - Outlook: https://account.microsoft.com/security → App passwords

## Quick Reference

| Task | Command |
|------|---------|
| Fetch Bitwarden OTP from Gmail | `python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx --template bitwarden` |
| Fetch GitHub OTP from iCloud | `python3 -m email_otp --provider icloud --email user@icloud.com --app-password xxxx --template github` |
| Auto-detect service | `python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx` |
| Filter by sender | `python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx --sender no-reply@bitwarden.com` |
| JSON output | `python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx --json` |
| List providers | `python3 -m email_otp --list-providers` |
| List templates | `python3 -m email_otp --list-templates` |

## Workflow

### 1. Generate App Password

**Gmail:**
1. Go to https://myaccount.google.com/apppasswords
2. Select app → "Mail" → Generate
3. Copy the 16-character password

**iCloud:**
1. Go to https://appleid.apple.com
2. Sign-In and Security → App-Specific Passwords
3. Generate a new password

**Outlook:**
1. Go to https://account.microsoft.com/security
2. Advanced security options → App passwords
3. Create a new app password

### 2. Fetch OTP

```bash
# Bitwarden device verification code
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx" --template bitwarden

# GitHub 2FA code
python3 -m email_otp --provider icloud --email user@icloud.com --app-password "xxxx-xxxx-xxxx-xxxx" --template github

# Auto-detect (scans recent emails for any OTP)
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx"
```

### 3. Credential Resolution

Credentials are resolved in this order:
1. **CLI arguments** (`--email`, `--app-password`)
2. **Environment variables** (`EMAIL_USERNAME`, `EMAIL_PASSWORD`, `EMAIL_PROVIDER`)
3. **`.env` file** (`~/.config/email-otp/.env`)
4. **macOS Keychain** (service: `email-otp.<provider>.password`)

## Examples

**User:** "Get my Bitwarden verification code from Gmail"
```bash
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "$GMAIL_APP_PASSWORD" --template bitwarden
```

**User:** "Fetch the latest OTP from my iCloud inbox"
```bash
python3 -m email_otp --provider icloud --email user@icloud.com --app-password "$ICLOUD_APP_PASSWORD"
```

**User:** "Get GitHub 2FA code"
```bash
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "$GMAIL_APP_PASSWORD" --template github --sender noreply@github.com
```

## Guidelines

- **App passwords only.** Never use your regular email password. Generate an app-specific password from your email provider's security settings.
- **Template selection.** Use `--template` when you know which service sent the OTP. Use auto-detect (omit `--template`) when unsure.
- **Sender filtering.** Use `--sender` to narrow search when auto-detect returns wrong results.
- **Unseen only.** Use `--unseen` to only check unread emails, avoiding re-processing old OTPs.
- **JSON output.** Use `--json` when parsing output programmatically.
- **Credential security.** Store app passwords in macOS Keychain or `.env` file, not in shell history.
- **Rate limiting.** Don't call this in tight loops — IMAP connections have overhead.
- **Mark seen.** By default, processed emails are marked as read. Use `--no-mark-seen` to keep them unread.
