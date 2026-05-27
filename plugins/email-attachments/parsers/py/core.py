import hashlib
import json
import logging
import os
import shutil
import subprocess
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(levelname)s: %(message)s")
logger = logging.getLogger("attachment_parser")


def has_cmd(cmd):
    """Check if a command exists on the system."""
    return shutil.which(cmd) is not None


def run_command(args, capture_output=True, text=True, check=False):
    """Run a shell command and return the result."""
    try:
        # If text is True, use utf-8 and ignore errors to prevent crashes on binary noise
        kwargs = {
            "capture_output": capture_output,
            "check": check,
        }
        if text:
            kwargs["text"] = True
            kwargs["errors"] = "replace"

        result = subprocess.run(args, **kwargs)
        return result
    except FileNotFoundError:
        # Command not found is handled by has_cmd check usually,
        # but this is a safety net.
        return None
    except subprocess.CalledProcessError as e:
        logger.warning(
            f"Command failed with exit code {e.returncode}: {' '.join(args)}"
        )
        return e
    except Exception as e:
        logger.error(f"Unexpected error running {' '.join(args)}: {e}")
        return None


def log_success(msg):
    """Log a success message with a checkmark."""
    logger.info(f"✓ {msg}")


def log_warn(msg):
    """Log a warning message with an alert symbol."""
    logger.warning(f"⚠ {msg}")


def hash_file(file_path):
    """Calculate SHA256 hash of a file."""
    sha256_hash = hashlib.sha256()
    try:
        with open(file_path, "rb") as f:
            for byte_block in iter(lambda: f.read(4096), b""):
                sha256_hash.update(byte_block)
    except OSError as e:
        logger.error(f"Cannot read file for hashing: {file_path} ({e})")
        return "hash_unavailable"
    return sha256_hash.hexdigest()


def get_mime_type(file_path):
    """Get MIME type of a file using 'file' command."""
    if has_cmd("file"):
        res = run_command(["file", "--brief", "--mime-type", file_path])
        if res and res.returncode == 0:
            return res.stdout.strip()
    return "application/octet-stream"


def get_file_size(file_path):
    """Get size of a file in bytes."""
    return os.path.getsize(file_path)


def get_timestamp():
    """Get current UTC timestamp in ISO 8601 format."""
    return datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")


def get_apryse_args(cmd):
    """Get Apryse command arguments with license key if available."""
    args = [cmd]
    lic_key = os.environ.get("APRYSE_LICENSE_KEY")
    if lic_key:
        args.extend(["--lic_key", lic_key])
    return args


def write_json(data, file_path):
    """Write data to a JSON file."""
    with open(file_path, "w") as f:
        json.dump(data, f, indent=2)


def read_file_safe(file_path, default=""):
    """Read file content safely, returning default on error."""
    if not os.path.exists(file_path):
        return default
    try:
        with open(file_path, "r", errors="replace") as f:
            return f.read()
    except (OSError, UnicodeDecodeError) as e:
        logger.warning(f"Cannot read file: {file_path} ({e})")
        return default
