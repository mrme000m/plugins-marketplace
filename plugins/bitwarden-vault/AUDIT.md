# bw-plugin — Audit Report

**Date:** 2026-05-27
**Scope:** `/Volumes/ExMac/code/tools/plugins/plugins/bitwarden-vault/src/` (10 files)
**Reference:** [Bitwarden CLI](https://bitwarden.com/help/cli/), [Secrets Manager CLI](https://bitwarden.com/help/secrets-manager-cli/)

---

## Executive Summary

`bw-plugin` is a **complete Go binary** providing multi-account Bitwarden management with keychain-backed credential storage, automatic API key authentication, and vault-based account discovery. The binary is fully buildable and functional with a comprehensive CLI interface.

**Verdict:** Fully functional, production-ready for its intended use case.

---

## File Inventory

| File | Lines | Purpose | Status |
|------|-------|---------|--------|
| `main.go` | ~555 | CLI entry point, argument parsing, command dispatch | Complete |
| `commands.go` | ~620 | Core vault operations (search, inject, export, etc.) | Complete |
| `config.go` | ~495 | Account registry, server-aware IDs, env helpers | Complete |
| `core.go` | ~390 | bw/bws execution, API key login, unlock, item/folder helpers | Complete |
| `auth.go` | ~350 | Keychain storage, API key auto-auth, credential resolution | Complete |
| `cmd_accounts.go` | ~560 | Account lifecycle (add/remove/edit/info/switch) | Complete |
| `cmd_xfer.go` | ~195 | Cross-account copy/move/share-list | Complete |
| `cmd_bwssetup.go` | ~300 | BWS profile setup with keychain storage | Complete |
| `cmd_discover.go` | ~260 | Vault-based account discovery from `bw-accounts` folders | Complete |
| `crypto.go` | ~313 | AES-256-CBC encryption/decryption, PBKDF2, OpenSSL compat | Complete |

**Total:** ~4,000 lines of Go code, fully compilable single binary.

---

## Previous Audit Findings — Resolution Status

### C1: No `main()` Function — RESOLVED
`main.go` provides a full CLI with argument parsing, global flags, subcommand dispatch, and help text. All commands are wired up.

### C2: State Architecture Contradicts `BITWARDENCLI_APPDATA_DIR` — RESOLVED
The account registry (`accounts.json`) tracks metadata only. No session keys or passwords in state. Relies on `bw`'s native `data.json` via `BITWARDENCLI_APPDATA_DIR` isolation per account.

### C3: Master Passwords Stored in State File — RESOLVED
Credentials are sourced from environment variables (primary), `.env` file, or macOS Keychain. Never persisted to disk in the registry.

### H2: `doLogin` Sets Temp Env Var in Process Memory — MITIGATED
`BWPLUGIN_TMP_PW` is set only in the **child process environment** (via `cmd.Env`), not the parent process. The password never enters `os.Environ()` of the running `bw-plugin` process.

### H3: `bwEnv()` Doesn't Filter `BW_SESSION` — RESOLVED
`bwEnv()` explicitly strips `BW_SESSION` to prevent cross-account session leakage.

### H4: No `go.sum` — ACCEPTABLE
The project uses only Go stdlib — no external dependencies.

### L4: No Timeouts on `exec.Command` Calls — RESOLVED
All auto-auth operations use `context.WithTimeout` with 30s timeouts.

### M2: `promptPassword` Uses `fmt.Scanln` — RESOLVED
`readLineHiddenClean` uses `stty -echo` to disable terminal echo.

### M5: `doAPIKeyLogin` Doesn't Call `bw unlock` — RESOLVED
`ensureAuthFull` explicitly calls `doUnlockTimed` after API key login.

---

## Current Architecture

### Credential Storage

| Storage | Location | Encryption |
|---------|----------|------------|
| macOS Keychain | System keychain (per-service) | OS-managed AES-256 |
| `.env` file | `~/.config/bw-plugin/.env` | File permissions (0600) |
| Environment vars | Process env | Memory-only |
| Registry | `~/.config/bw-plugin/accounts.json` | Account metadata only (no secrets) |

Keychain service names:
- `bw-plugin.account.<id>.client_id`
- `bw-plugin.account.<id>.client_secret`
- `bw-plugin.account.<id>.password`
- `bws.<profile>.token`

### Auto-Auth Flow

```
ensureSession(account)
  |
  +--> status == "unlocked" or "locked"
  |      +--> getCredential(env var | .env | keychain)
  |      +--> doUnlock() -> session key
  |
  +--> status == "unauthenticated"
         +--> ensureAuthFull(account)
                +--> API key login (bypasses device verification)
                |      +--> bw login --apikey
                |      +--> doUnlock() -> session key
                |
                +--> Fail: print setup instructions
                         "run: bw-plugin auth setup"
```

### Auth Commands

| Command | Purpose | Interactive |
|---------|---------|-------------|
| `auth setup` | Store API key + password in Keychain | Yes |
| `auth login` | Login with API key | No |
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
| Timeouts on external processes | PASS | 30s for all auth operations |
| State file permissions | PASS | `chmod 0600` |
| Config directory permissions | PASS | `chmod 0700` |
| No hardcoded credentials | PASS | No secrets in source |
| Child process password isolation | PASS | `BWPLUGIN_TMP_PW` only in child env |
| Vault discovery preserves existing secrets | PASS | Keychain credentials untouched |

---

## Known Limitations

1. **macOS-only keychain.** The credential storage uses macOS `security` CLI. On Linux, credentials must be provided via `.env` file or environment variables.
2. **No Secrets Manager SDK integration.** Still uses `bws` CLI passthrough rather than native Go SDK bindings.
3. **Master password still required for unlock.** API key login authenticates, but vault unlock requires the master password.
