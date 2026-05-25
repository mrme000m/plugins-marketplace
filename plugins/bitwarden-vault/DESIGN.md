# Premium Account + Secrets Manager Setup Flow

## Goal
Streamlined interactive flow to add a premium Bitwarden account, store all credentials securely, enable auto-OTP via IMAP, and generate/link a Secrets Manager access token.

## Account Model Extension

```go
type Account struct {
    // ... existing fields ...
    
    // Email OTP configuration
    EmailProvider   string `json:"email_provider,omitempty"`   // gmail | icloud | outlook | yahoo | custom
    EmailIMAPServer string `json:"email_imap_server,omitempty"` // imap.gmail.com | imap.mail.me.com | ...
    EmailIMAPPort   int    `json:"email_imap_port,omitempty"`   // 993 | 587
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

### Phase 2: Vault Credentials
```
?  Master password: ********
?  API Client ID (optional, for API key login): 
?  API Client Secret (optional):
```

### Phase 3: Device Verification OTP Setup
```
?  Which email receives Bitwarden device verification codes?
    1) Same as account email (misterme00@icloud.com)
    2) Different email
?  Choice [1]: 2
?  Verification email: mrme000.m0@gmail.com

?  Email provider for IMAP access:
    1) Gmail (imap.gmail.com)
    2) iCloud (imap.mail.me.com)
    3) Outlook/Hotmail (outlook.office365.com)
    4) Yahoo (imap.mail.yahoo.com)
    5) Other (manual)
?  Choice [1]: 1

?  App password for Gmail IMAP:
    Generate at: https://myaccount.google.com/apppasswords
    > ********
```

### Phase 4: Auto-Auth Test
```
→ Testing auto-auth flow...
→ Logging into vault-bitwarden-com-misterme00-icloud-com...
⚠ Device verification required
→ Checking Gmail for Bitwarden OTP...
✓ Found OTP: 123456
✓ Logged in (device verified)
✓ Unlocked vault
✓ Session: a1b2c3d4...
```

### Phase 5: Secrets Manager Link (if applicable)
```
→ Checking organization for Secrets Manager...
✓ Org "dev" has Secrets Manager enabled

?  Link a Secrets Manager machine account?
    1) Yes — list available machine accounts
    2) No — skip
?  Choice [1]: 1

→ Available machine accounts:
    1) CI/CD Pipeline (read: Production, Staging)
    2) Local CLI (read: Default)
?  Select machine account [1]: 2

→ Generating access token for "Local CLI"...
✓ Access token created: 0.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.xxxxxxxxxx:xxxxxxxxxx
⚠ Copy this token NOW — it won't be shown again

?  Store BWS access token in keychain? [Y/n]: Y
✓ Stored as bws.vault-bitwarden-com-misterme00-icloud-com.token

✓ Account setup complete!
```

## Credential Storage Layout

### macOS Keychain
| Service | Value | Account |
|---------|-------|---------|
| `bw-plugin.account.<id>.password` | Master password | `$USER` |
| `bw-plugin.account.<id>.client_id` | API Client ID | `$USER` |
| `bw-plugin.account.<id>.client_secret` | API Client Secret | `$USER` |
| `bw-plugin.account.<id>.email_app_password` | Email IMAP app password | `$USER` |
| `bws.<profile>.token` | BWS access token | `$USER` |

### Config Files
```
~/.config/bw-plugin/accounts.json      # Account registry
~/.config/bw-plugin/<id>/              # Per-account bw CLI data
~/.config/bws/config                   # BWS profiles
```

## Auto-OTP Flow (Runtime)

When `ensureAuthFull()` encounters device verification:

1. Check if `email_app_password` is stored for the account
2. Check `email_provider` to determine IMAP server
3. Connect via IMAP using generic Python script:
   ```python
   python3 email_otp.py \
     --provider gmail \
     --email mrme000.m0@gmail.com \
     --app-password xxxx \
     --sender do-not-reply@bitwarden.com
   ```
4. Extract 6-digit code from latest matching email
5. Retry `bw login --code <otp>`
6. If IMAP fails, fall back to interactive prompt

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
| `bw-plugin account add` | Full interactive setup with OTP + SM link |
| `bw-plugin account link-sm <id>` | Link Secrets Manager to existing account |
| `bw-plugin sm-tokens <id>` | List/manage SM access tokens for account |
| `bw-plugin copy <item> --from <id> --to <id>` | Cross-account copy |
| `bw-plugin move <item> --from <id> --to <id>` | Cross-account move |
