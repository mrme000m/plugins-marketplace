---
name: bw-status
description: Show Bitwarden vault status for all configured accounts
---

# bw-status Command

When the user runs `/bw-status`, show the current status of all configured Bitwarden accounts.

## Steps

1. Check that `bw-plugin` binary is available in PATH
2. Run `bw-plugin status -j` to get JSON status
3. Display a formatted summary showing:
   - Each account name (personal, work, api)
   - Login status (unauthenticated / locked / unlocked)
   - Active account indicator
   - Email associated with each account
   - Credential source (keychain / env var / none)

## Example Output

```
Bitwarden Vault Status
======================
* personal  — unlocked   misterme00@icloud.com    (keychain)
  work      — locked     i@mrme0.store             (env var)
  api       — unlocked   i@mrme0.store             (keychain + api key)

Active: personal
```

## Troubleshooting

If all accounts show `unauthenticated`:
```bash
bw-plugin auth setup     # Store API key + password in Keychain
bw-plugin auth login     # Login with API key (non-interactive)
bw-plugin auth test      # Verify auto-auth works
```

If `bw-plugin` is not found, prompt the user to install it:
```bash
cd <plugin-dir>/src && make install
```
