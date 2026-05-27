"""Bitwarden OTP extraction template."""

from .base import OTPTemplate


class BitwardenTemplate(OTPTemplate):
    """Template for extracting Bitwarden device verification codes."""

    NAME = "bitwarden"
    SENDERS = [
        "no-reply@bitwarden.com",
        "do-not-reply@bitwarden.com",
        "bitwarden.com",
    ]
    SUBJECT_PATTERNS = [
        r"verify your email",
        r"verification code",
        r"new device",
        r"bitwarden",
    ]
    BODY_PATTERNS = [
        r"verification code is[:\s]+(\d{6})",
        r"Verification Code[:\s]+(\d{6})",
        r"code[:\s]+(\d{6})",
        r"(\d{6})\s+is your Bitwarden",
        r"(\d{6})\s+is your verification",
        r"your code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your",
        r"enter this verification code[:\s]+(\d{6})",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"
