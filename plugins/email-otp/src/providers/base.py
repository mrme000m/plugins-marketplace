"""Base email provider interface."""

import imaplib
import ssl
from typing import Optional


class EmailProvider:
    """Abstract base class for email providers."""

    IMAP_SERVER: str = ""
    IMAP_PORT: int = 993
    NAME: str = "base"

    def __init__(self, email: str, app_password: str):
        self.email = email
        self.app_password = app_password
        self._connection: Optional[imaplib.IMAP4_SSL] = None

    @property
    def imap_server(self) -> str:
        return self.IMAP_SERVER

    @property
    def imap_port(self) -> int:
        return self.IMAP_PORT

    def connect(self) -> imaplib.IMAP4_SSL:
        """Establish IMAP SSL connection and login."""
        if self._connection:
            try:
                self._connection.noop()
                return self._connection
            except Exception:
                self._connection = None

        context = ssl.create_default_context()
        try:
            conn = imaplib.IMAP4_SSL(
                self.imap_server, self.imap_port, ssl_context=context
            )
        except Exception:
            context = ssl.create_default_context()
            context.check_hostname = False
            context.verify_mode = ssl.CERT_NONE
            conn = imaplib.IMAP4_SSL(
                self.imap_server, self.imap_port, ssl_context=context
            )

        conn.login(self.email, self.app_password)
        self._connection = conn
        return conn

    def disconnect(self):
        """Close IMAP connection."""
        if self._connection:
            try:
                self._connection.close()
            except Exception:
                pass
            try:
                self._connection.logout()
            except Exception:
                pass
            self._connection = None

    def search_criteria(self, sender: str = "", subject: str = "", unseen_only: bool = False) -> str:
        """Build IMAP search criteria string."""
        parts = []
        if unseen_only:
            parts.append("UNSEEN")
        if sender:
            parts.append(f'FROM "{sender}"')
        if subject:
            parts.append(f'SUBJECT "{subject}"')
        if not parts:
            return "ALL"
        return " ".join(parts)

    def mark_seen(self, msg_id: str):
        """Mark a message as seen."""
        if self._connection:
            try:
                self._connection.store(msg_id, "+FLAGS", "\\Seen")
            except Exception:
                pass

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *args):
        self.disconnect()
