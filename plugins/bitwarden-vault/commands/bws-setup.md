---
name: bws-setup
description: Interactive setup for Bitwarden Secrets Manager (bws) credentials. Configures profiles, stores access tokens in macOS Keychain or shell profile, and tests connectivity.
---

# bws-setup Command

When the user runs `/bws-setup` or `bw-plugin bws-setup`, perform an interactive setup for Bitwarden Secrets Manager CLI credentials.

## Steps

1. Verify `bws` binary is available (checks `~/bin/bws` and PATH)
2. Show current `~/.config/bws/config` and existing keychain entries
3. Prompt interactively for:
   - Profile name (default: `default`)
   - Server base URL (default: `https://api.bitwarden.com`)
   - Access token (from Bitwarden web app → Machine Account → Access Tokens)
   - Storage preference: Keychain / Shell profile / Both
4. Write the profile to `~/.config/bws/config`
5. Store the access token securely
6. Test connection by listing secrets
7. Show usage summary

## Access Token Source

The access token comes from the Bitwarden web app:
1. Navigate to your organization → **Secrets Manager**
2. **Machine Accounts** → select or create a machine account
3. **Access Tokens** tab → **Create Access Token**
4. Copy the token immediately (it is shown once)

## Storage Options

| Option | Location | Use Case |
|--------|----------|----------|
| Keychain | macOS Keychain (`bws.<profile>.token`) | Secure, recommended |
| Shell profile | `~/.zshrc` or `~/.bash_profile` | Convenient, less secure |
| Both | Keychain + shell profile | Flexibility |

## Example Session

```
  ┌─ Bitwarden Secrets Manager CLI Setup ─────────────┐

✓  bws found: /Users/m/bin/bws (bws 2.1.0)

ℹ  Current bws config (/Users/m/.config/bws/config):
    [profiles.default]
    server_base = "https://api.bitwarden.com"

?  Profile name [default]: production
?  Server base URL [https://api.bitwarden.com]:
?  Access token (from Bitwarden Secrets Manager web app): ****

ℹ  Where should the access token be stored?
    1) macOS Keychain (secure, recommended)
    2) Shell profile as env var (~/.zshrc or ~/.bash_profile)
    3) Both
?  Choice [1]: 1

ℹ  Applying configuration...
✓  Updated /Users/m/.config/bws/config with profile 'production'
✓  Stored access token in macOS Keychain (bws.production.token)

?  Test the connection now? [Y/n]: Y

✓  Connection successful! Secrets are accessible.

  ┌─ Setup Complete ──────────────────────────────────┐

  Profile:     production
  Server:      https://api.bitwarden.com
  Token:       0.1845****CA==
  Keychain:    ✓ stored
  Shell env:   — not used

  Usage examples:
    bws -p production secret list
    bws -p production secret get <secret-id>
    bws -p production project list
    BWS_PROFILE=production bws secret list

  To load token from keychain in scripts:
    export BWS_ACCESS_TOKEN=$(security find-generic-password -a "$USER" -s "bws.production.token" -w)

✓  Done!
```

## Troubleshooting

**Connection failed:**
- Verify the access token was copied correctly (it is shown only once in the web app)
- Check the server URL — EU organizations use `https://api.bitwarden.eu`
- Ensure the machine account has access to at least one project
- Try revoking the old token and generating a new one

**Keychain not available:**
- macOS only — use shell profile storage on Linux
- Ensure `security` CLI is in PATH

**Profile already exists:**
- The setup rewrites the profile section in `~/.config/bws/config`
- Previous keychain entries for the same profile name are overwritten
