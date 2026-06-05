---
name: perplexity-settings
description: Use this skill when auditing Perplexity account/client settings, checking auth, BWS cookie loading, credits, rate limits, dynamic models, memories, tasks, workflows, AI profile, Space memory config, or plugin setup. Triggers on "Perplexity settings", "pplx status", "credits", "rate limits", "memories", "workflows", "models", "auth check", "BWS_ACCESS_TOKEN", "perplexity-cookies".
version: 1.2.0
---

# Perplexity Settings

Audit account and client state through the actual `pplx` client commands available in this repository.

## Auth/Cookie Model

Cookie loading order:

1. `PERPLEXITY_COOKIES_PATH` JSON file override.
2. **Preferred:** Bitwarden Secrets Manager using `bitwarden-sdk`, `BWS_ACCESS_TOKEN`, project `pplx`, secret `perplexity-cookies`.
3. Legacy `bw` CLI Secure Note named `perplexity.ai`.

For BWS setup:

```bash
python scripts/setup_bws_secret.py create-token
python scripts/setup_bws_secret.py setup-cookies /path/to/cookies.json
python scripts/setup_bws_secret.py show
```

## Narrow Commands

Use the narrowest command for the requested area:

```bash
pplx status --raw
pplx models --raw
pplx spaces list
pplx threads list --search ""
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

## Health Diagnostics

```bash
bash <plugin>/scripts/pplx-health.sh --verbose --no-search
```

Run live search diagnostics only when the user asks or auth/connectivity must be verified.

## Reporting

Summarize, do not dump raw JSON. Include:
- auth/client status
- BWS token/secret presence, without exposing secret values
- dynamic model discovery state
- rate/credit signals
- memory/task/workflow counts
- recommended cleanup or setup actions

Ask before deleting memories, tasks, Spaces, or files.
