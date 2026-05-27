import argparse
import json
import os

from .core import logger, read_file_safe


def run_audit(msgid, output_base="./attachments"):
    dest_dir = os.path.join(output_base, msgid)
    norm_dir = os.path.join(dest_dir, "normalized")

    meta_path = os.path.join(norm_dir, "meta.json")
    risk_path = os.path.join(norm_dir, "risk.json")
    content_path = os.path.join(norm_dir, "content.txt")

    if not os.path.exists(meta_path) or not os.path.exists(risk_path):
        logger.error(f"Missing artifacts for msgid: {msgid}")
        return None

    with open(meta_path, "r") as f:
        meta = json.load(f)
    with open(risk_path, "r") as f:
        risk = json.load(f)

    content = read_file_safe(content_path)

    # Simple rule-based scoring (Claude will do the heavy lifting)
    score = 0
    findings = []

    if meta.get("extension_spoofed"):
        score += 5
        findings.append("CRITICAL: Extension spoofing detected.")

    if risk.get("security_checks", {}).get("yara_hits"):
        score += 10
        findings.append(f"CRITICAL: YARA hits: {risk['security_checks']['yara_hits']}")

    if risk.get("security_checks", {}).get("suspicious_strings"):
        score += 3
        findings.append(
            f"WARNING: Suspicious strings found: {len(risk['security_checks']['suspicious_strings'])}"
        )

    # Urgency detection
    urgency_patterns = [
        "urgent",
        "suspended",
        "immediate",
        "action required",
        "verify",
        "identity",
    ]
    if any(p in content.lower() for p in urgency_patterns):
        score += 2
        findings.append(
            "WARNING: Urgency or social engineering cues detected in content."
        )

    verdict = "safe-organize"
    if score >= 7:
        verdict = "high-risk-quarantine"
    elif score >= 3:
        verdict = "needs-review"

    report = {
        "msgid": msgid,
        "filename": meta.get("original_filename"),
        "verdict": verdict,
        "score": min(score, 10),
        "findings": findings,
        "meta": meta,
        "risk": risk,
    }

    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Audit attachment artifacts.")
    parser.add_argument("msgid", help="Message ID of parsed attachment")
    parser.add_argument("--out", default="./attachments", help="Output base directory")

    args = parser.parse_args()
    report = run_audit(args.msgid, args.out)
    if report:
        print(json.dumps(report, indent=2))
