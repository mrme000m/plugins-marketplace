# pplx-plugin

Marketplace-ready Perplexity AI toolkit for Claude Code, OpenCode, Codex, and shell-native agent harnesses.

**Version:** 1.2.0

## What It Provides

- **Grounded research** with Perplexity mode discipline and follow-up chaining
- **Space management** for persistent project knowledge bases
- **Thread workflows** for search, summary, export, and cross-thread analysis
- **Dependency grounding** to avoid stale external API patterns
- **Portable CLI fallback** backed by the local `pplx` Python client (`pplx_sdk.cli`)
- **BWS-first cookie auth diagnostics** via `bitwarden-sdk`, with legacy `bw` fallback
- **Hooks** for readiness checks, cost guardrails, dependency-change nudges, and compaction preservation
- **Subagents** for knowledge curation, thread analysis, and stale-doc risk scanning

## Plugin Layout

```text
.claude-plugin/plugin.json   # marketplace/plugin metadata
commands/                    # slash command instructions
skills/                      # auto-activating reusable guidance
agents/                      # focused subagent prompts
hooks/hooks.json             # lifecycle and guardrail hooks
scripts/                     # portable helpers for any shell-capable harness
```

## Requirements

- Python 3.10+
- `pplx-sdk` package (this repo or a compatible checkout)
- Python dependencies: `curl-cffi`, `bitwarden-sdk`, `python-dotenv`
- Preferred auth: `BWS_ACCESS_TOKEN` for Bitwarden Secrets Manager project `pplx`, secret `perplexity-cookies`
- Optional fallback: Bitwarden CLI (`bw`) Secure Note named `perplexity.ai`
- Optional: `jq` for richer shell diagnostics

## Cookie/Auth Setup

The `pplx` client loads cookies in this order:

1. `PERPLEXITY_COOKIES_PATH=/path/to/cookies.json`
2. **Preferred:** Bitwarden Secrets Manager via `bitwarden-sdk`, `BWS_ACCESS_TOKEN`, project `pplx`, secret `perplexity-cookies`
3. Legacy Bitwarden CLI fallback: Secure Note named `perplexity.ai`

In dev environments where `bw` CLI is already configured, the plugin can populate missing `.env` keys from a Bitwarden item named `pplx-env`:

```bash
bash pplx-plugin/scripts/pplx-bw-env.sh --item pplx-env --env /path/to/pplx/.env --dry-run
bash pplx-plugin/scripts/pplx-bw-env.sh --item pplx-env --env /path/to/pplx/.env
```

The item may be a Secure Note with `KEY=VALUE` lines or custom fields for `BITWARDEN_CLIENT_ID`, `BITWARDEN_CLIENT_SECRET`, `BWS_ACCESS_TOKEN`, and `PERPLEXITY_COOKIES_PATH`.

BWS bootstrap from the pplx client checkout:

```bash
python scripts/setup_bws_secret.py create-token
python scripts/setup_bws_secret.py setup-cookies /path/to/cookies.json
python scripts/setup_bws_secret.py show
```

The cookie JSON should be an object, for example:

```json
{
  "__Secure-next-auth.session-token": "...",
  "pplx.session-id": "...",
  "cf_clearance": "..."
}
```

## Setup

```bash
bash pplx-plugin/scripts/pplx-setup.sh --print-env
bash pplx-plugin/scripts/pplx-health.sh --verbose --no-search
```

If the client is not installed:

```bash
python3 -m pip install -e .
# or point helpers at a separate checkout
export PPLX_REPO_DIR=/path/to/pplx-sdk
```

## Claude Code Usage

Load the plugin directory with your Claude Code plugin mechanism, then use:

| Command | Purpose |
|---|---|
| `/pplx-research` | Grounded web/Space research with mode guardrails |
| `/pplx-space` | Create, audit, upload to, and search Spaces |
| `/pplx-threads` | Search, summarize, share, and analyze threads |
| `/pplx-upload` | Safely upload docs/manifests to Spaces |
| `/pplx-settings` | Audit account/client/BWS state |
| `/pplx-bws-setup` | Configure/verify BWS SDK cookie auth |
| `/pplx-cli-check` | Diagnose local CLI/auth/model discovery |

## Shell Harness Usage

Any shell-capable agent can use:

```bash
pplx search "query" --mode auto
pplx spaces list
pplx threads list --search "topic"
bash pplx-plugin/scripts/pplx-upload.sh <space> <file>
bash pplx-plugin/scripts/pplx-summarize.sh <thread-slug>
```

## SDK Usage (Python)

```python
from pplx_sdk import PerplexityClient

client = PerplexityClient()
result = client.search("What is Rust?")
print(result.answer)
print(result.backend_uuid)  # for follow-ups
print(result.related_queries)

# Spaces
spaces = client.list_spaces()
for s in spaces.spaces:
    print(s.title, s.slug)

# Upload text or files
client.upload_text_to_space(uuid, "notes.md", "# My Notes")
client.upload_path_to_space(uuid, "/path/to/file.md")

client.close()
```

## Safety Defaults

- Default to free `auto` mode.
- Ask before paid modes unless the user explicitly requested them.
- Do not upload secrets (`.env`, keys, cookies, vault exports).
- Never print cookie values or BWS tokens; report only validation status.
- Prefer local code inspection over web search for repo-specific behavior.
- Preserve `backend_uuid`, Space UUIDs/slugs, and thread slugs before compaction.

## Development Notes

The plugin follows modern Claude Code plugin conventions: lean command files, progressive-disclosure skills, portable helper scripts, no hardcoded development paths, explicit safety guardrails, and CLI fallback for non-MCP harnesses.
