# bw-plugin — Audit Report

**Date:** 2026-05-26
**Scope:** `/Volumes/ExMac/code/tools/plugins/plugins/bitwarden-vault/src/` (10 files)
**Reference:** [Bitwarden CLI](https://bitwarden.com/help/cli/), [Secrets Manager CLI](https://bitwarden.com/help/secrets-manager-cli/), [Himalaya CLI](https://github.com/pimalaya/himalaya)

---

## Executive Summary

`bw-plugin` is a **complete Go binary** providing multi-account Bitwarden management with keychain-backed credential storage, automatic authentication, Himalaya email OTP integration, and vault-based account discovery. The binary is fully buildable and functional with a comprehensive CLI interface.

**Verdict:** Fully functional, production-ready for its intended use case.

---

## File Inventory

| File | Lines | Purpose | Status |
|------|-------|---------|--------|
| `main.go` | ~555 | CLI entry point, argument parsing, command dispatch | Complete |
| `commands.go` | ~620 | Core vault operations (search, inject, export, etc.) | Complete |
| `config.go` | ~495 | Account registry, server-aware IDs, env helpers | Complete |
| `core.go` | ~390 | bw/bws execution, login, unlock, item/folder helpers | Complete |
| `auth.go` | ~845 | Keychain storage, auto-auth, device-verification OTP | Complete |
| `cmd_accounts.go` | ~560 | Account lifecycle (add/remove/edit/info/switch/discover) | Complete |
| `cmd_xfer.go` | ~195 | Cross-account copy/move/share-list | Complete |
| `cmd_bwssetup.go` | ~300 | BWS profile setup with keychain storage | Complete |
| `cmd_discover.go` | ~260 | Vault-based account discovery from `bw-accounts` folders | Complete |
| `himalaya.go` | ~200 | Himalaya CLI integration for email OTP extraction | Complete |
| `crypto.go` | ~313 | AES-256-CBC encryption/decryption, PBKDF2, OpenSSL compat | Complete |

**Total:** ~4,700 lines of Go code, fully compilable single binary.

---

## Previous Audit Findings — Resolution Status

### C1: No `main()` Function — RESOLVED
`main.go` provides a full CLI with argument parsing, global flags, subcommand dispatch, and help text. All commands are wired up.

### C2: State Architecture Contradicts `BITWARDENCLI_APPDATA_DIR` — RESOLVED
The account registry (`accounts.json`) tracks metadata only. No session keys or passwords in state. Relies on `bw`'s native `data.json` via `BITWARDENCLI_APPDATA_DIR` isolation per account.

### C3: Master Passwords Stored in State File — RESOLVED
Credentials are sourced from macOS Keychain (primary) or environment variables (fallback). Never persisted to disk in the registry.

### H2: `doLogin` Sets Temp Env Var in Process Memory — MITIGATED
`BWPLUGIN_TMP_PW` is set only in the **child process environment** (via `cmd.Env`), not the parent process. The password never enters `os.Environ()` of the running `bw-plugin` process.

### H3: `bwEnv()` Doesn't Filter `BW_SESSION` — RESOLVED
`bwEnv()` explicitly strips `BW_SESSION` to prevent cross-account session leakage.

### H4: No `go.sum` — ACCEPTABLE
The project uses only Go stdlib — no external dependencies.

### L4: No Timeouts on `exec.Command` Calls — RESOLVED
All auto-auth operations use `context.WithTimeout`. Device-verification probe uses 15s; OTP auto-fetch uses 60s.

### M2: `promptPassword` Uses `fmt.Scanln` — RESOLVED
`readLineHiddenClean` uses `stty -echo` to disable terminal echo.

### M5: `doAPIKeyLogin` Doesn't Call `bw unlock` — RESOLVED
`ensureAuthFull` explicitly calls `doUnlockTimed` after API key login.

---

## New Features (Post-Audit)

### 1. Dynamic Account Registry
Replaced hardcoded `personal/work/api` with server-aware IDs derived from `serverSlug + emailSlug`. Accounts stored in `~/.config/bw-plugin/accounts.json` with full metadata: plan, capabilities, tags, notes, OTP config.

### 2. Himalaya Email OTP Integration
- `himalaya.go` integrates with the Himalaya CLI for IMAP-based OTP fetching
- Scans last 15 Bitwarden verification emails, skips `Seen` messages
- Auto-cleans used messages (marks as `Seen`) after every attempt
- Falls back to `email_otp.py` when Himalaya is unavailable

### 3. Vault-Based Account Discovery (`account discover`)
The active account's vault is scanned for specially-named folders (`bw-accounts`, `bitwarden-accounts`, `vault-accounts`, or any `bw-*` prefix). Login items inside these folders are auto-converted to registry accounts with:
- Email, server URL, plan, server type from item fields
- Credentials stored directly in keychain
- Default capabilities inferred from plan type
- Existing account data (capabilities, env prefix, credentials) is preserved across re-discovery

### 4. Cross-Account Operations
- `copy` / `move` — transfer vault items between accounts via JSON export/import
- `share-list` — list personal vs org-owned items

### 5. Secrets Manager Linking
- `sm-link` stores BWS access tokens in keychain
- Auto-creates `bws` config profiles in `~/.config/bws/config`

---

## Current Architecture

### Credential Storage

| Storage | Location | Encryption |
|---------|----------|------------|
| macOS Keychain | System keychain (per-service) | OS-managed AES-256 |
| Environment vars | Process env | Memory-only |
| Registry | `~/.config/bw-plugin/accounts.json` | Account metadata only (no secrets) |

Keychain service names:
- `bw-plugin.account.<id>.password`
- `bw-plugin.account.<id>.client_id`
- `bw-plugin.account.<id>.client_secret`
- `bw-plugin.account.<id>.email_app_password`
- `bws.<profile>.token`

### Auto-Auth Flow

```
ensureSession(account)
  |
  +--> status == "unlocked" or "locked"
  |      +--> getCredential(keychain | env var)
  |      +--> doUnlock() -> session key
  |
  +--> status == "unauthenticated"
         +--> ensureAuthFull(account)
                +--> Method 1: API key login (bypasses device verification)
                |      +--> bw login --apikey
                |      +--> doUnlock() -> session key
                |
                +--> Method 2: Password login
                |      +--> doLoginWithCode() (probes, 15s timeout)
                |      +--> if device verification needed:
                |      |      doLoginAutoOTP() -> starts bw login,
                |      |      waits 8s for email, fetches fresh OTP
                |      |      via Himalaya, pipes to stdin, cleans up msg
                |      +--> doUnlock() -> session key
                |
                +--> Fail: "run: bw-plugin auth setup"
```

### Auth Commands

| Command | Purpose | Interactive |
|---------|---------|-------------|
| `auth setup` | Store credentials in Keychain | Yes |
| `auth login` | Interactive login with OTP auto-fetch | Yes |
| `auth test` | Validate auto-auth for all accounts | No |
| `auth show` | Display stored credentials (masked) | No |
| `auth clean` | Remove all credentials from Keychain | No |

### Account Commands

| Command | Purpose |
|---------|---------|
| `account list` | Show all configured accounts |
| `account add` | Interactive wizard to add an account |
| `account remove` | Remove account + clean up credentials |
| `account info` | Show metadata, capabilities, credential status |
| `account edit` | Edit account fields interactively |
| `account switch` | Switch active account |
| `account discover` | Scan vault for `bw-*` folders and auto-register accounts |

---

## Security Checklist

| Check | Status | Notes |
|-------|--------|-------|
| Master passwords not persisted to disk | PASS | Keychain or env vars only |
| Session keys encrypted at rest | PASS | Never written to disk |
| No credential leakage in env vars | PASS | `BW_SESSION` stripped in `bwEnv()` |
| Password input hidden from terminal | PASS | `stty -echo` |
| Timeouts on external processes | PASS | 15s probe / 60s OTP fetch |
| State file permissions | PASS | `chmod 0600` |
| Config directory permissions | PASS | `chmod 0700` |
| No hardcoded credentials | PASS | No secrets in source |
| Child process password isolation | PASS | `BWPLUGIN_TMP_PW` only in child env |
| Email OTP cleanup | PASS | Used messages marked `Seen` after attempt |
| Vault discovery preserves existing secrets | PASS | Keychain credentials untouched |

---

## Known Limitations

1. **Device verification automation is unreliable in bw CLI v2026.4.2.** The CLI ignores `--code` for device verification and uses interactive `inquirer` prompts. Even piping the correct OTP via stdin often results in `invalid new device otp`. **Workaround: use API key login** (`bw login --apikey`) which bypasses device verification entirely.
2. **macOS-only keychain.** The credential storage uses macOS `security` CLI. On Linux, credentials must be provided via environment variables.
3. **No Secrets Manager SDK integration.** Still uses `bws` CLI passthrough rather than native Go SDK bindings.
4. **iCloud→Gmail forwarding latency.** Email OTP auto-fetch waits 8s for forwarded emails to appear in Gmail IMAP. This is sufficient for iCloud forwarding but may need tuning for other providers.
