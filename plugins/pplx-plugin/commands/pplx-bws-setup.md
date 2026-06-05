---
name: pplx-bws-setup
aliases: [pplx-bws, pplx-cookies, pplx-auth-setup]
description: Configure and verify Bitwarden Secrets Manager cookie auth for PPLX using bitwarden-sdk, BWS_ACCESS_TOKEN, project pplx, and secret perplexity-cookies
allowed-tools: [Bash, Read]
argument-hint: "status|create-token|setup-cookies <cookies.json>|show|health"
---

# /pplx-bws-setup

Configure and verify the preferred PPLX cookie flow: Bitwarden Secrets Manager through `bitwarden-sdk`.

## Cookie Resolution Order

The `pplx` client resolves cookies in this order:

1. `PERPLEXITY_COOKIES_PATH=/path/to/cookies.json`
2. **Preferred:** BWS SDK using `BWS_ACCESS_TOKEN`, project `pplx`, secret `perplexity-cookies`
3. Legacy Bitwarden CLI Secure Note named `perplexity.ai`

## Safety Rules

- Never print `BWS_ACCESS_TOKEN`, `BITWARDEN_CLIENT_SECRET`, cookie values, or raw `.env` contents.
- Report only whether required variables and secrets exist.
- Ask before overwriting an existing BWS secret unless the user explicitly requested setup/update.
- Use `--no-search` health checks unless the user requests live connectivity.

## Commands

If this is a dev environment with `bw` already configured and `.env` is missing or incomplete, populate missing `.env` keys from a Bitwarden item first:

```bash
bash <plugin>/scripts/pplx-bw-env.sh --item pplx-env --env /path/to/pplx/.env --dry-run
bash <plugin>/scripts/pplx-bw-env.sh --item pplx-env --env /path/to/pplx/.env
```

The Bitwarden item can be a Secure Note with `KEY=VALUE` lines or custom fields named `BITWARDEN_CLIENT_ID`, `BITWARDEN_CLIENT_SECRET`, `BWS_ACCESS_TOKEN`, or `PERPLEXITY_COOKIES_PATH`.

From the pplx client checkout:

```bash
# Check current BWS project/secret status
python scripts/setup_bws_secret.py show

# Create or refresh BWS service account token into .env
python scripts/setup_bws_secret.py create-token

# Upload browser-exported cookies JSON into BWS secret 'perplexity-cookies'
python scripts/setup_bws_secret.py setup-cookies /path/to/cookies.json

# Verify plugin/client auth without live search
bash <plugin>/scripts/pplx-health.sh --verbose --no-search
```

## Workflow

1. Determine the pplx client checkout. Prefer `PPLX_REPO_DIR` if set.
2. Confirm `.env` exists in the client checkout and contains `BITWARDEN_CLIENT_ID` and `BITWARDEN_CLIENT_SECRET` or `BWS_ACCESS_TOKEN`.
3. If `BWS_ACCESS_TOKEN` exists, run `python scripts/setup_bws_secret.py show` to verify project `pplx` and secret `perplexity-cookies`.
4. If only client ID/secret exist, run `python scripts/setup_bws_secret.py create-token`, then `show`.
5. If the BWS secret is missing, ask for or use the user-provided cookies JSON path and run `setup-cookies`.
6. Finish with `pplx-health.sh --verbose --no-search`; run live search only if requested.

## Expected Success

```text
Project 'pplx' ID: <redacted/id>
Secret 'perplexity-cookies' ID: <redacted/id>
Valid JSON with <n> cookie key(s).
```

Do not include secret values in the final response.
