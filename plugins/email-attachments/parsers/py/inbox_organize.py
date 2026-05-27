import argparse
import json
import os

from .core import logger, read_file_safe


def run_organize(msgid, output_base="./attachments"):
    dest_dir = os.path.join(output_base, msgid)
    norm_dir = os.path.join(dest_dir, "normalized")

    meta_path = os.path.join(norm_dir, "meta.json")
    content_path = os.path.join(norm_dir, "content.txt")

    if not os.path.exists(meta_path):
        logger.error(f"Missing artifacts for msgid: {msgid}")
        return None

    with open(meta_path, "r") as f:
        meta = json.load(f)
    content = read_file_safe(content_path).lower()

    # Heuristic-based classification
    category = "other"
    priority = "medium"
    folder = "Inbox/"

    if any(
        p in content for p in ["invoice", "bill", "amount due", "statement", "payment"]
    ):
        category = "invoice"
        folder = "Invoices/"
        priority = "high"
    elif any(p in content for p in ["contract", "agreement", "nda", "terms"]):
        category = "contract"
        folder = "Contracts/"
        priority = "high"
    elif any(p in content for p in ["resume", "cv", "application", "candidate"]):
        category = "resume"
        folder = "Candidates/"
    elif any(p in content for p in ["report", "analytics", "summary", "status"]):
        category = "report"
        folder = "Reports/"

    # Entity Extraction (Basic Regex/Search)
    # This is where Claude really shines, so we provide a baseline

    result = {
        "msgid": msgid,
        "filename": meta.get("original_filename"),
        "category": category,
        "priority": priority,
        "suggested_folder": folder,
        "summary": f"Detected as {category} based on keywords.",
    }

    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Classify parsed attachment.")
    parser.add_argument("msgid", help="Message ID of parsed attachment")
    parser.add_argument("--out", default="./attachments", help="Output base directory")

    args = parser.parse_args()
    result = run_organize(args.msgid, args.out)
    if result:
        print(json.dumps(result, indent=2))
