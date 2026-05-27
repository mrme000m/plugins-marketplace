"""EML embedded attachment handler — extracts and recursively processes nested attachments.

Guard: recursion depth is capped at MAX_DEPTH (default 5) to prevent infinite loops
from maliciously nested EML files.
"""

import json
import os
import re
import shutil
import subprocess
from email import policy
from email.parser import BytesParser

MAX_DEPTH = 5
_DEPTH_ENV = "EML_EMBEDDED_DEPTH"


def current_depth():
    """Return the current recursion depth from environment."""
    try:
        return int(os.environ.get(_DEPTH_ENV, "0"))
    except ValueError:
        return 0


def inc_depth():
    """Increment and return the new recursion depth."""
    d = current_depth() + 1
    os.environ[_DEPTH_ENV] = str(d)
    return d


def dec_depth():
    """Decrement recursion depth after processing."""
    d = max(0, current_depth() - 1)
    os.environ[_DEPTH_ENV] = str(d)


def extract_attachments(eml_path, output_dir, parent_msgid, parent_sender=None):
    """Extract all embedded attachments from an EML file, save them, and return metadata.

    Returns a list of dicts with keys:
        - filename, mime_type, size_bytes, saved_path, content_id
    """
    attachments = []
    os.makedirs(output_dir, exist_ok=True)

    with open(eml_path, "rb") as f:
        msg = BytesParser(policy=policy.default).parse(f)

    for part in msg.walk():
        disp = part.get_content_disposition() or ""
        ctype = part.get_content_type()

        # Skip the message container itself
        if ctype.startswith("multipart/"):
            continue

        # Consider attachment if disposition says so, or it's a non-inline non-text part
        fn = part.get_filename()
        is_inline_image = ctype.startswith("image/") and "inline" in disp
        is_attachment = "attachment" in disp or (fn and not ctype.startswith("text/"))

        if not fn and is_inline_image:
            # Generate a name for inline images
            cid = part.get("Content-ID", "")
            ext = ctype.split("/")[-1]
            if cid:
                fn = f"inline_{sanitize_filename(cid.strip('<>'))}.{ext}"
            else:
                fn = f"inline_image_{len(attachments)}.{ext}"

        if not fn:
            continue

        fn = sanitize_filename(fn)
        payload = part.get_payload(decode=True) or b""

        save_path = os.path.join(output_dir, fn)
        # Handle duplicates
        counter = 1
        base_fn = fn
        while os.path.exists(save_path):
            stem, ext = os.path.splitext(base_fn)
            save_path = os.path.join(output_dir, f"{stem}_{counter}{ext}")
            counter += 1

        with open(save_path, "wb") as f:
            f.write(payload)

        attachments.append({
            "filename": os.path.basename(save_path),
            "mime_type": ctype,
            "size_bytes": len(payload),
            "saved_path": save_path,
            "content_id": part.get("Content-ID", ""),
            "content_disposition": disp,
        })

    return attachments


def sanitize_filename(name):
    """Sanitize a filename for safe filesystem use."""
    name = name.strip()
    # Remove path traversal attempts
    name = os.path.basename(name)
    # Replace unsafe chars
    name = re.sub(r'[<>:`"/\\|?*\x00-\x1f]', "_", name)
    # Limit length
    if len(name) > 255:
        stem, ext = os.path.splitext(name)
        name = stem[:255 - len(ext)] + ext
    if not name:
        name = "unnamed_attachment"
    return name


def recursively_process_attachments(attachments, output_base, parent_msgid, parent_sender=None):
    """Recursively call parse_router.sh (or attachment_processor) on each extracted attachment.

    Returns a list of result dicts with nested processing info.
    """
    results = []
    script_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    parse_router = os.path.join(script_dir, "parse_router.sh")

    # Check recursion depth before proceeding
    depth = inc_depth()
    try:
        if depth > MAX_DEPTH:
            for att in attachments:
                results.append({
                    "filename": att["filename"],
                    "mime_type": att["mime_type"],
                    "status": "depth_exceeded",
                    "output_dir": "",
                    "depth": depth,
                })
            return results

        for att in attachments:
            att_path = att["saved_path"]
            att_fn = att["filename"]

            # Build a nested output dir under the parent's output
            att_slug = re.sub(r"[^a-z0-9]", "-", att_fn.lower()).strip("-")[:40]
            att_msgid = f"{parent_msgid}-embedded-{att_slug}"
            att_out_dir = os.path.join(output_base, att_msgid)
            att_norm_dir = os.path.join(att_out_dir, "normalized")
            att_preview_dir = os.path.join(att_norm_dir, "preview")
            os.makedirs(att_norm_dir, exist_ok=True)
            os.makedirs(att_preview_dir, exist_ok=True)

            # Run the router on this embedded attachment
            result = {
                "filename": att_fn,
                "mime_type": att["mime_type"],
                "status": "pending",
                "depth": depth,
            }
            try:
                if os.path.exists(parse_router):
                    env = os.environ.copy()
                    env["PYTHONPATH"] = f"{script_dir}:{env.get('PYTHONPATH', '')}"
                    res = subprocess.run(
                        ["bash", parse_router, att_path, att_norm_dir, att_preview_dir],
                        capture_output=True,
                        text=True,
                        timeout=120,
                        env=env,
                    )
                    result["returncode"] = res.returncode
                    result["output_preview"] = res.stdout[-500:] if res.stdout else ""
                    if res.returncode == 0:
                        result["status"] = "success"
                    else:
                        result["status"] = "failed"
                        result["stderr"] = res.stderr[:500] if res.stderr else ""
                else:
                    # Fallback: just copy as generic
                    shutil.copy2(att_path, os.path.join(att_norm_dir, "content.bin"))
                    result["status"] = "generic_fallback"
            except subprocess.TimeoutExpired:
                result["status"] = "timeout"
            except OSError as e:
                result["status"] = "os_error"
                result["error"] = str(e)
            except Exception as e:
                result["status"] = "exception"
                result["error"] = str(e)

            result["output_dir"] = att_out_dir
            results.append(result)
    finally:
        dec_depth()

    return results


def build_embedded_index(parent_msgid, attachments, processed_results, output_dir):
    """Write embedded_index.json summarizing all nested attachments."""
    index = {
        "parent_message_id": parent_msgid,
        "total_embedded": len(attachments),
        "extracted_attachments": [],
    }

    for att, proc in zip(attachments, processed_results):
        index["extracted_attachments"].append({
            "filename": att["filename"],
            "mime_type": att["mime_type"],
            "size_bytes": att["size_bytes"],
            "content_id": att["content_id"],
            "processing_status": proc.get("status", "unknown"),
            "output_dir": proc.get("output_dir", ""),
        })

    index_path = os.path.join(output_dir, "embedded_index.json")
    with open(index_path, "w") as f:
        json.dump(index, f, indent=2)

    return index_path
