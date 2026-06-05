---
name: bitwarden-vault
description: |
  Manage Bitwarden vault operations: TOTP generation, credential injection,
  vault search, and Secrets Manager (bws) integration.

  **Trigger phrases:** "bitwarden", "password", "vault", "TOTP", "2FA",
  "auth code", "inject password", "search vault", "export passwords",
  "bws", "secrets manager", "credential", "keychain".
---

# Bitwarden Vault Manager

## Keywords

bitwarden, password, vault, credential, secret, TOTP, 2FA, authentication code, inject secrets, env vars, export vault, backup passwords, bw, bws, keychain, auto-auth, api key, secrets manager

## Overview

Use the official `bw` (Bitwarden Password Manager CLI) for vault operations. Single account: Personal Premium (`misterme00@icloud.com` on `vault.bitwarden.com`).

**Authentication model:**

When any vault operation is needed, agents first check `bw status`:
1. **Already authenticated** (user logged in manually) → use session as-is, do NOT re-authenticate
2. **Locked** → inline unlock with operation (see pattern below)
3. **Unauthenticated** → login with API key: `bw login --apikey` (uses `BW_CLIENTID` + `BW_CLIENTSECRET`)

**Agent unlock pattern** (each bash call is a fresh shell — BW_SESSION doesn't persist):
```bash
export BW_PASSWORD=$(security find-generic-password -a "bw-master-password" -w) && \
export BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw) && \
unset BW_PASSWORD && \
bw <command>
```

Credentials are resolved from:
1. Environment variables: `BW_CLIENTID`, `BW_CLIENTSECRET`, `BW_PASSWORD`
2. macOS Keychain: `bw-api-client-id`, `bw-api-client-secret`, `bw-master-password`

**Prerequisites:**
- `bw` CLI installed (`/opt/homebrew/bin/bw` or via `brew install bitwarden-cli`)
- API key credentials available (env vars or Keychain)
- Master password available (env var or Keychain) — for vault unlock

## Quick Reference

| Task | Command |
|------|---------|
| **Auth** | |
| Check status | `bw status` |
| Login (API key) | `bw login --apikey` |
| Unlock vault | `export BW_PASSWORD=$(security find-generic-password -a "bw-master-password" -w) && export BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw) && unset BW_PASSWORD && bw <cmd>` |
| Lock vault | `bw lock` |
| Logout | `bw logout` |
| **Vault Operations** | |
| Search items | `bw list items --search "query"` |
| Get password | `bw get password "item"` |
| Get username | `bw get username "item"` |
| Get TOTP | `bw get totp "item"` |
| Get notes | `bw get notes "item"` |
| Get full item JSON | `bw get item "item"` |
| Create item | `bw get template item \| jq ... \| bw encode \| bw create item` |
| Edit item | `bw get item <id> \| jq ... \| bw encode \| bw edit item <id>` |
| Delete item | `bw delete item <id>` |
| Generate password | `bw generate --length 32 --uppercase --lowercase --numbers --special` |
| Export vault | `bw export --format json --output <path>` |
| Sync vault | `bw sync` |
| **Secrets Manager (bws)** | |
| List secrets | `bws secret list` |
| Get secret | `bws secret get <SECRET_ID>` |
| Create secret | `bws secret create <KEY> <VALUE> <PROJECT_ID>` |
| List projects | `bws project list` |
| Inject into command | `bws run -- ./start.sh` |

## Workflow

### 1. Auth Check (Always First)

```bash
bw status
# If "unauthenticated" → bw login --apikey
# If "locked" → bw unlock --passwordenv BW_PASSWORD
# If "authenticated" → proceed
```

### 2. Credential Retrieval

When locked, inline unlock in the same command:

```bash
export BW_PASSWORD=$(security find-generic-password -a "bw-master-password" -w) && \
export BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw) && \
unset BW_PASSWORD && \
bw get password "GitHub"
```

### 3. TOTP / 2FA Codes

```bash
bw get totp "amazon.com"

# Copy to clipboard
bw get totp "amazon.com" | pbcopy
```

### 4. Credential Injection

```bash
# Export to env var for a command
export CLOUDFLARE_TOKEN=$(bw get password "cloudflare-api")
./deploy.sh

# Or with bws for machine-to-machine
bws run -- './deploy.sh'
```

### 5. Export and Backup

```bash
bw export --format json --output ~/Backups/bw-export.json
# With password protection
bw export --format encrypted_json --output ~/Backups/ --password "strong-password"
```

## Examples

**User:** "Get my GitHub password from Bitwarden"
```bash
bw list items --search "github"
bw get password "GitHub"
```

**User:** "I need a TOTP code for AWS"
```bash
bw get totp "aws" | pbcopy
```

**User:** "Search my vault for 'stripe'"
```bash
bw list items --search "stripe"
```

## Guidelines

- **Never re-authenticate unnecessarily.** If the user has manually authenticated (email/pass/2FA), agents must use that session. Only use API key login when fully unauthenticated.
- **API key login is a fallback.** Primary auth path is the user's manual login. API key is for automated recovery.
- **Never persist credentials to files.** Do not write passwords, session keys, or API keys to files. Prefer env vars.
- **TOTP timing.** Retrieve TOTP codes immediately before the user needs them.
- **Validate before bulk operations.** Run `bw status` before scripts that perform multiple vault operations.
- **Secrets Manager separation.** Vault (`bw`) and Secrets Manager (`bws`) are separate systems with separate credentials.
