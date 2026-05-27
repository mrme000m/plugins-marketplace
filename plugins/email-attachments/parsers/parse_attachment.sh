#!/usr/bin/env bash
# parse_attachment.sh — Wrapper for the Python-based attachment processor
#
# USAGE:
#   parse_attachment.sh <attachment-file> [output-dir] [--sender "..."] [--msgid "..."]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check for python3
if ! command -v python3 &>/dev/null; then
    echo "Error: python3 is required." >&2
    exit 1
fi

# Ensure output directory exists (if caller passed a specific path)
if [[ "${2:-}" != "" ]]; then
    mkdir -p "$2"
fi

# Run the Python processor
# We use -m to handle relative imports correctly within the py package
# Note: we need to be in the parent directory of 'py' to use -m py.attachment_processor
# or we can set PYTHONPATH.
export PYTHONPATH="$SCRIPT_DIR:${PYTHONPATH:-}"
python3 -m py.attachment_processor "$@"
