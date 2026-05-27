"""iCloud Mail IMAP provider."""

from .base import EmailProvider


class ICloudProvider(EmailProvider):
    """iCloud Mail IMAP provider using app-specific passwords."""

    IMAP_SERVER = "imap.mail.me.com"
    IMAP_PORT = 993
    NAME = "icloud"
