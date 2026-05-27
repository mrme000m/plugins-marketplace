"""Base OTP extraction template."""

import re
from typing import Optional


class OTPTemplate:
    """Abstract base class for OTP extraction templates."""

    NAME: str = "base"
    SENDERS: list = []
    SUBJECT_PATTERNS: list = []
    BODY_PATTERNS: list = []
    CODE_LENGTH: int = 6
    CODE_TYPE: str = "digits"

    def matches(self, sender: str, subject: str) -> bool:
        """Check if this template matches the given email."""
        sender_lower = sender.lower()
        subject_lower = subject.lower()

        for pattern in self.SENDERS:
            if pattern.lower() in sender_lower:
                return True

        for pattern in self.SUBJECT_PATTERNS:
            if re.search(pattern, subject_lower, re.IGNORECASE):
                return True

        return False

    def extract(self, body: str) -> Optional[str]:
        """Extract OTP code from email body."""
        for pattern in self.BODY_PATTERNS:
            match = re.search(pattern, body, re.IGNORECASE | re.MULTILINE)
            if match:
                code = match.group(1)
                if self._validate_code(code):
                    return code
        return None

    def _validate_code(self, code: str) -> bool:
        """Validate the extracted code format."""
        if not code:
            return False
        if self.CODE_TYPE == "digits":
            return code.isdigit() and len(code) == self.CODE_LENGTH
        elif self.CODE_TYPE == "alphanumeric":
            return code.isalnum() and len(code) == self.CODE_LENGTH
        return len(code) == self.CODE_LENGTH
