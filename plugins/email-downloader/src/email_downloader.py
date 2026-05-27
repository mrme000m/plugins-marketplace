#!/usr/bin/env python3
"""
Modular Email Downloader for Agentic Usage.
Supports IMAP with username/password authentication.
Integrates with the email-attachments plugin for parsing.
"""

import argparse
import email
import imaplib
import json
import os
import re
import ssl
import subprocess
import sys
from datetime import datetime
from email.header import decode_header
from pathlib import Path


def extract_domain(from_addr):
    """Extract the sender domain from a From header string."""
    match = re.search(r"@([a-zA-Z0-9._-]+)", from_addr)
    if match:
        return match.group(1).lower()
    return "unknown"


def safe_filename(text, max_len=50):
    """Create a filesystem-safe filename from a string."""
    safe = "".join(c for c in text[:max_len] if c.isalnum() or c in " _-").strip()
    return safe or "untitled"


class EmailDownloader:
    def __init__(self, username, password, imap_server, port=993):
        self.username = username
        self.password = password
        self.imap_server = imap_server
        self.port = port
        self.connection = None

    def connect(self):
        """Establish connection to the IMAP server."""
        try:
            context = ssl.create_default_context()
            self.connection = imaplib.IMAP4_SSL(
                self.imap_server, self.port, ssl_context=context
            )
            self.connection.login(self.username, self.password)
            return True
        except Exception as e:
            # Try relaxed SSL as fallback
            try:
                context = ssl.create_default_context()
                context.check_hostname = False
                context.verify_mode = ssl.CERT_NONE
                self.connection = imaplib.IMAP4_SSL(
                    self.imap_server, self.port, ssl_context=context
                )
                self.connection.login(self.username, self.password)
                return True
            except Exception:
                raise e

    def list_mailboxes(self):
        """List all available mailboxes/folders."""
        if not self.connection:
            return []
        status, mailboxes = self.connection.list()
        folders = []
        if status == "OK":
            for mb in mailboxes:
                decoded = mb.decode()
                # Handle different formats of IMAP LIST response
                match = re.search(r'"([^"]+)"$', decoded)
                if match:
                    folders.append(match.group(1))
                else:
                    folders.append(decoded.split(" ")[-1].strip('"'))
        return folders

    def select_mailbox(self, mailbox="INBOX"):
        """Select a mailbox to read from."""
        status, data = self.connection.select(mailbox)
        if status == "OK":
            return int(data[0])
        return 0

    def search_emails(
        self,
        criteria="ALL",
        sender=None,
        subject=None,
        since=None,
        before=None,
        unseen_only=False,
    ):
        search_criteria = []
        if unseen_only:
            search_criteria.append("UNSEEN")
        if sender:
            search_criteria.append(f'FROM "{sender}"')
        if subject:
            search_criteria.append(f'SUBJECT "{subject}"')
        if since:
            search_criteria.append(f"SINCE {since}")
        if before:
            search_criteria.append(f"BEFORE {before}")

        if not search_criteria:
            search_criteria.append(criteria)

        search_string = " ".join(search_criteria)
        status, messages = self.connection.search(None, search_string)
        if status == "OK":
            return messages[0].split()
        return []

    def decode_mime_header(self, header_value):
        """Decode a MIME-encoded email header."""
        if header_value is None:
            return "N/A"
        decoded_parts = decode_header(header_value)
        result = []
        for part, charset in decoded_parts:
            if isinstance(part, bytes):
                charset = charset or "utf-8"
                try:
                    result.append(part.decode(charset, errors="replace"))
                except (LookupError, UnicodeDecodeError):
                    result.append(part.decode("utf-8", errors="replace"))
            else:
                result.append(part)
        return "".join(result)

    def get_email_body(self, msg):
        """Extract the body text from an email message."""
        body = ""
        html_body = ""
        if msg.is_multipart():
            for part in msg.walk():
                content_type = part.get_content_type()
                content_disposition = str(part.get("Content-Disposition", ""))
                if "attachment" in content_disposition:
                    continue
                try:
                    payload = part.get_payload(decode=True)
                    if payload is None:
                        continue
                    charset = part.get_content_charset() or "utf-8"
                    text = payload.decode(charset, errors="replace")
                    if content_type == "text/plain":
                        body += text
                    elif content_type == "text/html":
                        html_body += text
                except Exception:
                    continue
        else:
            try:
                payload = msg.get_payload(decode=True)
                if payload:
                    charset = msg.get_content_charset() or "utf-8"
                    body = payload.decode(charset, errors="replace")
            except Exception:
                body = "Could not decode email body."
        return body if body else html_body

    def download_attachments(self, msg, save_dir):
        """Download attachments from an email message. Returns list of saved paths."""
        attachments = []
        if msg.is_multipart():
            for part in msg.walk():
                content_disposition = str(part.get("Content-Disposition", ""))
                if "attachment" in content_disposition:
                    filename = part.get_filename()
                    if filename:
                        filename = self.decode_mime_header(filename)
                        filename = "".join(
                            c for c in filename if c.isalnum() or c in "._- "
                        )
                        filepath = os.path.join(save_dir, filename)
                        counter = 1
                        base, ext = os.path.splitext(filepath)
                        while os.path.exists(filepath):
                            filepath = f"{base}_{counter}{ext}"
                            counter += 1
                        with open(filepath, "wb") as f:
                            f.write(part.get_payload(decode=True))
                        attachments.append(filepath)
        return attachments

    def fetch_emails(
        self,
        email_ids,
        save_dir="downloads",
        limit=None,
        mark_as_read=False,
        parse_attachments=False,
    ):
        raw_dir = os.path.join(save_dir, "raw")
        md_dir = os.path.join(save_dir, "md")
        os.makedirs(raw_dir, exist_ok=True)
        os.makedirs(md_dir, exist_ok=True)

        emails_data = []
        if limit:
            email_ids = email_ids[-limit:]

        for email_id in email_ids:
            try:
                fetch_flag = "(RFC822)" if mark_as_read else "(BODY.PEEK[])"
                status, msg_data = self.connection.fetch(email_id, fetch_flag)
                if status != "OK":
                    continue

                raw_email = msg_data[0][1]
                msg = email.message_from_bytes(raw_email)

                subject = self.decode_mime_header(msg["Subject"])
                from_addr = self.decode_mime_header(msg["From"])
                to_addr = self.decode_mime_header(msg["To"])
                date_str = msg["Date"]
                message_id = msg["Message-ID"] or f"unknown_{email_id.decode()}"
                domain = extract_domain(from_addr)

                body = self.get_email_body(msg)
                safe_subj = safe_filename(subject)
                base_name = f"{email_id.decode()}_{safe_subj}"

                domain_raw_dir = os.path.join(raw_dir, domain)
                domain_md_dir = os.path.join(md_dir, domain)
                attach_dir = os.path.join(domain_md_dir, "attachments", base_name)
                os.makedirs(domain_raw_dir, exist_ok=True)
                os.makedirs(domain_md_dir, exist_ok=True)
                os.makedirs(attach_dir, exist_ok=True)

                eml_path = os.path.join(domain_raw_dir, f"{base_name}.eml")
                with open(eml_path, "wb") as f:
                    f.write(raw_email)

                attachments = self.download_attachments(msg, attach_dir)
                if not attachments and os.path.exists(attach_dir):
                    os.rmdir(attach_dir)

                md_path = os.path.join(domain_md_dir, f"{base_name}.md")
                with open(md_path, "w", encoding="utf-8") as f:
                    f.write(f"# {subject}\n\n")
                    f.write(f"- **From:** {from_addr}\n")
                    f.write(f"- **To:** {to_addr}\n")
                    f.write(f"- **Date:** {date_str}\n")
                    f.write(f"- **Message-ID:** {message_id}\n")
                    f.write(f"- **Attachments:** {len(attachments)}\n")
                    if attachments:
                        f.write("\n## Attachments\n\n")
                        for att in attachments:
                            rel = os.path.relpath(att, domain_md_dir)
                            f.write(f"- `{os.path.basename(att)}` (`{rel}`)\n")
                    f.write("\n---\n\n")
                    f.write(body)

                email_info = {
                    "id": email_id.decode(),
                    "subject": subject,
                    "from": from_addr,
                    "to": to_addr,
                    "date": date_str,
                    "message_id": message_id,
                    "domain": domain,
                    "eml_path": eml_path,
                    "md_path": md_path,
                    "attachments": attachments,
                }

                if parse_attachments and attachments:
                    self._trigger_parsing(attachments, from_addr, message_id)

                emails_data.append(email_info)

            except Exception as e:
                print(f"Error processing email {email_id}: {e}", file=sys.stderr)
                continue

        return emails_data

    def _trigger_parsing(self, attachments, sender, msgid):
        # Find parse_attachment.sh
        # Assuming it's in plugins/email-attachments/parsers/parse_attachment.sh
        # relative to the workspace root.
        parser_script = Path("plugins/email-attachments/parsers/parse_attachment.sh")
        if not parser_script.exists():
            # Try to find it relative to this script if we are in plugins/email-downloader/src
            parser_script = (
                Path(__file__).parents[3]
                / "email-attachments"
                / "parsers"
                / "parse_attachment.sh"
            )

        if parser_script.exists():
            for att in attachments:
                try:
                    subprocess.run(
                        [
                            "bash",
                            str(parser_script),
                            att,
                            "--sender",
                            sender,
                            "--msgid",
                            msgid,
                        ],
                        check=False,
                    )
                except Exception as e:
                    print(f"Failed to parse attachment {att}: {e}", file=sys.stderr)

    def disconnect(self):
        """Close the connection to the IMAP server."""
        if self.connection:
            try:
                self.connection.close()
                self.connection.logout()
            except Exception:
                pass


def main():
    parser = argparse.ArgumentParser(description="Email Downloader Plugin")
    parser.add_argument("--username", help="IMAP username")
    parser.add_argument("--password", help="IMAP password")
    parser.add_argument("--server", help="IMAP server address")
    parser.add_argument("--port", type=int, default=993, help="IMAP port (default 993)")
    parser.add_argument(
        "--mailbox", default="INBOX", help="Mailbox to search (default INBOX)"
    )
    parser.add_argument("--output", default="attachments", help="Output directory")
    parser.add_argument("--limit", type=int, help="Limit number of emails")
    parser.add_argument("--unread", action="store_true", help="Only unread emails")
    parser.add_argument("--sender", help="Filter by sender")
    parser.add_argument("--subject", help="Filter by subject keyword")
    parser.add_argument("--since", help="Filter since date (DD-Mon-YYYY)")
    parser.add_argument(
        "--parse",
        action="store_true",
        help="Automatically trigger attachment parsing if available",
    )
    parser.add_argument("--json", action="store_true", help="Output results as JSON")
    parser.add_argument(
        "--list-folders", action="store_true", help="List available mailboxes/folders"
    )
    parser.add_argument(
        "--list-emails",
        action="store_true",
        help="List email summaries without downloading bodies",
    )

    args = parser.parse_args()

    # Load from env if not provided
    username = args.username or os.environ.get("EMAIL_USERNAME")
    password = args.password or os.environ.get("EMAIL_PASSWORD")
    server = args.server or os.environ.get("EMAIL_IMAP_SERVER")

    if not all([username, password, server]):
        if args.json:
            print(
                json.dumps(
                    {"error": "Missing credentials (username, password, or server)"}
                )
            )
        else:
            print(
                "Error: Missing credentials. Provide via arguments or environment variables (EMAIL_USERNAME, EMAIL_PASSWORD, EMAIL_IMAP_SERVER)."
            )
        sys.exit(1)

    downloader = EmailDownloader(username, password, server, args.port)
    try:
        downloader.connect()

        if args.list_folders:
            folders = downloader.list_mailboxes()
            if args.json:
                print(json.dumps({"folders": folders}))
            else:
                print("Available Mailboxes:")
                for f in folders:
                    print(f"  📁 {f}")
            return

        downloader.select_mailbox(args.mailbox)

        email_ids = downloader.search_emails(
            unseen_only=args.unread,
            sender=args.sender,
            subject=args.subject,
            since=args.since,
        )

        if not email_ids:
            if args.json:
                print(json.dumps({"count": 0, "emails": []}))
            else:
                print(f"No emails found matching criteria in {args.mailbox}.")
            return

        if args.list_emails:
            if args.limit:
                email_ids = email_ids[-args.limit :]

            summaries = []
            for eid in email_ids:
                status, data = downloader.connection.fetch(
                    eid, "(BODY[HEADER.FIELDS (SUBJECT FROM DATE)])"
                )
                msg = email.message_from_bytes(data[0][1])
                summaries.append(
                    {
                        "id": eid.decode(),
                        "subject": downloader.decode_mime_header(msg["Subject"]),
                        "from": downloader.decode_mime_header(msg["From"]),
                        "date": msg["Date"],
                    }
                )

            if args.json:
                print(
                    json.dumps({"count": len(summaries), "emails": summaries}, indent=2)
                )
            else:
                print(f"Emails in {args.mailbox} ({len(summaries)}):")
                for s in summaries:
                    print(f"- [{s['id']}] {s['subject']} ({s['from']})")
            return

        emails = downloader.fetch_emails(
            email_ids,
            save_dir=args.output,
            limit=args.limit,
            parse_attachments=args.parse,
        )

        if args.json:
            print(json.dumps({"count": len(emails), "emails": emails}, indent=2))
        else:
            print(f"Successfully downloaded {len(emails)} email(s) to {args.output}.")
            for e in emails:
                print(f"- [{e['id']}] {e['subject']} (from: {e['from']})")
                if e["attachments"]:
                    print(
                        f"  Attachments: {', '.join([os.path.basename(a) for a in e['attachments']])}"
                    )

    except Exception as e:
        if args.json:
            print(json.dumps({"error": str(e)}))
        else:
            print(f"Error: {e}")
        sys.exit(1)
    finally:
        downloader.disconnect()


if __name__ == "__main__":
    main()
