import os

from .core import logger, read_file_safe


class BaseParser:
    """Base class for all attachment parsers."""

    def __init__(self, file_path, norm_dir, preview_dir, msgid=None):
        self.file_path = file_path
        self.norm_dir = norm_dir
        self.preview_dir = preview_dir
        self.basename = os.path.basename(file_path)
        self.msgid = msgid
        self.content_txt_path = os.path.join(norm_dir, "content.txt")
        self.content_md_path = os.path.join(norm_dir, "content.md")

        os.makedirs(norm_dir, exist_ok=True)
        os.makedirs(preview_dir, exist_ok=True)

    def parse(self):
        """Main parsing logic to be implemented by subclasses."""
        raise NotImplementedError("Subclasses must implement parse()")

    def write_text(self, text):
        """Write extracted text to content.txt."""
        with open(self.content_txt_path, "w", encoding="utf-8") as f:
            f.write(text)

    def write_markdown(self, markdown):
        """Write extracted markdown to content.md."""
        with open(self.content_md_path, "w", encoding="utf-8") as f:
            f.write(markdown)

    def generate_markdown_wrapper(self, title, extra_meta=None, text_content=None):
        """Generate a standard markdown wrapper for the extracted text."""
        if text_content is None:
            text_content = read_file_safe(self.content_txt_path)

        lines = [f"# {title}: {self.basename}", ""]
        if extra_meta:
            for key, value in extra_meta.items():
                lines.append(f"- **{key}:** {value}")
            lines.append("")

        lines.append(text_content)
        self.write_markdown("\n".join(lines))
