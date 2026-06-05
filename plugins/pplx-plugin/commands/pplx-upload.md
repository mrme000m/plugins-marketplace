---
name: pplx-upload
aliases: [pplx-upload-file, pplx-space-upload]
description: Upload project files, manifests, docs, or generated summaries to a Perplexity Space safely
allowed-tools: [mcp__perplexity__list_spaces, mcp__perplexity__upload_file_to_space, mcp__perplexity__get_upload_status, mcp__perplexity__search_in_space, Bash, Read]
argument-hint: "<file> to <space> [--test-query <query>]"
---

# /pplx-upload

Upload useful context to a Perplexity Space while avoiding secret leakage.

## Safety Checklist

Before uploading, inspect the file path and content class. Do **not** upload:
- `.env`, private keys, cookies, tokens, password exports, or vault data
- large binary artifacts unless the user explicitly requested it
- private customer data or credentials

Good upload candidates:
- `README`, architecture docs, API specs, schemas
- dependency manifests and lockfiles
- migration notes and changelogs
- concise research summaries

## Workflow

1. Identify the target file and target Space.
2. If no Space is specified, list Spaces and ask the user to choose.
3. Prefer MCP upload with `collection_uuid`, `filename`, and text content.
4. Verify upload status with `get_upload_status`.
5. Optionally run a small `search_in_space` query to confirm retrieval.

## CLI Fallback

```bash
bash <plugin>/scripts/pplx-upload.sh <space-slug-or-uuid> <file> [--by-uuid]
pplx spaces upload <space-uuid> <file>
pplx spaces files <space-uuid>
```

## SDK Convenience Methods

```python
from pplx_sdk import PerplexityClient
client = PerplexityClient()

# Upload text directly
client.upload_text_to_space(uuid, "notes.md", "# My Notes\n...")

# Upload from file path
client.upload_path_to_space(uuid, "/path/to/file.md")

client.close()
```

Confirm before uploading sensitive or unusually large files.
