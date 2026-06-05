---
name: thread-workflows
description: Use this skill when listing, searching, summarizing, sharing, renaming, deleting, exporting, or analyzing Perplexity threads; finding previous research; preserving backend_uuid values; or extracting decisions and patterns from conversation history. Triggers on "show my Perplexity threads", "find previous research", "summarize thread", "analyze my threads", "export Perplexity conversation".
version: 1.2.0
---

# Thread Workflows

Work with Perplexity thread history in a token-efficient way.

## Retrieval Strategy

1. Start with `list_threads` and a search term.
2. Fetch only likely matches with `get_thread`.
3. Extract durable identifiers: slug, context UUID, backend UUID, title.
4. Summarize rather than paste raw thread JSON.
5. Preserve follow-up IDs before compaction.

## Summary Template

```markdown
## Thread Summary: <title>
- Slug/context: <ids>
- Topic: <topic>
- Key answers: <bullets>
- Decisions/code patterns: <bullets>
- Sources: <short list>
- Follow-up opportunities: <bullets>
```

## Cross-Thread Analysis

For many threads, use the `thread-analyzer` agent. Compare repeated answers, conflicts, cited sources, and decisions.

## CLI Fallback

```bash
pplx threads list --search "topic" --limit 20
pplx threads get <slug> --raw
pplx threads share <slug>
bash <plugin>/scripts/pplx-summarize.sh <slug>
```

## SDK Usage

```python
from pplx_sdk import PerplexityClient
client = PerplexityClient()
threads = client.list_threads(search_term="topic", limit=20)
for t in threads.threads:
    print(t.title, t.slug)
thread = client.get_thread("<slug>")
print(thread.title, thread.backend_uuid)
client.close()
```

Ask before destructive operations such as delete.
