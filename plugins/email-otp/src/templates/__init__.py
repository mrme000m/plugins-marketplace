"""OTP extraction templates for different services."""

import re

from .base import OTPTemplate
from .bitwarden import BitwardenTemplate
from .github import GitHubTemplate
from .google import GoogleTemplate
from .aws import AWSTemplate
from .generic import GenericTemplate

TEMPLATES = {
    "bitwarden": BitwardenTemplate,
    "github": GitHubTemplate,
    "google": GoogleTemplate,
    "aws": AWSTemplate,
    "generic": GenericTemplate,
}


def get_template(name: str) -> OTPTemplate:
    """Get a template instance by name."""
    name = name.lower()
    if name not in TEMPLATES:
        raise ValueError(
            f"Unknown template: {name}. Available: {', '.join(TEMPLATES.keys())}"
        )
    return TEMPLATES[name]()


def list_templates() -> list:
    """List available template names."""
    return list(TEMPLATES.keys())


def auto_detect_template(sender: str, subject: str) -> OTPTemplate:
    """Auto-detect the best template based on email sender and subject.
    
    Prioritizes sender matching over subject matching for accuracy.
    """
    sender_lower = sender.lower()
    subject_lower = subject.lower()

    # First pass: match by sender only (highest confidence)
    for name, template_cls in TEMPLATES.items():
        if name == "generic":
            continue
        template = template_cls()
        for pattern in template.SENDERS:
            if pattern.lower() in sender_lower:
                return template

    # Second pass: match by subject (lower confidence)
    for name, template_cls in TEMPLATES.items():
        if name == "generic":
            continue
        template = template_cls()
        for pattern in template.SUBJECT_PATTERNS:
            if re.search(pattern, subject_lower, re.IGNORECASE):
                return template

    return GenericTemplate()
