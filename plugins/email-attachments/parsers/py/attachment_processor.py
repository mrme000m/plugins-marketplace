import json
import os
import re
import shutil
import sys
import uuid

from .core import (
    get_file_size,
    get_mime_type,
    get_timestamp,
    has_cmd,
    hash_file,
    logger,
    run_command,
    write_json,
)
from .doc_parser import DOCParser
from .docx_parser import DOCXParser
from .generic_parser import GenericParser
from .image_parser import ImageParser

# Import parsers
from .pdf_parser import PDFParser
from .pptx_parser import PPTXParser
from .text_parser import TextParser
from .xlsx_parser import XLSXParser


class AttachmentProcessor:
    def __init__(
        self, attachment_path, output_base="./attachments", sender=None, msgid=None
    ):
        self.attachment_path = attachment_path
        self.output_base = output_base
        self.sender = sender
        self.basename = os.path.basename(attachment_path)

        if msgid:
            self.msgid = msgid
        else:
            # Generate a readable ID based on filename + first 8 of SHA256
            name_slug = re.sub(r"[^a-z0-9]", "-", self.basename.lower()).strip("-")
            sha_short = hash_file(attachment_path)[:8]
            self.msgid = f"{name_slug}-{sha_short}"

        self.dest_dir = os.path.join(self.output_base, self.msgid)
        self.orig_dir = os.path.join(self.dest_dir, "original")
        self.norm_dir = os.path.join(self.dest_dir, "normalized")
        self.preview_dir = os.path.join(self.norm_dir, "preview")

        os.makedirs(self.orig_dir, exist_ok=True)
        os.makedirs(self.norm_dir, exist_ok=True)
        os.makedirs(self.preview_dir, exist_ok=True)

    def process(self):
        logger.info(f"Starting processing for: {self.basename}")

        # 1. Identity
        sha256 = hash_file(self.attachment_path)
        mime = get_mime_type(self.attachment_path)
        size = get_file_size(self.attachment_path)

        # Copy to original
        shutil.copy2(self.attachment_path, os.path.join(self.orig_dir, self.basename))

        # 2. Type Mapping
        magic_ext = self.map_mime_to_ext(mime)
        ext = os.path.splitext(self.basename)[1].lower().lstrip(".")

        # Spoofing check
        spoofed = self.check_spoofing(ext, magic_ext, mime)

        # 3. Security Checks (Simplified YARA/Strings for now)
        risk_data = self.perform_risk_assessment(sha256, mime, size, spoofed)
        write_json(risk_data, os.path.join(self.norm_dir, "risk.json"))

        # 4. Meta
        meta_data = {
            "original_filename": self.basename,
            "sha256": sha256,
            "mime_type": mime,
            "file_size_bytes": size,
            "detected_type": magic_ext,
            "extension_spoofed": spoofed,
            "sender": self.sender,
            "message_id": self.msgid,
            "parsed_at": get_timestamp(),
            "tools_available": {
                cmd: has_cmd(cmd)
                for cmd in [
                    "pdf2text",
                    "docpub",
                    "pdftotext",
                    "pdfinfo",
                    "pandoc",
                    "tesseract",
                    "magick",
                    "exiftool",
                    "yara",
                    "oleid",
                    "antiword",
                    "soffice",
                ]
            },
            "output_paths": {
                "original": os.path.join(self.orig_dir, self.basename),
                "normalized_dir": self.norm_dir,
            },
        }
        write_json(meta_data, os.path.join(self.norm_dir, "meta.json"))

        # 5. Dispatch
        parser_cls = self.get_parser_class(magic_ext)
        parser = parser_cls(self.attachment_path, self.norm_dir, self.preview_dir, msgid=self.msgid)

        try:
            parser.parse()
        except Exception as e:
            logger.error(f"Parser {parser_cls.__name__} failed: {e}")
            # Fallback to generic
            GenericParser(self.attachment_path, self.norm_dir, self.preview_dir, msgid=self.msgid).parse()

        self.print_summary(meta_data, risk_data)
        return self.dest_dir

    def map_mime_to_ext(self, mime):
        mapping = {
            "application/pdf": "pdf",
            "application/vnd.openxmlformats-officedocument.wordprocessingml.document": "docx",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
            "application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",
            "application/msword": "doc",
            "application/vnd.ms-excel": "xls",
            "application/vnd.ms-powerpoint": "ppt",
            "text/calendar": "ics",
            "text/csv": "text",
            "text/html": "text",
            "message/rfc822": "eml",
            "application/zip": "zip",
            "application/x-7z-compressed": "zip",
            "application/x-rar-compressed": "zip",
            "application/gzip": "zip",
            "application/x-tar": "zip",
            "video/mp4": "video",
            "video/quicktime": "video",
            "video/x-msvideo": "video",
            "audio/mpeg": "audio",
            "audio/wav": "audio",
            "audio/x-wav": "audio",
            "audio/ogg": "audio",
            "application/x-dosexec": "exe",
            "application/x-executable": "exe",
        }
        if mime in mapping:
            return mapping[mime]
        if mime.startswith("image/"):
            return "image"
        if mime.startswith("text/"):
            return "text"
        if mime.startswith("video/"):
            return "video"
        if mime.startswith("audio/"):
            return "audio"
        return "unknown"

    def check_spoofing(self, ext, magic_ext, mime):
        if not ext:
            return False
        if ext == magic_ext:
            return False

        harmless = ["md", "txt", "csv", "json", "xml", "yaml", "yml", "log"]
        if ext in harmless and magic_ext == "text":
            return False

        if magic_ext != "unknown":
            logger.warning(f"Extension spoofing detected: .{ext} vs magic={mime}")
            return True
        return False

    def perform_risk_assessment(self, sha256, mime, size, spoofed):
        yara_hits = []
        if has_cmd("yara"):
            rules_path = os.path.join(os.path.dirname(__file__), "..", "..", "rules")
            if os.path.exists(rules_path):
                res = run_command(["yara", "-r", rules_path, self.attachment_path])
                if res and res.returncode == 0:
                    yara_hits = res.stdout.splitlines()[:10]

        suspicious_strings = []
        pattern = "powershell|cmd\\.exe|wscript|eval\\(|base64|http://|https://"

        # Benign patterns to filter out
        noise_patterns = [
            "xmlns:",
            "purl.org",
            "ns.adobe.com",
            "w3.org",
            "schema.org",
            "schemas.microsoft.com",
            "openxmlformats.org",
            "Content-Transfer-Encoding:",
            "Content-Type:",
            "X-MS-Has-Attach:",
        ]
        noise_regex = re.compile(
            "|".join(re.escape(p) for p in noise_patterns), re.IGNORECASE
        )

        if has_cmd("strings"):
            # Use strings to avoid "Binary file matches" message from grep
            # and to safely extract text from binaries.
            res_strings = run_command(["strings", "-n", "8", self.attachment_path])
            if res_strings and res_strings.stdout:
                # Search within the extracted strings
                lines = res_strings.stdout.splitlines()
                regex = re.compile(pattern, re.IGNORECASE)
                for line in lines:
                    line = line.strip()
                    if regex.search(line) and not noise_regex.search(line):
                        suspicious_strings.append(line)
                        if len(suspicious_strings) >= 20:
                            break
        elif has_cmd("grep"):
            # Fallback to grep -a (treat as text)
            res = run_command(["grep", "-aiE", pattern, self.attachment_path])
            if res and res.returncode == 0:
                for line in res.stdout.splitlines():
                    line = line.strip()
                    if not noise_regex.search(line):
                        suspicious_strings.append(line)
                        if len(suspicious_strings) >= 20:
                            break

        return {
            "timestamp": get_timestamp(),
            "filename": self.basename,
            "sha256": sha256,
            "mime_type": mime,
            "file_size": size,
            "extension_spoofed": spoofed,
            "security_checks": {
                "magic_mismatch": spoofed,
                "yara_hits": yara_hits,
                "suspicious_strings": suspicious_strings,
                "overall_risk_score": "pending_claude_audit",
            },
        }

    def get_parser_class(self, magic_ext):
        mapping = {
            "pdf": PDFParser,
            "docx": DOCXParser,
            "xlsx": XLSXParser,
            "pptx": PPTXParser,
            "doc": DOCParser,
            "xls": DOCParser,
            "ppt": DOCParser,
            "ics": TextParser,
            "eml": TextParser,
            "text": TextParser,
            "image": ImageParser,
        }
        return mapping.get(magic_ext, GenericParser)

    def print_summary(self, meta, risk):
        print("\n" + "=" * 60)
        print(f"► ATTACHMENT PROCESSED: {self.basename}")
        print(f"  Detected Type: {meta['detected_type']} ({meta['mime_type']})")
        print(f"  Message-ID:    {self.msgid}")
        print(f"  SHA-256:       {meta['sha256'][:16]}...")

        print("\n  SECURITY SIGNALS:")
        print(
            f"    Extension Match: {'FAIL (Spoofed!)' if meta['extension_spoofed'] else 'PASS'}"
        )
        print(
            f"    YARA Hits:       {len(risk['security_checks']['yara_hits']) or 'None'}"
        )
        print(
            f"    Suspicious Str:  {len(risk['security_checks']['suspicious_strings']) or 'None'}"
        )

        print("\n  ARTIFACTS SAVED:")
        print(f"    Text Content:  {os.path.join(self.norm_dir, 'content.txt')}")
        print(f"    Markdown:      {os.path.join(self.norm_dir, 'content.md')}")
        print(f"    Risk Report:   {os.path.join(self.norm_dir, 'risk.json')}")

        # Check if previews were generated
        previews = os.listdir(self.preview_dir)
        if previews:
            print(f"    Previews:      {self.preview_dir} ({len(previews)} files)")

        print("=" * 60 + "\n")


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Process email attachments.")
    parser.add_argument("file", help="Path to attachment")
    parser.add_argument(
        "out", nargs="?", default="./attachments", help="Output directory"
    )
    parser.add_argument("--sender", help="Sender email")
    parser.add_argument("--msgid", help="Message ID")

    args = parser.parse_args()

    processor = AttachmentProcessor(args.file, args.out, args.sender, args.msgid)
    processor.process()
