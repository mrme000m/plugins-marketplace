#!/usr/bin/env bash
# parse_text.sh — Normalize plain text attachments (EML, TXT, CSV, ICS, etc.)
# Uses python3 for EML and ICS parsing; fallback to raw copy if python3 unavailable.
# USAGE: parse_text.sh <text-file> <normalized-out-dir> <preview-out-dir>

set -euo pipefail

FILE="$1"
NORM="$2"
PREVIEW="$3"

mkdir -p "$NORM" "$PREVIEW"

BASENAME="$(basename "$FILE")"
EXT="${BASENAME##*.}"
[[ "$EXT" == "$BASENAME" ]] && EXT=""
EXT_LC=$(printf '%s' "$EXT" | tr '[:upper:]' '[:lower:]')

echo "  [TEXT] Processing: $BASENAME"

has_py3() { command -v python3 &>/dev/null; }
py3_ok() { python3 "$@" 2>/dev/null; }

# ──────────────────────────────────────────────────────────────────────
# Handler 1: EML (RFC 822 email messages)
# ──────────────────────────────────────────────────────────────────────
handle_eml() {
    if has_py3; then
        echo "    → EML extraction via python3"
        # Markdown output
        py3_ok - "$FILE" "$NORM" <<'PYEOF'
import sys, email
from email import policy
from email.parser import BytesParser

infile = sys.argv[1]
outdir = sys.argv[2]

try:
    with open(infile, 'rb') as f:
        msg = BytesParser(policy=policy.default).parse(f)

    lines = []
    lines.append("# Email: " + str(msg.get('Subject', '(no subject)')))
    lines.append("- **From:** " + str(msg.get('From', 'N/A')))
    lines.append("- **To:** " + str(msg.get('To', 'N/A')))
    lines.append("- **Date:** " + str(msg.get('Date', 'N/A')))
    lines.append("- **Message-ID:** " + str(msg.get('Message-ID', 'N/A')))
    lines.append("")

    body = ""
    is_html = False
    if msg.is_multipart():
        for part in msg.walk():
            ctype = part.get_content_type()
            if ctype == "text/plain":
                body = part.get_content() or ""
                break
            elif ctype == "text/html" and not body:
                body = part.get_content() or ""
                is_html = True
    else:
        ctype = msg.get_content_type()
        body = msg.get_content() or ""
        is_html = (ctype == "text/html")

    lines.append("## Body")
    if is_html:
        lines.append("```html")
    lines.append(body if body else "[no body text]")
    if is_html:
        lines.append("```")
    lines.append("")

    lines.append("## Attachments")
    found = False
    for part in msg.walk():
        disp = part.get_content_disposition() or ""
        if "attachment" in disp:
            found = True
            fn = part.get_filename() or "unnamed"
            ctype = part.get_content_type()
            payload = part.get_payload(decode=True) or b''
            lines.append("- `" + fn + "` (" + ctype + ", " + str(len(payload)) + " bytes)")
    if not found:
        lines.append("_None_")

    with open(outdir + "/content.md", "w") as f:
        f.write("\n".join(lines))
except Exception as e:
    with open(outdir + "/content.md", "w") as f:
        f.write("[email parsing error: " + str(e) + "]\n\n")
        with open(infile, "r", errors="replace") as raw:
            f.write(raw.read())
PYEOF

        # Plain text output
        py3_ok - "$FILE" "$NORM" <<'PYEOF'
import sys, email
from email import policy
from email.parser import BytesParser
infile = sys.argv[1]
outdir = sys.argv[2]
try:
    with open(infile, 'rb') as f:
        msg = BytesParser(policy=policy.default).parse(f)
    out = []
    out.append("From: " + str(msg.get('From', 'N/A')))
    out.append("To: " + str(msg.get('To', 'N/A')))
    out.append("Subject: " + str(msg.get('Subject', 'N/A')))
    out.append("Date: " + str(msg.get('Date', 'N/A')))
    out.append("")
    if msg.is_multipart():
        for part in msg.walk():
            if part.get_content_type() == "text/plain":
                out.append(part.get_content() or "")
                break
    else:
        out.append(msg.get_content() or "")
    with open(outdir + "/content.txt", "w") as f:
        f.write("\n".join(out))
except Exception:
    with open(infile, "r", errors="replace") as src:
        with open(outdir + "/content.txt", "w") as dst:
            dst.write(src.read())
PYEOF
    else
        echo "    (!) python3 unavailable for EML; using raw"
        cp "$FILE" "$NORM/content.txt"
        cp "$FILE" "$NORM/content.md"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Handler 2: ICS (iCalendar files)
# ──────────────────────────────────────────────────────────────────────
handle_ics() {
    if has_py3; then
        echo "    → ICS extraction via python3"
        py3_ok - "$FILE" "$NORM" <<'PYEOF'
import sys
infile = sys.argv[1]
outdir = sys.argv[2]

try:
    with open(infile, "r", errors="replace") as f:
        lines = f.readlines()

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
        elif in_event and ':' in line_s:
            key, val = line_s.split(':', 1)
            if ';' in key:
                key = key.split(';', 1)[0]
            current[key] = val

    out = []
    out.append(f"# Calendar: {sys.argv[1].split('/')[-1]}")
    out.append(f"- **Total events:** {len(events)}")
    out.append("")

    if events:
        for i, ev in enumerate(events, 1):
            out.append(f"## Event {i}")
            out.append(f"- **Summary:** {ev.get('SUMMARY', '(no title)')}")
            out.append(f"- **Start:** {ev.get('DTSTART', 'N/A')}")
            out.append(f"- **End:** {ev.get('DTEND', 'N/A')}")
            out.append(f"- **Location:** {ev.get('LOCATION', 'N/A')}")
            out.append(f"- **Organizer:** {ev.get('ORGANIZER', 'N/A')}")
            desc = ev.get('DESCRIPTION', '')
            if desc:
                out.append(f"- **Description:** {desc}")
            out.append("")
    else:
        out.append("No VEVENT blocks found; raw ICS output:")
        out.append("```")
        out.extend([l.rstrip() for l in lines])
        out.append("```")

    with open(outdir + "/content.md", "w") as f:
        f.write("\n".join(out))
except Exception as e:
    with open(outdir + "/content.md", "w") as f:
        f.write(f"[ICS parsing error: {e}]\n")
        with open(infile, "r", errors="replace") as raw:
            f.write(raw.read())
PYEOF
    else
        cp "$FILE" "$NORM/content.md"
    fi

    # Plain text: just a cleaned-up copy
    if [[ -f "$NORM/content.md" ]]; then
        cp "$NORM/content.md" "$NORM/content.txt"
    else
        cp "$FILE" "$NORM/content.txt"
    fi
}

# ──────────────────────────────────────────────────────────────────────
# Handler 3: All other text files (TXT, CSV, MD, etc.)
# ──────────────────────────────────────────────────────────────────────
handle_plain_text() {
    cp "$FILE" "$NORM/content.txt"
    {
        printf '%s\n' "# Text File: $BASENAME"
        printf '%s\n' ""
        printf '%s\n' '```'
        cat "$NORM/content.txt"
        printf '%s\n' '```'
    } > "$NORM/content.md"
}

# ──────────────────────────────────────────────────────────────────────
# Dispatch
# ──────────────────────────────────────────────────────────────────────
case "$EXT_LC" in
    eml) handle_eml ;;
    ics) handle_ics ;;
    *)   handle_plain_text ;;
esac

echo "    ✓ Text parsed: content.txt, content.md"
