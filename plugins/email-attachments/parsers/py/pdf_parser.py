import glob
import os

from .base import BaseParser
from .core import has_cmd, logger, run_command


class PDFParser(BaseParser):
    def parse(self):
        logger.info(f"[PDF] Processing: {self.basename}")

        # 1. Text extraction
        text_extracted = False

        # Try Apryse pdf2text first (Highest quality)
        from .core import get_apryse_args

        if has_cmd("pdf2text"):
            logger.info("    → pdf2text (Apryse)")
            args = get_apryse_args("pdf2text")
            # pdf2text -f plain -o outdir input.pdf -> results in outdir/input.txt
            res = run_command(
                args + ["-f", "plain", "-o", self.norm_dir, self.file_path]
            )
            apryse_txt = os.path.join(
                self.norm_dir, os.path.splitext(self.basename)[0] + ".txt"
            )
            if res and res.returncode == 0 and os.path.exists(apryse_txt):
                os.rename(apryse_txt, self.content_txt_path)
                text_extracted = True

        if not text_extracted and has_cmd("pdftotext"):
            logger.info("    → pdftotext (layout-preserving)")
            res = run_command(
                ["pdftotext", "-layout", self.file_path, self.content_txt_path]
            )
            if res and res.returncode == 0:
                text_extracted = True
            else:
                res = run_command(["pdftotext", self.file_path, self.content_txt_path])
                if res and res.returncode == 0:
                    text_extracted = True
        if not text_extracted:
            self.write_text("[pdftotext failed or not available]")

        # 2. Metadata
        num_pages = 0
        if has_cmd("pdfinfo"):
            info_path = os.path.join(self.norm_dir, "pdfinfo.txt")
            res = run_command(["pdfinfo", self.file_path])
            if res and res.returncode == 0:
                with open(info_path, "w") as f:
                    f.write(res.stdout)
                for line in res.stdout.splitlines():
                    if line.startswith("Pages:"):
                        try:
                            num_pages = int(line.split(":")[1].strip())
                        except ValueError:
                            pass

        # 3. Previews
        if has_cmd("pdftoppm") and num_pages > 0:
            run_command(
                [
                    "pdftoppm",
                    "-png",
                    "-r",
                    "180",
                    "-l",
                    "10",
                    self.file_path,
                    os.path.join(self.preview_dir, "page"),
                ]
            )

        # 4. OCR Fallback
        ocr_performed = False
        content_size = (
            os.path.getsize(self.content_txt_path)
            if os.path.exists(self.content_txt_path)
            else 0
        )
        if content_size < 100 and num_pages > 0 and has_cmd("tesseract"):
            ocr_performed = True
            with open(self.content_txt_path, "a") as f:
                f.write("\n--- OCR FALLBACK ---\n")
                preview_files = sorted(
                    glob.glob(os.path.join(self.preview_dir, "page-*.png"))
                )
                f.write("\n--- OCR FALLBACK ---\n")
                preview_files = sorted(
                    glob.glob(os.path.join(self.preview_dir, "page-*.png"))
                )
                for i, img in enumerate(preview_files, 1):
                    f.write(f"\n--- Page {i} ---\n")
                    res = run_command(["tesseract", img, "stdout", "--psm", "6"])
                    if res and res.returncode == 0:
                        f.write(res.stdout)

        # 5. Markdown
        meta = {"Pages": num_pages, "OCR Fallback": "Yes" if ocr_performed else "No"}
        self.generate_markdown_wrapper("PDF Content", extra_meta=meta)
        logger.info(f"✓ PDF parsed: {self.basename}")
