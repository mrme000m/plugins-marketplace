#!/usr/bin/env python3
"""
email-otp — Fetch OTP codes from email inboxes via IMAP.

Usage:
  python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx --template bitwarden
  python3 -m email_otp --provider icloud --email user@icloud.com --app-password xxxx --sender no-reply@bitwarden.com
  python3 -m email_otp --provider outlook --email user@outlook.com --app-password xxxx --template github

Environment variables (fallback):
  EMAIL_USERNAME       Email address
  EMAIL_PASSWORD       App-specific password
  EMAIL_PROVIDER       Provider name (gmail, icloud, outlook)
  EMAIL_IMAP_SERVER    Custom IMAP server (overrides provider default)
  EMAIL_IMAP_PORT      Custom IMAP port (default: 993)

macOS Keychain (fallback):
  Service: email-otp.<provider>.password
  Account: <email>
"""

import argparse
import json
import os
import subprocess
import sys

from .core import OTPFetcher
from .providers import list_providers
from .templates import list_templates


def get_keychain_password(provider: str, email: str) -> str:
    """Try to load app password from macOS Keychain."""
    try:
        result = subprocess.run(
            [
                "security",
                "find-generic-password",
                "-a", email,
                "-s", f"email-otp.{provider}.password",
                "-w",
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode == 0 and result.stdout.strip():
            return result.stdout.strip()
    except Exception:
        pass
    return ""


def load_dotenv() -> dict:
    """Load credentials from ~/.config/email-otp/.env file."""
    env_file = os.path.expanduser("~/.config/email-otp/.env")
    if not os.path.exists(env_file):
        return {}

    creds = {}
    with open(env_file) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            if "=" in line:
                key, value = line.split("=", 1)
                creds[key.strip()] = value.strip().strip("\"'")
    return creds


def resolve_credentials(args) -> tuple:
    """Resolve email credentials from args, env, .env, or keychain."""
    email = args.email or os.environ.get("EMAIL_USERNAME", "")
    password = args.app_password or os.environ.get("EMAIL_PASSWORD", "")
    provider = args.provider or os.environ.get("EMAIL_PROVIDER", "gmail")

    if not email or not password:
        dotenv = load_dotenv()
        email = email or dotenv.get("EMAIL_USERNAME", "")
        password = password or dotenv.get("EMAIL_PASSWORD", "")
        provider = provider or dotenv.get("EMAIL_PROVIDER", "gmail")

    if email and not password:
        password = get_keychain_password(provider, email)

    return email, password, provider


def main():
    parser = argparse.ArgumentParser(
        description="Fetch OTP codes from email inboxes via IMAP",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  Fetch Bitwarden OTP from Gmail:
    python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx --template bitwarden

  Fetch GitHub OTP from iCloud:
    python3 -m email_otp --provider icloud --email user@icloud.com --app-password xxxx --template github

  Auto-detect service from email content:
    python3 -m email_otp --provider gmail --email user@gmail.com --app-password xxxx

  List available providers and templates:
    python3 -m email_otp --list-providers
    python3 -m email_otp --list-templates
        """,
    )

    parser.add_argument(
        "--provider",
        choices=list_providers() + ["custom"],
        default="gmail",
        help="Email provider (default: gmail)",
    )
    parser.add_argument("--email", help="Email address")
    parser.add_argument("--app-password", help="App-specific password")
    parser.add_argument(
        "--server",
        help="Custom IMAP server (for --provider custom)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=993,
        help="IMAP port (default: 993)",
    )
    parser.add_argument(
        "--template",
        choices=list_templates(),
        default="",
        help="OTP extraction template (default: auto-detect)",
    )
    parser.add_argument("--sender", help="Filter by sender email address")
    parser.add_argument("--subject", help="Filter by subject keyword")
    parser.add_argument(
        "--unseen",
        action="store_true",
        help="Only search unread emails",
    )
    parser.add_argument(
        "--max-emails",
        type=int,
        default=10,
        help="Maximum emails to check (default: 10)",
    )
    parser.add_argument(
        "--no-mark-seen",
        action="store_true",
        help="Don't mark processed emails as read",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output JSON instead of plain text",
    )
    parser.add_argument(
        "--list-providers",
        action="store_true",
        help="List available email providers",
    )
    parser.add_argument(
        "--list-templates",
        action="store_true",
        help="List available OTP templates",
    )

    args = parser.parse_args()

    if args.list_providers:
        for name in list_providers():
            print(f"  {name}")
        return

    if args.list_templates:
        for name in list_templates():
            print(f"  {name}")
        return

    email, password, provider = resolve_credentials(args)

    if not email:
        print("Error: --email required (or set EMAIL_USERNAME env var)", file=sys.stderr)
        sys.exit(1)
    if not password:
        print("Error: --app-password required (or set EMAIL_PASSWORD env var)", file=sys.stderr)
        sys.exit(1)

    if args.provider == "custom" and not args.server:
        print("Error: --server required for custom provider", file=sys.stderr)
        sys.exit(1)

    try:
        fetcher = OTPFetcher(
            provider_name=provider,
            email=email,
            app_password=password,
            imap_server=args.server or "",
            imap_port=args.port,
        )

        code = fetcher.fetch(
            template_name=args.template,
            sender=args.sender,
            subject=args.subject,
            unseen_only=args.unseen,
            max_emails=args.max_emails,
            mark_seen=not args.no_mark_seen,
        )

        if code:
            if args.json:
                print(json.dumps({"code": code, "provider": provider, "template": args.template or "auto"}))
            else:
                print(code)
            sys.exit(0)
        else:
            if args.json:
                print(json.dumps({"code": None, "error": "No OTP found"}))
            else:
                print("No OTP found", file=sys.stderr)
            sys.exit(1)

    except Exception as e:
        if args.json:
            print(json.dumps({"code": None, "error": str(e)}))
        else:
            print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
