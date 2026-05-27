"""AWS OTP extraction template."""

from .base import OTPTemplate


class AWSTemplate(OTPTemplate):
    """Template for extracting AWS verification codes."""

    NAME = "aws"
    SENDERS = [
        "no-reply@signin.aws",
        "amazon.com",
        "aws.amazon.com",
    ]
    SUBJECT_PATTERNS = [
        r"verification code",
        r"one-time password",
        r"aws",
        r"amazon",
    ]
    BODY_PATTERNS = [
        r"verification code is[:\s]+(\d{6})",
        r"code is[:\s]+(\d{6})",
        r"(\d{6})\s+is your AWS",
        r"(\d{6})\s+is your verification",
        r"one-time password[:\s]+(\d{6})",
        r"(\d{6})",
    ]
    CODE_LENGTH = 6
    CODE_TYPE = "digits"
