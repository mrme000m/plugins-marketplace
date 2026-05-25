---
name: bitwarden-secrets-manager
description: Manage Bitwarden Secrets Manager (bws) operations including secret creation, retrieval, project management, and credential injection via the bws CLI. Use when the user needs to interact with Bitwarden Secrets Manager, create secrets, list projects, or inject secrets as environment variables.
license: Complete terms in LICENSE.txt
---

# Bitwarden Secrets Manager

## Keywords

bws, secrets manager, bitwarden secrets, secret create, secret get, secret list, project, access token, service account, inject secrets, env vars, BWS_ACCESS_TOKEN

## Overview

Use `bws` — the Bitwarden Secrets Manager CLI — to manage machine-to-machine secrets separately from the password vault. The Secrets Manager organizes secrets into **projects** and authenticates via **service account access tokens**.

**Prerequisites:**
- `bws` CLI installed (typically at `~/bin/bws`)
- Service account access token stored in macOS Keychain or as `BWS_ACCESS_TOKEN` env var
- Profile configured in `~/.config/bws/config`

**Authentication model:**

1. **macOS Keychain** (primary) — stored via `bw-plugin bws-setup` or `bws-setup.sh`
2. **Environment variable** (fallback) — `BWS_ACCESS_TOKEN`
3. **Profile-based** — `bws -p <profile>` reads `server_base` from `~/.config/bws/config`

Unlike the vault (`bw`), Secrets Manager has no master password. Access is entirely token-based.

## Quick Reference

| Task | Command |
|------|---------|
| **Auth / Setup** | |
| Setup bws credentials interactively | `bw-plugin bws-setup` or `./bws-setup.sh` |
| List secrets (all projects) | `bws secret list` |
| List secrets in project | `bws secret list <PROJECT_ID>` |
| Get a secret | `bws secret get <SECRET_ID>` |
| Create a secret | `bws secret create <KEY> <VALUE> <PROJECT_ID> [--note "desc"]` |
| Edit a secret | `bws secret edit <SECRET_ID> --value "new-val"` |
| Delete secrets | `bws secret delete <SECRET_ID> [<SECRET_ID>...]` |
| List projects | `bws project list` |
| Run with secrets injected | `bws run -- ./start.sh` |
| Use a profile | `bws -p production secret list` |

## Workflow

### 1. First-Time Setup

Store your service account access token securely:

```bash
# Interactive setup (stores in keychain + shell profile)
./bws-setup.sh

# Or set env var directly
export BWS_ACCESS_TOKEN="<your-token>"
```

### 2. Create a Secret

```bash
# 1. Find your project ID
bws project list

# 2. Create a secret in that project
bws secret create DATABASE_URL "postgres://user:pass@host/db" 550e8400-e29b-41d4-a716-446655440000 --note "Production DB"
```

### 3. Retrieve Secrets

```bash
# List all secrets
bws secret list

# Get a specific secret
bws secret get 550e8400-e29b-41d4-a716-446655440000

# Output as env format
bws -o env secret get <SECRET_ID>
```

### 4. Inject into Commands

```bash
# Run a command with secrets as env vars (secrets prefixed with BWS_)
bws run -- ./deploy.sh

# With no inherited env (isolated)
bws run --no-inherit-env -- ./deploy.sh
```

## Output Formats

All `bws` commands support `-o <format>`:

| Format | Use Case |
|--------|----------|
| `json` | Scripting, default |
| `yaml` | Human-readable structured |
| `env` | `KEY=VALUE` format for shell sourcing |
| `table` | Human-readable table |
| `tsv` | Tab-separated for spreadsheets |

## Examples

**User:** "Create a secret called API_KEY in my production project"
→ Find the project ID first, then create:
```bash
bws project list
bws secret create API_KEY "sk-abc123" <PROJECT_ID> --note "OpenAI production key"
```

**User:** "List all my secrets manager secrets"
→ List across all projects:
```bash
bws secret list
```

**User:** "Run my app with secrets from bws"
→ Use the run command:
```bash
bws run -- npm start
```

**User:** "Get my database URL secret as env format"
→ Use env output:
```bash
bws -o env secret get <SECRET_ID>
```

## Guidelines

- **Token security.** The `BWS_ACCESS_TOKEN` grants access to all secrets the service account can read. Store it in macOS Keychain, never commit it.
- **Projects organize secrets.** Every secret belongs to a project. Use `bws project list` to find `PROJECT_ID` before creating secrets.
- **UUIDs for IDs.** Both `PROJECT_ID` and `SECRET_ID` are UUIDs — use `list` commands to discover them.
- **No folders.** Secrets Manager uses projects, not folders. Plan your project structure accordingly.
- **Service account permissions.** A service account must have access to a project to read/write its secrets. Manage this in the Bitwarden web app.
- **Prefer `run` for injection.** Use `bws run -- <command>` instead of writing secrets to files or echoing them.
- **EU users.** If your organization is on the EU server, set `server_base = "https://api.bitwarden.eu"` in the profile config.
- **Clipboard caution.** When displaying secrets, be aware they may appear in shell history. Use `bws run` or redirect to files instead.
