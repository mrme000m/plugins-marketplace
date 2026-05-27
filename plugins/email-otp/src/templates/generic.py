"""Generic OTP extraction template (fallback)."""

from .base import OTPTemplate


class GenericTemplate(OTPTemplate):
    """Generic fallback template for extracting OTP codes from any email."""

    NAME = "generic"
    SENDERS = []
    SUBJECT_PATTERNS = [
        r"verification",
        r"code",
        r"otp",
        r"one-time",
        r"2fa",
        r"two-factor",
    ]
    BODY_PATTERNS = [
        r"verification code is[:\s]+(\d{6})",
        r"code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your",
        r"one-time password[:\s]+(\d{6})",
        r"OTP[:\s]+(\d{6})",
        r"code[:\s]+(\d{6})",
        r"(\d{6})",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"

    def matches(self, sender: str, subject: str) -> bool:
        """Generic template matches any email with OTP-related keywords."""
        return super().matches(sender, subject)
