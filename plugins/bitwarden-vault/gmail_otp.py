#!/usr/bin/env python3
"""
gmail_otp.py — Fetch the latest Bitwarden device verification code from Gmail via IMAP.

Usage:
  python3 gmail_otp.py <email> <app_password>

Returns:
  The 6-digit verification code (or empty string if not found).

Requires:
  - Gmail account with 2FA enabled
  - App-specific password generated at https://myaccount.google.com/apppasswords
  - Bitwarden "New Device Verification" email in inbox
"""

import imaplib
import re
import ssl
import sys
from email.parser import BytesParser
from email.policy import default


def fetch_bw_otp(email: str, app_password: str, timeout: int = 30) -> str:
    """Connect to Gmail IMAP and extract the latest Bitwarden verification code."""

    context = ssl.create_default_context()
    with imaplib.IMAP4_SSL(
        "imap.gmail.com", ssl_context=context, timeout=timeout
    ) as mail:
        mail.login(email, app_password)
        mail.select("inbox")

        # Search for unread emails from Bitwarden (most recent first)
        # Bitwarden verification emails come from: do-not-reply@bitwarden.com
        _, search_data = mail.search(None, '(FROM "do-not-reply@bitwarden.com")')

        msg_ids = search_data[0].split()
        if not msg_ids:
            # Fallback: search all emails from bitwarden (not just unread)
            _, search_data = mail.search(None, '(FROM "bitwarden.com")')
            msg_ids = search_data[0].split()

        if not msg_ids:
            return ""

        # Check the most recent 3 emails
        for msg_id in reversed(msg_ids[-3:]):
            _, msg_data = mail.fetch(msg_id, "(RFC822)")
            for response_part in msg_data:
                if isinstance(response_part, tuple):
                    msg = BytesParser(policy=default).parsebytes(response_part[1])
                    body = ""

                    if msg.is_multipart():
                        for part in msg.walk():
                            content_type = part.get_content_type()
                            if (
                                content_type == "text/plain"
                                or content_type == "text/html"
                            ):
                                try:
                                    body += part.get_payload(decode=True).decode(
                                        "utf-8", errors="ignore"
                                    )
                                except Exception:
                                    pass
                    else:
                        try:
                            body = msg.get_payload(decode=True).decode(
                                "utf-8", errors="ignore"
                            )
                        except Exception:
                            pass

                    # Look for verification code patterns
                    # Bitwarden uses: "Your verification code is: 123456"
                    # Or: "Verification Code: 123456"
                    # Or: "code: 123456"
                    patterns = [
                        r"verification code is[:\s]+(\d{6})",
                        r"Verification Code[:\s]+(\d{6})",
                        r"code[:\s]+(\d{6})",
                        r"(\d{6})\s+is your Bitwarden",
                        r"(\d{6})\s+is your verification",
                    ]

                    for pattern in patterns:
                        match = re.search(pattern, body, re.IGNORECASE)
                        if match:
                            return match.group(1)

        return ""


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: gmail_otp.py <email> <app_password>", file=sys.stderr)
        sys.exit(1)

    email = sys.argv[1]
    app_password = sys.argv[2]

    try:
        code = fetch_bw_otp(email, app_password)
        if code:
            print(code)
            sys.exit(0)
        else:
            print("", file=sys.stderr)
            sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
