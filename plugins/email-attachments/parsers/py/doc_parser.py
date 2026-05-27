import os
import shutil

from .base import BaseParser
from .core import get_mime_type, has_cmd, logger, run_command


class DOCParser(BaseParser):
    def parse(self):
        mime = get_mime_type(self.file_path)
        logger.info(f"[OLE] Processing: {self.basename} (type: {mime})")

        # 1. Metadata Analysis
        if has_cmd("oleid"):
            run_command(
                ["oleid", self.file_path], capture_output=True
            )  # Write to file later if needed

        if has_cmd("olemeta"):
            meta_res = run_command(["olemeta", self.file_path])
            if meta_res:
                with open(os.path.join(self.norm_dir, "olemeta.txt"), "w") as f:
                    f.write(meta_res.stdout)

        if has_cmd("oleobj"):
            obj_dir = os.path.join(self.norm_dir, "ole_objects")
            os.makedirs(obj_dir, exist_ok=True)
            run_command(["oleobj", self.file_path, "-d", obj_dir])

        # 2. Text Extraction
        text_extracted = False

        if has_cmd("antiword"):
            res = run_command(["antiword", self.file_path])
            if res and res.returncode == 0:
                self.write_text(res.stdout)
                text_extracted = True

        if not text_extracted and (has_cmd("soffice") or has_cmd("libreoffice")):
            soffice = "soffice" if has_cmd("soffice") else "libreoffice"
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
            txt_output = os.path.join(
                self.norm_dir, os.path.splitext(self.basename)[0] + ".txt"
            )
            if os.path.exists(txt_output):
                os.rename(txt_output, self.content_txt_path)
                text_extracted = True

        if not text_extracted and has_cmd("strings"):
            res = run_command(["strings", "-n", "8", self.file_path])
            if res:
                self.write_text(res.stdout)
                text_extracted = True

        if not text_extracted:
            self.write_text(f"[No text extraction available for {self.basename}]")

        self.generate_markdown_wrapper(
            "Legacy Office Document", extra_meta={"MIME type": mime}
        )
        logger.info(f"✓ OLE parsed: {self.basename}")
