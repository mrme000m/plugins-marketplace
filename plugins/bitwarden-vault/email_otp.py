#!/usr/bin/env python3
"""
email_otp.py — Fetch the latest Bitwarden device verification code from any IMAP provider.

Usage:
  python3 email_otp.py --provider gmail --email <email> --app-password <password>
  python3 email_otp.py --server imap.mail.me.com --email <email> --app-password <password>

Supported providers:
  gmail    → imap.gmail.com:993
  icloud   → imap.mail.me.com:993
  outlook  → outlook.office365.com:993
  yahoo    → imap.mail.yahoo.com:993
  custom   → requires --server

Returns:
  The 6-digit verification code (or empty string if not found).
"""

import argparse
import imaplib
import re
import ssl
import sys
from email.parser import BytesParser
from email.policy import default

PROVIDERS = {
    "gmail": ("imap.gmail.com", 993),
    "icloud": ("imap.mail.me.com", 993),
    "outlook": ("outlook.office365.com", 993),
    "yahoo": ("imap.mail.yahoo.com", 993),
}


def fetch_otp(
    server, port, email, app_password, sender="do-not-reply@bitwarden.com", timeout=30
):
    context = ssl.create_default_context()
    with imaplib.IMAP4_SSL(server, port, ssl_context=context, timeout=timeout) as mail:
        mail.login(email, app_password)
        mail.select("inbox")

        # Search for emails from Bitwarden
        _, search_data = mail.search(None, f'(FROM "{sender}")')
        msg_ids = search_data[0].split()

        if not msg_ids:
            # Fallback: search broader
            _, search_data = mail.search(None, '(FROM "bitwarden")')
            msg_ids = search_data[0].split()

        if not msg_ids:
            return ""

        # Check most recent 3 emails
        for msg_id in reversed(msg_ids[-3:]):
            _, msg_data = mail.fetch(msg_id, "(RFC822)")
            for response_part in msg_data:
                if isinstance(response_part, tuple):
                    msg = BytesParser(policy=default).parsebytes(response_part[1])
                    body = extract_body(msg)

                    patterns = [
                        r"verification code is[:\s]+(\d{6})",
                        r"Verification Code[:\s]+(\d{6})",
                        r"code[:\s]+(\d{6})",
                        r"(\d{6})\s+is your Bitwarden",
                        r"(\d{6})\s+is your verification",
                        r"your code is[:\s]+(\d{6})",
                    ]

                    for pattern in patterns:
                        match = re.search(pattern, body, re.IGNORECASE)
                        if match:
                            return match.group(1)

        return ""


def extract_body(msg):
    body = ""
    if msg.is_multipart():
        for part in msg.walk():
            content_type = part.get_content_type()
            if content_type in ("text/plain", "text/html"):
                try:
                    body += part.get_payload(decode=True).decode(
                        "utf-8", errors="ignore"
                    )
                except Exception:
                    pass
    else:
        try:
            body = msg.get_payload(decode=True).decode("utf-8", errors="ignore")
        except Exception:
            pass
    return body


def main():
    parser = argparse.ArgumentParser(
        description="Fetch Bitwarden OTP from email via IMAP"
    )
    parser.add_argument(
        "--provider",
        choices=list(PROVIDERS.keys()) + ["custom"],
        help="Email provider preset",
    )
    parser.add_argument("--server", help="IMAP server hostname (for custom provider)")
    parser.add_argument(
        "--port", type=int, default=993, help="IMAP port (default: 993)"
    )
    parser.add_argument("--email", required=True, help="Email address")
    parser.add_argument("--app-password", required=True, help="App-specific password")
    parser.add_argument(
        "--sender", default="do-not-reply@bitwarden.com", help="Expected sender email"
    )
    args = parser.parse_args()

    if args.provider and args.provider != "custom":
        server, port = PROVIDERS[args.provider]
    else:
        if not args.server:
            print("Error: --server required for custom provider", file=sys.stderr)
            sys.exit(1)
        server = args.server
        port = args.port

    try:
        code = fetch_otp(server, port, args.email, args.app_password, args.sender)
        if code:
            print(code)
            sys.exit(0)
        else:
            print("", file=sys.stderr)
            sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
