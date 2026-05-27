"""Gmail IMAP provider."""

from .base import EmailProvider


class GmailProvider(EmailProvider):
    """Gmail IMAP provider using app passwords."""

    IMAP_SERVER = "imap.gmail.com"
    IMAP_PORT = 993
    NAME = "gmail"

    def search_criteria(self, sender: str = "", subject: str = "", unseen_only: bool = False) -> str:
        """Gmail supports X-GM-RAW for advanced search, but standard IMAP works fine."""
        return super().search_criteria(sender, subject, unseen_only)
