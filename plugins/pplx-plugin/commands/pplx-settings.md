---
name: pplx-settings
aliases: [perplexity-settings, pplx-config, pplx-account]
description: Audit Perplexity account status, credits, rate limits, memories, tasks, workflows, model availability, and plugin configuration
allowed-tools: [Bash]
argument-hint: "status|models|credits|memories|tasks|workflows|rate-limits|ai-profile|health"
---

# /pplx-settings

Audit Perplexity account and client settings using the available `pplx` client surface.

## Preferred CLI Commands

```bash
pplx status --raw
pplx models --raw
pplx spaces list
pplx threads list --search "" --limit 20
bash <plugin>/scripts/pplx-health.sh --verbose --no-search
```

For detailed account info, use the Python SDK directly:

```python
from pplx_sdk import PerplexityClient
client = PerplexityClient()
print(client.get_user_settings())
print(client.get_rate_limit_status())
print(client.get_ai_profile())
print(client.list_memories())
print(client.list_tasks())
print(client.list_workflows())
client.close()
```

## Workflow

1. Determine the requested settings area.
2. Use the narrowest command; avoid dumping all account data by default.
3. Summarize as a concise table with remediation actions.
4. For destructive actions like memory deletion or task deletion, ask for confirmation.
5. If MCP settings tools are available in the harness, they may be used, but do not assume they exist.

## Audit Output

Include:
- Auth/session health
- Subscription/credits/rate-limit signals if available
- Model discovery status
- Memories/tasks/workflows counts
- Plugin/client path configuration
- Recommended fixes
