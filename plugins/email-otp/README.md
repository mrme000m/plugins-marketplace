# email-otp

Modular email OTP extraction plugin. Fetches one-time passwords from email inboxes via IMAP with pluggable providers and service-specific templates.

## Features

- **Pluggable email providers** — Gmail, iCloud, Outlook (easily extensible)
- **Service-specific templates** — Bitwarden, GitHub, Google, AWS, generic fallback
- **Auto-detection** — Automatically identifies the service from email sender/subject
- **Zero dependencies** — Uses only Python standard library
- **Credential resolution** — CLI args → env vars → `.env` file → macOS Keychain
- **JSON output** — Machine-readable output for programmatic use

## Quick Start

```bash
# Fetch Bitwarden OTP from Gmail
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx" --template bitwarden

# Fetch GitHub 2FA code from iCloud
python3 -m email_otp --provider icloud --email user@icloud.com --app-password "xxxx-xxxx-xxxx-xxxx" --template github

# Auto-detect any OTP
python3 -m email_otp --provider gmail --email user@gmail.com --app-password "xxxx xxxx xxxx xxxx"
```

## Architecture

```
src/
├── cli.py              # CLI entry point
├── core.py             # Core OTP fetcher (IMAP + extraction)
├── providers/
│   ├── base.py         # Abstract provider interface
│   ├── gmail.py        # Gmail IMAP provider
│   ├── icloud.py       # iCloud IMAP provider
│   └── outlook.py      # Outlook IMAP provider
└── templates/
    ├── base.py         # Abstract template interface
    ├── bitwarden.py    # Bitwarden OTP patterns
    ├── github.py       # GitHub OTP patterns
    ├── google.py       # Google OTP patterns
    ├── aws.py          # AWS OTP patterns
    └── generic.py      # Generic fallback patterns
```

## Adding a New Provider

Create `src/providers/myservice.py`:

```python
from .base import EmailProvider

class MyServiceProvider(EmailProvider):
    IMAP_SERVER = "imap.myservice.com"
    IMAP_PORT = 993
    NAME = "myservice"
```

Register in `src/providers/__init__.py`:

```python
from .myservice import MyServiceProvider

PROVIDERS["myservice"] = MyServiceProvider
```

## Adding a New Template

Create `src/templates/myservice.py`:

```python
from .base import OTPTemplate

class MyServiceTemplate(OTPTemplate):
    NAME = "myservice"
    SENDERS = ["noreply@myservice.com"]
    SUBJECT_PATTERNS = [r"verification code", r"one-time password"]
    BODY_PATTERNS = [
        r"code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"
```

Register in `src/templates/__init__.py`:

```python
from .myservice import MyServiceTemplate

TEMPLATES["myservice"] = MyServiceTemplate
```

## App Password Setup

- **Gmail:** https://myaccount.google.com/apppasswords
- **iCloud:** Apple ID → Sign-In and Security → App-Specific Passwords
- **Outlook:** https://account.microsoft.com/security → App passwords

## Requirements

- Python 3.10+
- No external dependencies (stdlib only)
