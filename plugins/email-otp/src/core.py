"""
Core OTP fetcher — connects to email via IMAP, searches for OTP emails,
and extracts codes using service-specific templates.
"""

import re
from email.parser import BytesParser
from email.policy import default
from typing import Optional

from .providers import get_provider, EmailProvider
from .templates import get_template, auto_detect_template, OTPTemplate


class OTPFetcher:
    """Fetches OTP codes from email inboxes via IMAP."""

    def __init__(
        self,
        provider_name: str,
        email: str,
        app_password: str,
        imap_server: str = "",
        imap_port: int = 993,
    ):
        self.provider: EmailProvider = get_provider(
            provider_name, email=email, app_password=app_password
        )
        if imap_server:
            self.provider.IMAP_SERVER = imap_server
        if imap_port != 993:
            self.provider.IMAP_PORT = imap_port

    def fetch(
        self,
        template_name: str = "",
        sender: str = "",
        subject: str = "",
        unseen_only: bool = False,
        max_emails: int = 10,
        mark_seen: bool = True,
    ) -> Optional[str]:
        """
        Fetch the most recent OTP code from the inbox.

        Args:
            template_name: Specific template to use (e.g., 'bitwarden', 'github').
                          If empty, auto-detects based on email content.
            sender: Filter by sender email address.
            subject: Filter by subject keyword.
            unseen_only: Only search unread emails.
            max_emails: Maximum number of recent emails to check.
            mark_seen: Mark processed emails as read.

        Returns:
            The OTP code string, or None if not found.
        """
        template: Optional[OTPTemplate] = None
        if template_name:
            template = get_template(template_name)
            if not sender and template.SENDERS:
                sender = template.SENDERS[0]

        with self.provider:
            conn = self.provider.connect()
            conn.select("INBOX")

            criteria = self.provider.search_criteria(sender, subject, unseen_only)
            _, search_data = conn.search(None, criteria)
            msg_ids = search_data[0].split()

            if not msg_ids:
                if sender:
                    _, search_data = conn.search(None, "ALL")
                    msg_ids = search_data[0].split()
                if not msg_ids:
                    return None

            for msg_id in reversed(msg_ids[-max_emails:]):
                _, msg_data = conn.fetch(msg_id, "(RFC822)")
                for response_part in msg_data:
                    if not isinstance(response_part, tuple):
                        continue

                    msg = BytesParser(policy=default).parsebytes(response_part[1])
                    from_addr = str(msg.get("From", ""))
                    subject_str = str(msg.get("Subject", ""))
                    body = self._extract_body(msg)

                    current_template = template
                    if not current_template:
                        current_template = auto_detect_template(from_addr, subject_str)

                    code = current_template.extract(body)
                    if code:
                        if mark_seen:
                            self.provider.mark_seen(msg_id)
                        return code

        return None

    def _extract_body(self, msg) -> str:
        """Extract plain text body from email message."""
        body = ""
        if msg.is_multipart():
            for part in msg.walk():
                content_type = part.get_content_type()
                if content_type in ("text/plain", "text/html"):
                    try:
                        payload = part.get_payload(decode=True)
                        if payload:
                            body += payload.decode("utf-8", errors="ignore")
                    except Exception:
                        pass
        else:
            try:
                payload = msg.get_payload(decode=True)
                if payload:
                    body = payload.decode("utf-8", errors="ignore")
            except Exception:
                pass
        return body
