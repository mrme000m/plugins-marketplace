# Premium Account + Secrets Manager Setup Flow

## Goal
Streamlined interactive flow to add a premium Bitwarden account, store API key credentials securely, and generate/link a Secrets Manager access token.

## Account Model

```go
type Account struct {
    ID           string              `json:"id"`
    Name         string              `json:"name"`
    Email        string              `json:"email"`
    Server       string              `json:"server"`
    ServerType   string              `json:"server_type"`
    Plan         AccountPlan         `json:"plan"`
    Capabilities AccountCapabilities `json:"capabilities"`
    Org          *AccountOrg         `json:"org,omitempty"`
    Tags         []string            `json:"tags,omitempty"`
    Notes        string              `json:"notes,omitempty"`
    EnvPrefix    string              `json:"env_prefix,omitempty"`
}
```

## Flow: `bw-plugin account add`

### Phase 1: Account Identity
```
?  Account name (label): Personal Premium
?  Email: misterme00@icloud.com
?  Server URL [https://vault.bitwarden.com]:
?  Server type: 1) cloud (US) | 2) eu | 3) self-hosted | 4) custom [1]:
?  Plan: 1) free | 2) premium | 3) families | 4) teams | 5) enterprise [2]:
```

### Phase 2: API Key Credentials
```
?  API Key credentials (required for authentication):
    Get yours at: vault.bitwarden.com → Settings → My Account → API Key
    1) Enter credentials now
    2) Skip — configure later with 'bw-plugin auth setup'
?  Choice [1]: 1
    Client ID: user.xxxx-xxxx-xxxx
    Client Secret: ********
✓ API credentials saved to Keychain
```

After setup completes, the tool prints next-step suggestions:
- Run `bw-plugin auth login` to authenticate
- Run `bw-plugin sm-link` to link a Secrets Manager machine account (if applicable)

## Credential Storage Layout

### macOS Keychain
| Service | Value | Account |
|---------|-------|---------|
| `bw-plugin.account.<id>.client_id` | API Client ID | `$USER` |
| `bw-plugin.account.<id>.client_secret` | API Client Secret | `$USER` |
| `bw-plugin.account.<id>.password` | Master password (for unlock) | `$USER` |
| `bws.<profile>.token` | BWS access token | `$USER` |

### Config Files
```
~/.config/bw-plugin/accounts.json      # Account registry
~/.config/bw-plugin/.env               # Optional env vars for credentials
~/.config/bw-plugin/<id>/              # Per-account bw CLI data
~/.config/bws/config                   # BWS profiles
```

## Credential Resolution Order

1. **Environment variables** (`BW_CLIENTID`/`BW_CLIENTSECRET` or `<PREFIX>_CLIENTID`/`<PREFIX>_CLIENTSECRET`)
2. **`.env` file** (`~/.config/bw-plugin/.env`)
3. **macOS Keychain** — set via `bw-plugin auth setup`
4. **Interactive setup prompt** — clear error with command to run `bw-plugin auth setup`

## Auto-Auth Flow (Runtime)

When `ensureAuthFull()` is called:

1. Look up API key credentials (env vars → .env → keychain)
2. If found: `bw login --apikey` (bypasses device verification)
3. Unlock vault with master password: `bw unlock --passwordenv ...`
4. Return session key
5. If credentials missing: print setup instructions and exit with error

## Cross-Account Secret Sharing

With the new account model, secrets can be moved between accounts:

```bash
# Copy a secret from premium to free org account
bw-plugin copy "Cloudflare API" \
  --from vault-bitwarden-com-misterme00-icloud-com \
  --to vault-bitwarden-com-i-mrme0-store

# Move (copy + delete source)
bw-plugin move "Stripe Key" \
  --from vault-bitwarden-com-misterme00-icloud-com \
  --to vault-bitwarden-com-i-mrme0-store
```

## New Commands

| Command | Purpose |
|---------|---------|
| `bw-plugin account add` | Full interactive setup with credential storage |
| `bw-plugin sm-link <id>` | Link Secrets Manager to existing account |
| `bw-plugin copy <item> --from <id> --to <id>` | Cross-account copy |
| `bw-plugin move <item> --from <id> --to <id>` | Cross-account move |
