---
name: pplx-space
aliases: [perplexity-space, pplx-spaces]
description: Manage Perplexity Spaces for project knowledge bases, dependency grounding, files, links, skills, and scoped research
allowed-tools: [mcp__perplexity__list_spaces, mcp__perplexity__get_space, mcp__perplexity__create_space, mcp__perplexity__edit_space, mcp__perplexity__delete_space, mcp__perplexity__upload_file_to_space, mcp__perplexity__list_space_files, mcp__perplexity__delete_space_files, mcp__perplexity__get_upload_status, mcp__perplexity__list_space_threads, mcp__perplexity__search_in_space, Bash, Read]
argument-hint: "list|create|upload|files|search|links|skills|audit"
---

# /pplx-space

Operate Perplexity Spaces as persistent, project-scoped knowledge bases.

## Operations

- **list**: show available Spaces with title, slug, UUID, owner/shared status, and last activity.
- **create**: ask for title, description, instructions, access level, and web-search default. Default to private.
- **upload**: upload local files or selected context. Prefer MCP upload; fall back to `scripts/pplx-upload.sh` or the CLI.
- **files**: list Space files, status, failed uploads, and stale candidates.
- **search**: run scoped search against a Space before broad web search.
- **links**: manage focused web links/domains through the CLI when MCP lacks coverage.
- **skills**: upload or audit Space custom skills through the CLI when needed.
- **audit**: inspect instructions, files, duplicate docs, failed uploads, and whether manifests are current.

## Best Practices

1. One primary Space per project or domain.
2. Upload dependency manifests before API docs so answers are version-grounded.
3. Use clear instructions: tell Perplexity when to prefer uploaded files and when to use web.
4. Verify upload status before relying on new files.
5. Do not upload secrets, `.env`, private keys, or credential exports.

## CLI Fallback

```bash
pplx spaces list
pplx spaces get <slug>
pplx spaces create --title "Project Docs" --description "..." --instructions "..."
pplx spaces upload <space-uuid> <file>
pplx spaces search <space-uuid> "query" --mode auto
bash <plugin>/scripts/pplx-upload.sh <space-slug-or-uuid> <file> [--by-uuid]
```

## SDK Convenience Methods

```python
from pplx_sdk import PerplexityClient
client = PerplexityClient()
client.upload_text_to_space(uuid, "notes.md", "# My Notes")
client.upload_path_to_space(uuid, "/path/to/file.md")
client.close()
```

Always confirm before deleting Spaces or files.
