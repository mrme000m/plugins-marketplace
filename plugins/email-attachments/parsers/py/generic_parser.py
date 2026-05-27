import os

from .base import BaseParser
from .core import get_mime_type, has_cmd, logger, read_file_safe, run_command


class GenericParser(BaseParser):
    def parse(self):
        logger.info(f"[GENERIC] Processing: {self.basename}")
        mime = get_mime_type(self.file_path)

        # 1. File type info
        with open(os.path.join(self.norm_dir, "fileinfo.txt"), "w") as f:
            f.write(f"{self.basename}: {mime}\n")
            if has_cmd("file"):
                res = run_command(["file", self.file_path])
                if res:
                    f.write(res.stdout)

        # 2. Hex dump
        if has_cmd("xxd"):
            run_command(
                ["xxd", "-l", "4096", self.file_path], capture_output=True
            )  # Could write to file
        elif has_cmd("hexdump"):
            run_command(
                ["hexdump", "-C", "-n", "4096", self.file_path], capture_output=True
            )

        # 3. Strings
        strings_path = os.path.join(self.norm_dir, "strings.txt")
        if has_cmd("strings"):
            res = run_command(["strings", "-n", "8", self.file_path])
            if res:
                with open(strings_path, "w") as f:
                    f.write(res.stdout)

        # 4. Heuristics
        if mime.startswith("text/"):
            content = read_file_safe(self.file_path)
            self.write_text(content)
            self.generate_markdown_wrapper("Text-like File")
        else:
            self.write_text(f"[Binary/unknown file: {mime}]")
            md_lines = [
                f"# Unknown/Binary File: {self.basename}",
                "",
                f"- MIME type: {mime}",
                "- See hexdump.txt and strings.txt for manual analysis",
                "",
                "## Extracted Strings (first 100)",
                "```",
            ]
            if os.path.exists(strings_path):
                with open(strings_path, "r") as f:
                    md_lines.extend([line.strip() for _, line in zip(range(100), f)])
            md_lines.append("```")
            self.write_markdown("\n".join(md_lines))

        logger.info(f"✓ Generic parsed: {self.basename}")
