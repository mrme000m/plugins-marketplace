---
name: bitwarden-vault
description: Manage Bitwarden vault operations including password retrieval, TOTP generation, credential injection into commands, vault search, and multi-account switching. Use when the user needs to interact with their Bitwarden password vault.
license: Complete terms in LICENSE.txt
---

# Bitwarden Vault Manager

## Keywords

bitwarden, password, vault, credential, secret, TOTP, 2FA, authentication code, inject secrets, env vars, export vault, backup passwords, bw, bwc, bws

## Overview

Use `bw-plugin` — a multi-account Bitwarden CLI wrapper — to perform vault operations without hardcoding credentials. The tool manages three isolated accounts (personal, work, api) with on-demand unlock via environment variables. Session keys are never persisted to disk.

**Account model:**

| Account | Alias | Password Env | Server |
|---------|-------|-------------|--------|
| personal | `bwp` | `BWP_PASSWORD` | vault.bitwarden.com |
| work | `bww` | `BWW_PASSWORD` | nodewarden.hmmr.workers.dev |
| api | `bwa` | `BWA_PASSWORD` | vault.bitwarden.com |

Target a specific account with `--account <name>` or by invoking the alias (`bwp`, `bww`, `bwa`). The active account is tracked in `~/.config/bw-plugin/state.json`.

**Prerequisites:**
- `bw-plugin` binary in PATH (built from `src/` or pre-installed at `~/bin/bw-plugin`)
- `bw` (Bitwarden CLI) and optionally `bws` (Secrets Manager CLI)
- Account password env vars set: `BWP_PASSWORD`, `BWW_PASSWORD`, `BWA_PASSWORD`

## Quick Reference

| Task | Command |
|------|---------|
| Check all accounts status | `bw-plugin` or `bw-plugin status -j` |
| Switch active account | `bw-plugin switch [account]` or `bwp` / `bww` / `bwa` |
| Login to active account | `bw-plugin login` or `bw-plugin login --apikey` |
| Unlock vault (get session) | `bw-plugin unlock` or `bw-plugin unlock --raw` |
| Search vault items | `bw-plugin search "query"` |
| Search ALL accounts | `bw-plugin search -a "query"` |
| Get TOTP code | `bw-plugin totp "item"` or `bw-plugin totp "item" --copy` |
| Inject credentials into command | `bw-plugin inject "item" -- <command>` |
| Export vault (encrypted) | `bw-plugin export -e -o ~/Backups` |
| Decrypt export | `bw-plugin decrypt <file.enc>` |
| Generate password | `bw-plugin generate --length 32 --uppercase` |
| Run with Secrets Manager | `bw-plugin bws run -- 'npm run dev'` |
| Passthrough to bw CLI | `bw-plugin get password "item"` |

## Workflow

### 1. Authentication

Before any vault operation, ensure the target account is accessible:

```bash
# Check current state
bw-plugin status -j

# Switch if needed
bw-plugin switch work

# Login (only needed once per account)
bw-plugin login

# Unlock to get a session key
export BW_SESSION=$(bw-plugin unlock --raw)
```

Vault operations (`search`, `inject`, `totp`, `export`) auto-unlock on-demand when the password env var is set. No manual unlock is needed in that case.

### 2. Credential Retrieval

**For viewing or copying:**
```bash
# Search then retrieve
bw-plugin search "github"
bw-plugin get password "GitHub"
```

**For injecting into commands:**
```bash
# Credentials become env vars in the child process
bw-plugin inject "cloudflare-api" -- ./deploy.sh
# Injects: BW_USERNAME, BW_PASSWORD, BW_ITEM_NAME, BW_ITEM_URL, BW_<custom_fields>
```

Always prefer `inject` over writing credentials to files or echoing them to output.

### 3. TOTP / 2FA Codes

```bash
# Print code
bw-plugin totp "aws"

# Copy to clipboard
bw-plugin totp "aws" --copy
```

TOTP codes are time-sensitive. Retrieve them immediately before the user needs to input them.

### 4. Export and Backup

```bash
# Export with PIN encryption (AES-256-CBC + PBKDF2, 1M iterations)
bw-plugin export -p personal -e -o ~/Backups

# Decrypt later
bw-plugin decrypt ~/Backups/bw-personal-20260101-120000.enc
```

The PIN is set interactively during export. It cannot be recovered if lost.

## Examples

**User:** "Get my GitHub password from Bitwarden"
→ Search first to confirm the item name, then retrieve:
```bash
bw-plugin search "github"
bw-plugin get password "GitHub"
```

**User:** "I need a TOTP code for AWS"
→ Retrieve the code, copy to clipboard if needed:
```bash
bw-plugin totp "aws" --copy
```

**User:** "Inject my Cloudflare API credentials into this deploy script"
→ Use inject to pass credentials as env vars:
```bash
bw-plugin inject "cloudflare-api" -- ./deploy.sh
```

**User:** "Switch to my work Bitwarden account"
→ Switch to the work account:
```bash
bw-plugin switch work
# or
bww
```

**User:** "Export and encrypt my personal vault"
→ Export with encryption:
```bash
bw-plugin export -p personal -e -o ~/Backups
```

**User:** "Search all my vaults for 'stripe'"
→ Search across all accounts:
```bash
bw-plugin search -a "stripe"
```

## Guidelines

- **Never persist credentials.** Do not write passwords or session keys to files. Prefer `inject` to pass credentials as env vars to child processes.
- **Auto-unlock behavior.** Vault operations auto-unlock when the password env var is set. If unset, prompt the user to set it or run `bw-plugin unlock`.
- **Account targeting.** Use `--account` or the alias (`bwp`/`bww`/`bwa`) when the user specifies an account. Default to the active account otherwise.
- **JSON mode.** Use `-j` / `--json` when parsing output programmatically.
- **TOTP timing.** Retrieve TOTP codes immediately before the user needs them — they expire quickly.
- **Injection safety.** The `inject` command runs the child process with credentials in env vars. Ensure the command is trusted.
- **Export encryption.** Always use `-e` when exporting. The PIN is interactive-only and unrecoverable.
- **Clipboard fallback.** Use `--copy` for TOTP when the user needs to paste. Falls back to printing if no clipboard utility is available.
- **Validate before bulk operations.** Run `bw-plugin validate` before scripts that perform multiple vault operations.
- **No session persistence.** `BW_SESSION` is never written to disk. It is derived on-demand via `bw unlock` using the password env var.
- **Use passthrough for unlisted operations.** Any `bw` command not explicitly handled is passed through with the active account's context.