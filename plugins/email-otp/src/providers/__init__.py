"""Email provider modules for IMAP connections."""

from .base import EmailProvider
from .gmail import GmailProvider
from .icloud import ICloudProvider
from .outlook import OutlookProvider

PROVIDERS = {
    "gmail": GmailProvider,
    "icloud": ICloudProvider,
    "outlook": OutlookProvider,
}


def get_provider(name: str, **kwargs) -> EmailProvider:
    """Get a provider instance by name."""
    name = name.lower()
    if name not in PROVIDERS:
        raise ValueError(
            f"Unknown provider: {name}. Available: {', '.join(PROVIDERS.keys())}"
        )
    return PROVIDERS[name](**kwargs)


def list_providers() -> list:
    """List available provider names."""
    return list(PROVIDERS.keys())
