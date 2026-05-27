"""Google OTP extraction template."""

from .base import OTPTemplate


class GoogleTemplate(OTPTemplate):
    """Template for extracting Google verification codes."""

    NAME = "google"
    SENDERS = [
        "no-reply@accounts.google.com",
        "accounts.google.com",
        "google.com",
    ]
    SUBJECT_PATTERNS = [
        r"verification code",
        r"security alert",
        r"sign-in",
        r"google",
    ]
    BODY_PATTERNS = [
        r"verification code is[:\s]+(\d{6})",
        r"code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your Google",
        r"(\d{6})\s+is your verification",
        r"G-([\d]{6})",
        r"(\d{6})",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"
