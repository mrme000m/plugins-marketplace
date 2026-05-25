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

## Example Output

```
Bitwarden Vault Status
======================
● personal  — unlocked   misterme00@icloud.com
○ work      — locked     i@mrme0.store
○ api       — unauthenticated  i@mrme0.store

Active: personal
```

## Troubleshooting

If `bw-plugin` is not found, prompt the user to install it:
```bash
cd <plugin-dir>/src && go build -o bw-plugin . && cp bw-plugin ~/bin/
```
