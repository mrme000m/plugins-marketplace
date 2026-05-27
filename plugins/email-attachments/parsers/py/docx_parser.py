import os
import xml.etree.ElementTree as ET
import zipfile

from .base import BaseParser
from .core import has_cmd, logger, run_command


class DOCXParser(BaseParser):
    def parse(self):
        logger.info(f"[DOCX] Processing: {self.basename}")

        text_extracted = False
        from .core import get_apryse_args

        # 1. Apryse docpub (Highest quality conversion)
        if has_cmd("docpub"):
            logger.info("    → docpub (Apryse)")
            args = get_apryse_args("docpub")
            # docpub -f pdf -o outdir input.docx -> results in outdir/input.pdf
            res = run_command(args + ["-f", "pdf", "-o", self.norm_dir, self.file_path])
            pdf_out = os.path.join(
                self.norm_dir, os.path.splitext(self.basename)[0] + ".pdf"
            )
            if res and res.returncode == 0 and os.path.exists(pdf_out):
                # Now extract text from the high-quality PDF
                if has_cmd("pdf2text"):
                    args_p2t = get_apryse_args("pdf2text")
                    run_command(
                        args_p2t + ["-f", "plain", "-o", self.norm_dir, pdf_out]
                    )
                    apryse_txt = os.path.join(
                        self.norm_dir, os.path.splitext(self.basename)[0] + ".txt"
                    )
                    if os.path.exists(apryse_txt):
                        os.rename(apryse_txt, self.content_txt_path)
                        text_extracted = True
                elif has_cmd("pdftotext"):
                    run_command(
                        ["pdftotext", "-layout", pdf_out, self.content_txt_path]
                    )
                    text_extracted = True

                # Cleanup PDF if we have other extraction methods or keep for preview?
                # For now let's keep it if we need previews later or remove if redundant.
                # os.remove(pdf_out)

        # 2. Pandoc
        if not text_extracted and has_cmd("pandoc"):
            media_dir = os.path.join(self.norm_dir, "media")
            os.makedirs(media_dir, exist_ok=True)
            res = run_command(
                [
                    "pandoc",
                    f"--extract-media={media_dir}",
                    self.file_path,
                    "-t",
                    "gfm",
                    "-o",
                    self.content_md_path,
                ]
            )
            if res and res.returncode == 0:
                run_command(
                    [
                        "pandoc",
                        self.content_md_path,
                        "-t",
                        "plain",
                        "-o",
                        self.content_txt_path,
                    ]
                )
                text_extracted = True

        # 2. LibreOffice Fallback
        if not text_extracted and (has_cmd("soffice") or has_cmd("libreoffice")):
            soffice = "soffice" if has_cmd("soffice") else "libreoffice"
            logger.info("    → libreoffice fallback")
            run_command(
                [
                    soffice,
                    "--headless",
                    "--convert-to",
                    "txt",
                    "--outdir",
                    self.norm_dir,
                    self.file_path,
                ]
            )
            # soffice outputs to [basename].txt
            txt_output = os.path.join(
                self.norm_dir, os.path.splitext(self.basename)[0] + ".txt"
            )
            if os.path.exists(txt_output):
                os.rename(txt_output, self.content_txt_path)
                text_extracted = True

        # 3. Python XML Fallback
        if not text_extracted:
            logger.info("    → python XML fallback")
            try:
                with zipfile.ZipFile(self.file_path, "r") as z:
                    with z.open("word/document.xml") as f:
                        tree = ET.parse(f)
                texts = []
                ns_para = (
                    "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}p"
                )
                ns_text = (
                    "{http://schemas.openxmlformats.org/wordprocessingml/2006/main}t"
                )
                for para in tree.iter(ns_para):
                    line = "".join(t.text or "" for t in para.iter(ns_text))
                    if line:
                        texts.append(line)
                self.write_text("\n".join(texts))
                text_extracted = True
            except Exception as e:
                logger.warning(f"Python XML extraction failed: {e}")

        # 4. Strings Fallback
        if not text_extracted and has_cmd("strings"):
            res = run_command(["strings", "-n", "8", self.file_path])
            if res:
                self.write_text(res.stdout)
                text_extracted = True

        if not text_extracted:
            self.write_text(f"[No text extracted for {self.basename}]")

        self.generate_markdown_wrapper("Word Document")
        logger.info(f"✓ DOCX parsed: {self.basename}")
