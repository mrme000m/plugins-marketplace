---
name: bitwarden-secrets
description: |
  Full CRUD management of Bitwarden Secrets Manager projects, secrets,
  and access tokens via the `bws` CLI.

  **Trigger phrases:** "create secret", "edit secret", "delete secret",
  "list secrets", "list projects", "bws create", "bws edit", "bws delete",
  "bws list", "bws get", "bws run", "inject env", "machine account",
  "access token", "Secrets Manager", "bws setup", "bws secret",
  "bws project".
---

# Bitwarden Secrets Manager CLI

## Keywords

bws, secrets-manager, secret, project, machine-account, access-token, inject, environment variable, key-value, credential, API key, token

## Overview

Use the `bws` CLI to manage secrets in Bitwarden Secrets Manager. Secrets are organized into projects. Access is controlled by machine account permissions (set in the Bitwarden web UI).

**Authentication:**
- Environment variable: `export BWS_ACCESS_TOKEN=<token>`
- Per-command flag: `bws secret list --access-token <token>`

**Rate limits:** Many separate sessions from the same IP may trigger rate limits. Reuse the same access token where possible.

**Scope of access:** A machine account's permissions determine what actions are available. Read-only accounts can list and get secrets but cannot create, edit, or delete.

## Quick Reference

| Task | Command |
|------|---------|
| Authenticate (env) | `export BWS_ACCESS_TOKEN=<token>` |
| Authenticate (inline) | `bws <cmd> --access-token <token>` |
| List secrets | `bws secret list` |
| Get secret by ID | `bws secret get <SECRET_ID>` |
| Get secret by key | `bws secret list \| jq '.[] \| select(.key=="KEY")'` |
| Create secret | `bws secret create <KEY> <VALUE> <PROJECT_ID> [--note "..."]` |
| Edit secret | `bws secret edit <SECRET_ID> [--key <KEY>] [--value <VALUE>] [--note "..."] [--project-id <PID>]` |
| Delete secret(s) | `bws secret delete <SECRET_ID> [<SECRET_ID> ...]` |
| List projects | `bws project list` |
| Get project | `bws project get <PROJECT_ID>` |
| Create project | `bws project create "<NAME>"` |
| Edit project | `bws project edit <PROJECT_ID> --name "<NEW_NAME>"` |
| Delete project(s) | `bws project delete <PROJECT_ID> [<PROJECT_ID> ...]` |
| Inject secrets into command | `bws run -- '<command>'` |
| Inject project-scoped secrets | `bws run --project-id <PID> -- '<command>'` |
| List as environment | `bws secret list --output environment` |
| Show version | `bws --version` |

## Output Formats

Use `--output <format>` (or `-o`) with any command:

| Format | Description |
|--------|-------------|
| `json` | Default. JSON array of objects |
| `yaml` | YAML format |
| `env` / `environment` | `KEY="value"` format |
| `table` | Human-readable table |
| `none` | No output (for scripts) |

## Secret Object Structure

```json
{
  "object": "secret",
  "id": "be8e0ad8-d545-4017-a55a-b02f014d4158",
  "organizationId": "10e8cbfa-7bd2-4361-bd6f-b02e013f9c41",
  "projectId": "e325ea69-a3ab-4dff-836f-b02e013fe530",
  "key": "SES_KEY",
  "value": "0.982492bc-7f37-4475-9e60",
  "note": "API Key for AWS SES",
  "creationDate": "2023-06-28T20:13:20.643567Z",
  "revisionDate": "2023-06-28T20:45:37.46232Z"
}
```

## Project Object Structure

```json
{
  "object": "project",
  "id": "1c80965c-acb3-486e-ac24-b03000dc7318",
  "organizationId": "10e8cbfa-7bd2-4361-bd6f-b02e013f9c41",
  "name": "My project",
  "creationDate": "2023-06-29T13:22:37.942559Z",
  "revisionDate": "2023-06-29T13:22:37.942559Z"
}
```

## Workflow: Secrets

### List All Secrets

```bash
bws secret list
```

### List Secrets as Environment Variables

```bash
bws secret list --output environment
# or
bws secret list -o env
```

### Filter Secrets by Project

```bash
bws secret list | jq '.[] | select(.projectId=="PROJECT_ID")'
```

### Get Secret by ID

```bash
bws secret get be8e0ad8-d545-4017-a55a-b02f014d4158
```

### Find Secret by Key Name

```bash
bws secret list | jq '.[] | select(.key=="DATABASE_URL")'
```

### Create Secret

```bash
bws secret create DATABASE_URL "postgres://user:pass@host/db" PROJECT_ID --note "Production DB"
```

### Create Secret (with explicit project flag)

```bash
bws secret create API_KEY "sk_live_123" PROJECT_ID --note "Stripe API key"
```

### Edit Secret Value

```bash
bws secret edit SECRET_ID --value "new-value-here"
```

### Edit Secret Key

```bash
bws secret edit SECRET_ID --key "NEW_KEY_NAME"
```

### Edit Secret Note

```bash
bws secret edit SECRET_ID --note "Updated description"
```

### Edit Multiple Fields

```bash
bws secret edit SECRET_ID --key NEW_KEY --value "new-value" --note "new note"
```

### Move Secret to Different Project

```bash
bws secret edit SECRET_ID --project-id NEW_PROJECT_ID
```

### Delete Single Secret

```bash
bws secret delete be8e0ad8-d545-4017-a55a-b02f014d4158
```

### Delete Multiple Secrets

```bash
bws secret delete ID1 ID2 ID3
```

## Workflow: Projects

### List Projects

```bash
bws project list
```

### Get Project Details

```bash
bws project get PROJECT_ID
```

### Create Project

```bash
bws project create "My New Project"
```

### Edit Project Name

```bash
bws project edit PROJECT_ID --name "Updated Project Name"
```

### Delete Single Project

```bash
bws project delete PROJECT_ID
```

### Delete Multiple Projects

```bash
bws project delete ID1 ID2
```

## Workflow: Secret Injection

### Inject All Secrets as Environment Variables

```bash
bws run -- 'npm run start'
```

Secrets are injected as env vars with their key names. The command runs in a child process with secrets available.

### Inject Project-Scoped Secrets Only

```bash
bws run --project-id PROJECT_ID -- 'npm run start'
```

### Inject and Use in Shell

```bash
bws run -- 'echo $DATABASE_URL'
```

### Inject Multiple Commands

```bash
bws run -- 'docker compose up -d && ./run-tests.sh; docker compose down'
```

### Inject with Custom Shell

```bash
bws run --shell /bin/zsh -- 'echo $API_KEY'
```

### Export to .env File

```bash
bws secret list --output environment > .env
```

## Workflow: Authentication

### Using bw-plugin (Recommended)

```bash
# Link a machine account to an organization
bw-plugin sm-link [account]

# Set up bws credentials interactively
bw-plugin bws-setup

# Run bws commands via bw-plugin wrapper
bw-plugin bws secret list
bw-plugin bws run -- 'npm start'
```

### Direct bws Usage

```bash
# Set access token for session
export BWS_ACCESS_TOKEN="0.48c78342-1635-48a6-accd-afbe01336365.C0tMmQqHnAp1h0gL8bngprlPOYutt0:B3h5D+YgLvFiQhWkIq6Bow=="

# Use per-command
bws secret list --access-token "$BWS_ACCESS_TOKEN"
```

## Examples

**User:** "List all my secrets"

```bash
bws secret list | jq '.[] | {key, id, projectId}'
```

**User:** "Create a new secret called API_KEY"

```bash
# First get the project ID
PROJECT_ID=$(bws project list | jq -r '.[0].id')
bws secret create API_KEY "sk_live_12345" "$PROJECT_ID" --note "Stripe production key"
```

**User:** "Update a secret value"

```bash
SECRET_ID=$(bws secret list | jq -r '.[] | select(.key=="API_KEY") | .id')
bws secret edit "$SECRET_ID" --value "sk_live_newvalue"
```

**User:** "Delete a secret"

```bash
bws secret delete SECRET_ID
```

**User:** "Create a new project"

```bash
bws project create "Production Infrastructure"
```

**User:** "List all projects"

```bash
bws project list | jq '.[] | {id, name}'
```

**User:** "Run my app with secrets injected"

```bash
bws run -- 'npm run dev'
```

**User:** "Get secrets from a specific project only"

```bash
PROJECT_ID=$(bws project list | jq -r '.[] | select(.name=="Production") | .id')
bws run --project-id "$PROJECT_ID" -- 'npm run start'
```

**User:** "Export secrets to a .env file"

```bash
bws secret list --output environment > .env
```

**User:** "Find a secret by its key name"

```bash
bws secret list | jq '.[] | select(.key=="DATABASE_URL")'
```

**User:** "Show all secrets in a table format"

```bash
bws secret list --output table
```

**User:** "Move a secret to a different project"

```bash
NEW_PROJECT_ID=$(bws project list | jq -r '.[] | select(.name=="New Project") | .id')
SECRET_ID=$(bws secret list | jq -r '.[] | select(.key=="API_KEY") | .id')
bws secret edit "$SECRET_ID" --project-id "$NEW_PROJECT_ID"
```

## Guidelines

- **Machine account permissions matter.** A read-only machine account cannot create, edit, or delete secrets/projects. Verify permissions in the Bitwarden web UI before attempting writes.
- **Use `bws run` for injection.** Never write secrets to files or echo them to output. Use `bws run -- '<command>'` to inject secrets as environment variables.
- **Quote commands for run.** Always wrap commands in single quotes when using `bws run` to prevent shell interpretation of special characters (`$`, `&`, `;`, `"`).
- **Handle rate limits.** Reuse access tokens across commands rather than creating many short-lived sessions from the same IP.
- **Project scoping.** Use `--project-id` with `bws run` to limit which secrets are injected.
- **Output formats.** Use `--output environment` for `.env` file generation. Use `--output json` (default) for programmatic processing with `jq`.
- **Secret IDs are UUIDs.** The `edit` and `delete` commands require exact secret IDs, not key names. Resolve via `bws secret list` first.
- **Notes are optional but recommended.** Use `--note` when creating secrets to document their purpose.
- **Delete is permanent.** Unlike `bw` vault items, deleted secrets in Secrets Manager are immediately and irreversibly removed.
- **Access tokens are scoped.** An access token only grants access to secrets/projects the associated machine account has permissions for.
- **Prefer `bw-plugin bws` wrapper.** When available, use `bw-plugin bws <command>` for automatic access token management and account isolation.
- **Environment variable names.** Be cautious with secret key names that match sensitive environment variables (e.g., `PATH`, `SHELL`). The `bws run` command injects secrets by key name, which can overwrite critical system variables.
