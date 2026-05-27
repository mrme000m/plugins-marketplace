import email
import os
import re
import shutil
from email import policy
from email.parser import BytesParser

from .base import BaseParser
from .core import logger, read_file_safe


class TextParser(BaseParser):
    def parse(self):
        ext = os.path.splitext(self.basename)[1].lower()
        logger.info(f"[TEXT] Processing: {self.basename} (ext: {ext})")

        if ext == ".eml" or self._is_eml_by_mime():
            self.handle_eml()
        elif ext == ".ics":
            self.handle_ics()
        else:
            self.handle_plain_text()

    def _is_eml_by_mime(self):
        """Check file magic to detect EML without extension."""
        try:
            with open(self.file_path, "rb") as f:
                header = f.read(1024)
                # Look for email headers
                return b"From:" in header or b"Return-Path:" in header or b"Received:" in header
        except Exception:
            return False

    def handle_eml(self):
        """Parse an RFC-822 email, extracting headers, HTML/plain body, and embedded attachments."""
        try:
            with open(self.file_path, "rb") as f:
                msg = BytesParser(policy=policy.default).parse(f)

            # ── Headers ─────────────────────────────────────────────────
            subject = msg.get("Subject", "(no subject)")
            frm = msg.get("From", "N/A")
            to = msg.get("To", "N/A")
            date = msg.get("Date", "N/A")
            msgid = msg.get("Message-ID", "N/A")
            content_type = msg.get_content_type()

            # ── Body extraction ─────────────────────────────────────────
            plain_body = ""
            html_body = ""
            is_multipart = msg.is_multipart()

            if is_multipart:
                for part in msg.walk():
                    ctype = part.get_content_type()
                    disp = part.get_content_disposition() or ""

                    # Skip attachments for body extraction
                    if "attachment" in disp:
                        continue
                    # Skip multipart containers
                    if ctype.startswith("multipart/"):
                        continue

                    if ctype == "text/plain" and not plain_body:
                        plain_body = part.get_content() or ""
                    elif ctype == "text/html" and not html_body:
                        html_body = part.get_content() or ""
            else:
                body = msg.get_content() or ""
                if msg.get_content_type() == "text/html":
                    html_body = body
                else:
                    plain_body = body

            # Use HTML as fallback if no plain text
            body_for_txt = plain_body if plain_body else html_body
            body_for_md = plain_body if plain_body else html_body
            has_html = bool(html_body)

            # ── Embedded attachments ───────────────────────────────────
            embedded_dir = os.path.join(self.norm_dir, "embedded")
            os.makedirs(embedded_dir, exist_ok=True)

            attachments_info = []
            has_attachments = False

            # Try to import and use the embedded handler
            try:
                from .eml_embedded import (
                    build_embedded_index,
                    extract_attachments,
                    recursively_process_attachments,
                )

                parent_msgid = self.msgid or self._msgid_from_basename()

                # Extract all attachments
                attachments = extract_attachments(
                    self.file_path, embedded_dir,
                    parent_msgid=parent_msgid,
                    parent_sender=frm
                )

                if attachments:
                    has_attachments = True
                    # Recursively process each
                    parent_out = os.path.dirname(self.norm_dir)  # parent attachment output dir
                    processed = recursively_process_attachments(
                        attachments, parent_out,
                        parent_msgid=parent_msgid,
                        parent_sender=frm
                    )
                    # Build index
                    index_path = build_embedded_index(
                        parent_msgid,
                        attachments, processed, self.norm_dir
                    )
                    logger.info(f"  Embedded index: {index_path}")
                    attachments_info = processed

            except ImportError as e:
                logger.warning(f"eml_embedded module not available: {e}")
                # Manual fallback: list attachments without processing
                for part in msg.walk():
                    disp = part.get_content_disposition() or ""
                    fn = part.get_filename()
                    ctype = part.get_content_type()
                    if fn and "attachment" in disp:
                        has_attachments = True
                        payload = part.get_payload(decode=True) or b""
                        safe_fn = re.sub(r'[<>:"/\\|?*\x00-\x1f]', "_", fn)
                        save_path = os.path.join(embedded_dir, safe_fn)
                        with open(save_path, "wb") as f:
                            f.write(payload)
                        attachments_info.append({
                            "filename": safe_fn,
                            "mime_type": ctype,
                            "size_bytes": len(payload),
                            "saved_path": save_path,
                        })

            # ── Save raw MIME for inspection ───────────────────────────
            raw_path = os.path.join(self.norm_dir, "raw.mime")
            try:
                shutil.copy2(self.file_path, raw_path)
            except (OSError, shutil.SameFileError) as e:
                logger.warning(f"Could not save raw MIME copy: {e}")

            # ── Build text output ──────────────────────────────────────
            txt_lines = [
                f"From: {frm}",
                f"To: {to}",
                f"Subject: {subject}",
                f"Date: {date}",
                f"Message-ID: {msgid}",
                f"Content-Type: {content_type}",
                "",
                body_for_txt if body_for_txt else "[no body text]",
            ]
            self.write_text("\n".join(txt_lines))

            # ── Build markdown output ──────────────────────────────────
            md_lines = [
                f"# Email: {subject}",
                f"- **From:** {frm}",
                f"- **To:** {to}",
                f"- **Date:** {date}",
                f"- **Message-ID:** {msgid}",
                f"- **Content-Type:** {content_type}",
                "",
            ]

            if has_html:
                md_lines.append("## Body (HTML)")
                md_lines.append("```html")
                md_lines.append(html_body)
                md_lines.append("```")
                md_lines.append("")
            else:
                md_lines.append("## Body")
                md_lines.append(body_for_md if body_for_md else "[no body text]")
                md_lines.append("")

            # ── Attachments section ────────────────────────────────────
            md_lines.append("## Attachments")
            if has_attachments:
                for att in attachments_info:
                    md_lines.append(f"- `{att['filename']}` ({att.get('mime_type', 'unknown')}, "
                                    f"{att.get('size_bytes', '?')} bytes) "
                                    f"→ [{att.get('status', 'saved')}]")
            else:
                md_lines.append("_None_")

            self.write_markdown("\n".join(md_lines))
            logger.info(f"✓ EML parsed: {self.basename} (attachments: {len(attachments_info)})")

        except Exception as e:
            logger.error(f"EML parsing error: {e}")
            self.handle_plain_text(error=f"EML parsing error: {e}")

    def _msgid_from_basename(self):
        """Generate a message ID from the filename."""
        slug = re.sub(r"[^a-z0-9]", "-", self.basename.lower()).strip("-")[:50]
        return slug or "eml-unknown"

    def handle_ics(self):
        try:
            content = read_file_safe(self.file_path)
            lines = content.splitlines()
            events = []
            current = {}
            in_event = False
            for line in lines:
                line_s = line.strip()
                if line_s == "BEGIN:VEVENT":
                    in_event = True
                    current = {}
                elif line_s == "END:VEVENT":
                    in_event = False
                    events.append(current)
                elif in_event and ":" in line_s:
                    key, val = line_s.split(":", 1)
                    if ";" in key:
                        key = key.split(";", 1)[0]
                    current[key] = val

            md_lines = [f"# Calendar: {self.basename}"]
            md_lines.append(f"- **Total events:** {len(events)}")
            md_lines.append("")

            if events:
                for i, ev in enumerate(events, 1):
                    md_lines.append(f"## Event {i}")
                    md_lines.append(f"- **Summary:** {ev.get('SUMMARY', '(no title)')}")
                    md_lines.append(f"- **Start:** {ev.get('DTSTART', 'N/A')}")
                    md_lines.append(f"- **End:** {ev.get('DTEND', 'N/A')}")
                    md_lines.append(f"- **Location:** {ev.get('LOCATION', 'N/A')}")
                    md_lines.append(f"- **Organizer:** {ev.get('ORGANIZER', 'N/A')}")
                    desc = ev.get("DESCRIPTION", "")
                    if desc:
                        md_lines.append(f"- **Description:** {desc}")
                    md_lines.append("")
            else:
                md_lines.append("No VEVENT blocks found; raw ICS output:")
                md_lines.append("```")
                md_lines.extend([l.rstrip() for l in lines])
                md_lines.append("```")

            self.write_markdown("\n".join(md_lines))
            self.write_text(
                "\n".join(md_lines)
            )  # For ICS, MD and TXT are similar enough

        except Exception as e:
            logger.error(f"ICS parsing error: {e}")
            self.handle_plain_text(error=f"ICS parsing error: {e}")

    def handle_plain_text(self, error=None):
        content = read_file_safe(self.file_path)
        if error:
            content = f"[{error}]\n\n" + content
        self.write_text(content)
        self.generate_markdown_wrapper("Text File")
