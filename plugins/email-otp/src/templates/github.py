"""GitHub OTP extraction template."""

from .base import OTPTemplate


class GitHubTemplate(OTPTemplate):
    """Template for extracting GitHub 2FA and verification codes."""

    NAME = "github"
    SENDERS = [
        "noreply@github.com",
        "github.com",
    ]
    SUBJECT_PATTERNS = [
        r"verification code",
        r"one-time password",
        r"2fa",
        r"two-factor",
        r"github",
    ]
    BODY_PATTERNS = [
        r"verification code is[:\s]+(\d{6})",
        r"code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your GitHub",
        r"(\d{6})\s+is your verification",
        r"one-time password[:\s]+(\d{6})",
        r"OTP[:\s]+(\d{6})",
        r"(\d{6})",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"
