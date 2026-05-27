import os
import shutil

from .base import BaseParser
from .core import has_cmd, logger, run_command


class ImageParser(BaseParser):
    def parse(self):
        logger.info(f"[IMAGE] Processing: {self.basename}")

        # 1. EXIF metadata
        if has_cmd("exiftool"):
            res = run_command(["exiftool", self.file_path])
            if res:
                with open(os.path.join(self.norm_dir, "exif.txt"), "w") as f:
                    f.write(res.stdout)

        # 2. Cleaned preview
        cleaned_path = os.path.join(self.preview_dir, "cleaned.png")
        if has_cmd("magick"):
            run_command(
                [
                    "magick",
                    self.file_path,
                    "-colorspace",
                    "Gray",
                    "-density",
                    "300",
                    "-deskew",
                    "40%",
                    "-contrast-stretch",
                    "2%x2%",
                    "-normalize",
                    cleaned_path,
                ]
            )
        elif has_cmd("convert"):
            run_command(["convert", self.file_path, cleaned_path])

        # 3. OCR
        if has_cmd("tesseract"):
            img_for_ocr = (
                cleaned_path if os.path.exists(cleaned_path) else self.file_path
            )
            res = run_command(["tesseract", img_for_ocr, "stdout", "--psm", "6"])
            if res and res.returncode == 0:
                self.write_text(res.stdout)
            else:
                self.write_text("[OCR failed or no text detected]")
        else:
            self.write_text("[tesseract not installed]")

        # 4. Copy original to preview
        ext = os.path.splitext(self.basename)[1].lower() or ".png"
        shutil.copy2(self.file_path, os.path.join(self.preview_dir, f"original{ext}"))

        # 5. Markdown
        extra = {}
        if os.path.exists(os.path.join(self.norm_dir, "exif.txt")):
            extra["Metadata"] = "EXIF available in exif.txt"
        self.generate_markdown_wrapper("Image", extra_meta=extra)
        logger.info(f"✓ Image parsed: {self.basename}")
