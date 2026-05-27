---
name: bitwarden-vault
description: |
  Manage Bitwarden vault operations: TOTP generation, credential injection,
  vault search, multi-account switching, and Secrets Manager (bws) integration.

  **Trigger phrases:** "bitwarden", "password", "vault", "TOTP", "2FA",
  "auth code", "inject password", "search vault", "export passwords",
  "switch account", "bws", "secrets manager", "credential", "keychain".
license: Complete terms in LICENSE.txt
---

# Bitwarden Vault Manager

## Keywords

bitwarden, password, vault, credential, secret, TOTP, 2FA, authentication code, inject secrets, env vars, export vault, backup passwords, bw, bws, keychain, auto-auth, api key, secrets manager

## Overview

Use `bw-plugin` — a multi-account Bitwarden CLI wrapper — to perform vault operations without hardcoding credentials. The tool manages three isolated accounts (personal, work, api) with automatic authentication via API key + macOS Keychain or environment variables. Session keys are never persisted to disk.

**Account model:**

| Account | Alias | Server |
|---------|-------|--------|
| personal | `bwp` | vault.bitwarden.com |
| work | `bww` | nodewarden.hmmr.workers.dev |
| api | `bwa` | vault.bitwarden.com |

Target a specific account with `--account <name>` or by invoking the alias (`bwp`, `bww`, `bwa`). The active account is tracked via `active_id` in `~/.config/bw-plugin/accounts.json`.

**Authentication model:**

Credentials are resolved in this priority order:
1. **Environment variables** — `BW_CLIENTID`/`BW_CLIENTSECRET`, or `BWP_CLIENTID`/`BWP_CLIENTSECRET`, etc.
2. **`.env` file** — `~/.config/bw-plugin/.env`
3. **macOS Keychain** — stored via `bw-plugin auth setup`

When any vault operation needs a session, `bw-plugin` automatically:
1. Checks current authentication status
2. If unauthenticated: logs in with API key (bypasses device verification) and unlocks with master password
3. If locked: unlocks with stored master password
4. Returns the session key — all transparently
5. If credentials are missing: prints clear setup instructions with the exact command to run

**Prerequisites:**
- `bw-plugin` binary in PATH (built from `src/` or pre-installed at `~/bin/bw-plugin`)
- `bw` (Bitwarden CLI) and optionally `bws` (Secrets Manager CLI)
- API key credentials stored in macOS Keychain (via `bw-plugin auth setup`) or environment variables
- Master password stored for vault unlock

## Quick Reference

| Task | Command |
|------|---------|
| **Auth Management** | |
| Store API key + password in Keychain | `bw-plugin auth setup` |
| Login with API key | `bw-plugin auth login [account]` |
| Test all accounts auth flow | `bw-plugin auth test` |
| Show stored credentials (masked) | `bw-plugin auth show` |
| Remove stored credentials | `bw-plugin auth clean` |
| Setup Secrets Manager auth | `bw-plugin bws-setup` |
| **Account Management** | |
| Check all accounts status | `bw-plugin` or `bw-plugin status -j` |
| Switch active account | `bw-plugin switch [account]` or `bwp` / `bww` / `bwa` |
| Login to active account | `bw-plugin login` |
| Unlock vault (get session) | `bw-plugin unlock` or `bw-plugin unlock --raw` |
| **Vault Operations** | |
| Search vault items | `bw-plugin search "query"` |
| Search ALL accounts | `bw-plugin search -a "query"` |
| Get TOTP code | `bw-plugin totp "item"` or `bw-plugin totp "item" --copy` |
| Inject credentials into command | `bw-plugin inject "item" -- <command>` |
| Export vault (encrypted) | `bw-plugin export -e -o ~/Backups` |
| Decrypt export | `bw-plugin decrypt <file.enc>` |
| Generate password | `bw-plugin generate --length 32 --uppercase` |
| Run with Secrets Manager | `bw-plugin bws run -- 'npm run dev'` |
| Passthrough to bw CLI | `bw-plugin get password "item"` |
| **Secrets Manager (bws)** | |
| List secrets | `bws secret list` |
| Get secret | `bws secret get <SECRET_ID>` |
| Create secret | `bws secret create <KEY> <VALUE> <PROJECT_ID>` |
| List projects | `bws project list` |
| Inject secrets into command | `bws run -- ./start.sh` |

## Workflow

### 1. First-Time Setup

```bash
# Store API key credentials + master password in macOS Keychain (interactive)
bw-plugin auth setup

# Login to each account (uses API key, no device verification)
bw-plugin auth login

# Verify everything works
bw-plugin auth test
```

**`bw-plugin auth setup` prompts for each account:**
- API Client ID + Secret (required — enables API key login, bypasses device verification)
- Master password (required — for vault unlock)

After setup, **all vault operations auto-authenticate**. No manual login or unlock needed.

### 2. Daily Usage

All vault operations (`search`, `inject`, `totp`, `export`) automatically handle authentication:

```bash
# These work without any manual login/unlock step
bw-plugin search -a "cloudflare"
bw-plugin totp "aws" --copy
bw-plugin inject "cloudflare-api" -- ./deploy.sh
```

The auto-auth flow:
1. Checks `bw status` for the target account
2. If **locked**: unlocks with master password from Keychain or env vars
3. If **unauthenticated**: logs in with API key (bypasses device verification) then unlocks
4. All operations have 30-second timeouts and closed stdin to prevent prompt hangs
5. If credentials are missing: prints clear error with setup command

### 3. Credential Retrieval

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

### 4. TOTP / 2FA Codes

```bash
# Print code
bw-plugin totp "aws"

# Copy to clipboard
bw-plugin totp "aws" --copy
```

TOTP codes are time-sensitive. Retrieve them immediately before the user needs to input them.

### 5. Export and Backup

```bash
# Export with PIN encryption (AES-256-CBC + PBKDF2, 1M iterations)
bw-plugin export -p personal -e -o ~/Backups

# Decrypt later
bw-plugin decrypt ~/Backups/bw-personal-20260101-120000.enc
```

The PIN is set interactively during export. It cannot be recovered if lost.

### 6. Secrets Manager (bws)

The plugin integrates with Bitwarden Secrets Manager for machine-to-machine secrets:

```bash
# Setup bws credentials (interactive)
bw-plugin bws-setup

# List secrets
bws secret list

# Create a secret (need project ID first)
bws project list
bws secret create API_KEY "sk-abc123" <PROJECT_ID> --note "Production"

# Run with secrets injected
bws run -- ./deploy.sh
```

See the `bitwarden-secrets-manager` skill for detailed bws usage.

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

**User:** "Search all my vaults for 'stripe'"
→ Search across all accounts (auto-authenticates as needed):
```bash
bw-plugin search -a "stripe"
```

**User:** "Set up Bitwarden auth for all accounts"
→ Interactive setup flow:
```bash
bw-plugin auth setup
bw-plugin auth login
bw-plugin auth test
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

**User:** "Create a secret in Secrets Manager"
→ Use bws after setup:
```bash
bws project list
bws secret create DATABASE_URL "postgres://..." <PROJECT_ID>
```

## Guidelines

- **Auto-auth behavior.** Vault operations automatically login and unlock using stored credentials. No manual `bw-plugin login` or `bw-plugin unlock` needed after initial `auth setup`.
- **API key login required.** Authentication uses API key credentials (Client ID + Client Secret). This bypasses Bitwarden's device verification entirely.
- **Credential resolution.** Credentials are read from: environment variables → `.env` file → macOS Keychain (set via `auth setup`). All sources work transparently.
- **Setup on missing credentials.** If credentials are not found, the tool prints clear instructions including the exact command to run (`bw-plugin auth setup`) and alternative methods (env vars, .env file).
- **Never persist credentials to files.** Do not write passwords or session keys to files. Prefer `inject` to pass credentials as env vars to child processes.
- **Account targeting.** Use `--account` or the alias (`bwp`/`bww`/`bwa`) when the user specifies an account. Default to the active account otherwise.
- **JSON mode.** Use `-j` / `--json` when parsing output programmatically.
- **TOTP timing.** Retrieve TOTP codes immediately before the user needs them — they expire quickly.
- **Injection safety.** The `inject` command runs the child process with credentials in env vars. Ensure the command is trusted.
- **Export encryption.** Always use `-e` when exporting. The PIN is interactive-only and unrecoverable.
- **Clipboard fallback.** Use `--copy` for TOTP when the user needs to paste. Falls back to printing if no clipboard utility is available.
- **Validate before bulk operations.** Run `bw-plugin validate` before scripts that perform multiple vault operations.
- **No session persistence.** `BW_SESSION` is never written to disk. It is derived on-demand via `bw unlock` using stored credentials.
- **Use passthrough for unlisted operations.** Any `bw` command not explicitly handled is passed through with the active account's context.
- **Secrets Manager separation.** Vault (`bw`) and Secrets Manager (`bws`) are separate systems with separate credentials. Use `bws-setup` for Secrets Manager tokens.
