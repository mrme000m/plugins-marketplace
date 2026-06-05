---
name: pplx-cli-check
aliases: [pplx-health, pplx-diagnose, perplexity-health]
description: Verify PPLX CLI health, BWS/bitwarden-sdk authentication, valid cookie loading, dynamic model discovery, and optional live search connectivity
allowed-tools: [Bash, Read]
argument-hint: "[--verbose] [--no-search]"
---

# /pplx-cli-check

Diagnose the local `pplx` client, BWS SDK cookie flow, and this plugin's portable helper scripts.

## Workflow

1. Locate the plugin directory from the loaded command path when possible.
2. Run the portable health helper:
   ```bash
   bash pplx-plugin/scripts/pplx-health.sh --verbose --no-search
   ```
   Use the actual plugin path if it is installed elsewhere.
3. If the user explicitly wants an end-to-end test, run without `--no-search`.
4. Summarize results as a checklist:
   - Python 3.10+
   - `pplx` executable or importable Python package (`pplx_sdk`)
   - `bitwarden_sdk`, `curl_cffi`, and `dotenv`
   - BWS setup: `BWS_ACCESS_TOKEN`, project `pplx`, secret `perplexity-cookies`
   - actual cookie loader resolution
   - legacy `bw` fallback only if BWS is not configured
   - dynamic model discovery
   - live search connectivity, if tested
5. Provide concrete remediation commands for failures.

## BWS Cookie Setup

From the pplx client checkout:

```bash
python scripts/setup_bws_secret.py create-token
python scripts/setup_bws_secret.py setup-cookies /path/to/cookies.json
python scripts/setup_bws_secret.py show
```

The client resolves cookies in order:

1. `PERPLEXITY_COOKIES_PATH`
2. BWS SDK secret `perplexity-cookies` in project `pplx`
3. legacy `bw` Secure Note `perplexity.ai`

## Notes

- The helper resolves paths relative to its own location; avoid hardcoding `/Volumes/...` or `~/.claude/...` paths.
- Live search can consume quota. Prefer `--no-search` for readiness checks.
- Never print cookie values or BWS tokens; report only presence and validation status.
