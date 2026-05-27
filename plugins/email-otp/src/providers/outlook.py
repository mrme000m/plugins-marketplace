"""Outlook/Hotmail IMAP provider."""

from .base import EmailProvider


class OutlookProvider(EmailProvider):
    """Outlook/Hotmail IMAP provider using app passwords."""

    IMAP_SERVER = "outlook.office365.com"
    IMAP_PORT = 993
    NAME = "outlook"
