# bw-plugin — Audit Report

**Date:** 2026-05-25  
**Auditor:** System Analysis  
**Scope:** `/Volumes/ExMac/code/Agents/bw-plugin/` (3 files: `go.mod`, `config.go`, `core.go`)  
**Reference:** [Bitwarden Secrets Manager SDK](https://bitwarden.com/help/secrets-manager-sdk/), [Password Manager CLI](https://bitwarden.com/help/cli/), [Secrets Manager CLI](https://bitwarden.com/help/secrets-manager-cli/)

---

## Executive Summary

`bw-plugin` is a **Go binary** intended as a multi-account Bitwarden manager with both `bw` (Password Manager) and `bws` (Secrets Manager) integration. It is **incomplete** — no `main()` function, no CLI entry point, no `go.sum`, and no dependencies declared. The code has a solid foundation but contains **architectural contradictions**, **security issues**, and **incomplete features**.

**Verdict:** 40% complete, needs significant rework before production use.

---

## File Inventory

| File | Lines | Purpose | Status |
|------|-------|---------|--------|
| `go.mod` | 3 | Module declaration | ⚠️ Incomplete — no dependencies |
| `config.go` | 250 | Account config, state management, env helpers | ✅ Usable but has issues |
| `core.go` | 421 | bw/bws execution, login, item helpers, clipboard, PID mgmt | ⚠️ Partial — no main(), no CLI |

**Total:** 674 lines of Go code, **0 lines of executable entry point**.

---

## Critical Issues

### C1: No `main()` Function — Cannot Build or Run

There is no `func main()` or CLI entry point. The code defines utilities but nothing to invoke them. The `init()` in `config.go` ensures directories exist, but that's the only side effect.

**Impact:** The code cannot be compiled into a usable binary.

**Fix:** Add a `main.go` with a CLI framework (e.g., `cobra`, `urfave/cli`, or manual `os.Args` parsing).

### C2: State Architecture Contradicts `BITWARDENCLI_APPDATA_DIR`

The `State` struct stores session keys manually:

```go
type State struct {
    ActiveAccount    string
    PersonalSession  string    // ← Manual session key storage
    WorkSession      string
    APISession       string
    AccountPasswords map[string]string  // ← Passwords in JSON on disk
}
```

This **contradicts** the `BITWARDENCLI_APPDATA_DIR` approach. When you set `BITWARDENCLI_APPDATA_DIR`, `bw login` stores the session **inside** `data.json` automatically. The plugin doesn't need to track session keys — it should rely on `bw`'s native session management.

**Impact:** Redundant, potentially insecure session storage. Session keys in `state.json` are plaintext JSON (not encrypted like `data.json`).

**Fix:** Remove `PersonalSession`, `WorkSession`, `APISession` from `State`. Use `bw status` to check session validity. Rely on `data.json` for session persistence.

### C3: Master Passwords Stored in State File

```go
AccountPasswords map[string]string `json:"account_passwords,omitempty"`
```

This field in `State` is designed to store master passwords in `state.json`. Even though `state.json` is `chmod 0600`, **master passwords should never be persisted to disk** — not even in a restricted file.

**Impact:** If the machine is compromised, all three vault master passwords are recoverable from a single file.

**Fix:** Remove `AccountPasswords` entirely. Use environment variables or macOS Keychain for password storage.

---

## High-Severity Issues

### H1: No Secrets Manager SDK Integration

Despite the project name suggesting SDK usage, the code only shells out to the `bws` CLI binary:

```go
func bwsRun(args ...string) ([]byte, error) {
    cmd := exec.Command(findBWS(), args...)
    cmd.Env = os.Environ()
    return cmd.Output()
}
```

The [Bitwarden Secrets Manager SDK](https://bitwarden.com/help/secrets-manager-sdk/) provides **native Go bindings** via CGO:

```go
import "github.com/bitwarden/sdk-go/v2"
bitwardenClient, _ := sdk.NewBitwardenClient(&apiURL, &identityURL)
err := bitwardenClient.AccessTokenLogin(accessToken, &stateFile)
secrets, _ := bitwardenClient.Secrets().List("org_id")
```

**Impact:** Missing the core value proposition. The SDK provides:
- Direct API access (no subprocess overhead)
- End-to-end encryption handled by the Rust core
- `Secrets.Sync()` for incremental sync
- No dependency on `bws` binary being installed

**Fix:** Add SDK dependency and implement native SDK client alongside (or instead of) CLI shelling.

### H2: `doLogin` Sets Temp Env Var with Master Password in Process Memory

```go
func doLogin(account string, password string) (string, error) {
    envName := "BWPLUGIN_TMP_PW"
    os.Setenv(envName, password)  // ← Password in process env
    defer os.Unsetenv(envName)
    out, err = bwRunCombined(account, "login", acc.Email, "--passwordenv", envName, "--raw")
```

While `--passwordenv` avoids passing the password as a CLI argument (which would show in `ps`), it still places the password in the process environment. On macOS, environment variables are visible to child processes and can be read from `/proc/<pid>/environ` on Linux.

**Better approach:** Use `bw login --passwordfile` with a temp file that's deleted immediately, or use `--raw` with stdin piping.

### H3: `bwEnv()` Filters Incorrectly

```go
func bwEnv(account string) []string {
    env := os.Environ()
    var filtered []string
    hasAppdata := false
    for _, e := range env {
        if strings.HasPrefix(e, "BITWARDENCLI_APPDATA_DIR=") {
            filtered = append(filtered, "BITWARDENCLI_APPDATA_DIR="+appdata)
            hasAppdata = true
        } else {
            filtered = append(filtered, e)
        }
    }
    if !hasAppdata {
        filtered = append(filtered, "BITWARDENCLI_APPDATA_DIR="+appdata)
    }
    return filtered
}
```

This replaces ALL env vars starting with `BITWARDENCLI_APPDATA_DIR=` — but if there are multiple (unlikely but possible), only the first match gets replaced and the rest are passed through unchanged. More importantly, it doesn't filter out `BW_SESSION` which could leak a session from a different account.

**Fix:** Also filter out `BW_SESSION` to prevent cross-account session leakage.

### H4: No `go.sum` — Dependencies Not Locked

`go.mod` declares `go 1.22` but has zero dependencies. The code uses only stdlib, so this is technically fine for now. But once the SDK is added, a `go.sum` will be required for reproducible builds.

---

## Medium-Severity Issues

### M1: `setServer()` Called Redundantly

```go
func setServer(account string) error {
    acc, ok := getAccount(account)
    if !ok {
        return fmt.Errorf("unknown account: %s", account)
    }
    _, _ = bwRunCombined(account, "config", "server", acc.Server)
    return nil
}
```

When using `BITWARDENCLI_APPDATA_DIR`, the server URL is stored in each account's `data.json`. Setting it via `bw config server` is unnecessary after the first login — `bw` remembers the server per `data.json`. This function is called before every `login` and `unlock`, adding unnecessary subprocess overhead.

### M2: `promptPassword` Uses `fmt.Scanln` — Password Visible on Screen

```go
func promptPassword(prompt string) (string, error) {
    fmt.Fprintf(os.Stderr, "%s: ", prompt)
    var password string
    _, err := fmt.Scanln(&password)
    return password, err
}
```

This reads the password as plain text — characters are echoed to the terminal. Should use `golang.org/x/term.ReadPassword` for hidden input.

### M3: `copyToClipboard` Has Windows/Linux Fallbacks But Plugin Is macOS-Targeted

The clipboard function supports macOS (`pbcopy`), Linux (`xclip`, `wl-copy`), and Windows (PowerShell). Given the account configuration is hardcoded to macOS paths (`/opt/homebrew/bin/bw`), the cross-platform clipboard support adds complexity without benefit.

### M4: `isProcessRunning` Uses Signal 0 — Always Succeeds on Unix for Zombies

```go
func isProcessRunning(pid int) bool {
    proc, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    err = proc.Signal(os.Signal(nil))
    return err == nil
}
```

On Unix, `os.FindProcess` always succeeds (it just creates a Process object). `Signal(nil)` checks if the process exists, but zombie processes will still return `nil` error. Should use `syscall.Kill(pid, 0)` and check for `syscall.ESRCH`.

### M5: `doAPIKeyLogin` Doesn't Call `bw unlock` After `bw login --apikey`

Per the [official docs](https://bitwarden.com/help/personal-api-key/):

> Using the personal API key for CLI authentication... to use many of the CLI commands you will need to subsequently decrypt your data using the `unlock` command. Your API key is **not a substitute for your master password.**

The `doAPIKeyLogin` function runs `bw login --apikey` but doesn't follow up with `bw unlock`. This means the session is authenticated but vault data is still encrypted.

---

## Low-Severity Issues

### L1: `getAccount` Reads Config File on Every Call

```go
func getAccount(name string) (Account, bool) {
    cfg := loadConfig()  // ← Reads from disk every time
    if acc, ok := cfg.Accounts[name]; ok {
        return acc, true
    }
    acc, ok := defaultAccounts[name]
    return acc, ok
}
```

`loadConfig()` reads `config.json` from disk on every call. Should cache after first load.

### L2: `init()` Swallows Errors

```go
func init() {
    _ = ensureDirs()  // ← Error silently ignored
}
```

If directory creation fails, the plugin will fail later with confusing errors. Should at least log.

### L3: `accountOrder` Not Synced with `defaultAccounts` Keys

If a new account is added to `defaultAccounts` but not to `accountOrder`, it won't appear in iterations. Should derive `accountOrder` from `defaultAccounts` keys or use a single source of truth.

### L4: No Timeouts on `exec.Command` Calls

All `bwRun` variants use `exec.Command` without timeouts. If `bw` hangs (network issue, hCaptcha challenge), the plugin hangs indefinitely.

### L5: `findBW()` and `findBWS()` Don't Validate Binary Executability

The code checks `os.Stat(c)` but doesn't verify the file is executable. A non-executable file at `/opt/homebrew/bin/bw` would pass the check but fail at runtime.

---

## Architecture Assessment

### What the Plugin Tries to Be

Based on the code structure, `bw-plugin` appears designed as a **Go binary** that would serve as:

1. **A unified CLI** for managing all three Bitwarden accounts
2. **A `bw serve` daemon** (PID management suggests a long-running process)
3. **A bridge between `bw` CLI and `bws` SDK** for applications

### What It Actually Is

A **library of utilities** with no entry point. It has:
- ✅ Account configuration system
- ✅ State management (but flawed)
- ✅ `bw` subprocess execution with `BITWARDENCLI_APPDATA_DIR` isolation
- ✅ Login/unlock helpers
- ✅ Item retrieval helpers
- ✅ Clipboard integration
- ✅ Process/PID management (for a `serve` mode that doesn't exist)
- ❌ No CLI interface
- ❌ No `serve` command implementation
- ❌ No SDK integration (only CLI shelling)
- ❌ No tests
- ❌ No build instructions

### Comparison with Shell-Based `bwc`

| Feature | Shell `bwc` (~/bin/bwc) | Go `bw-plugin` |
|---------|------------------------|----------------|
| Session isolation | ✅ `BITWARDENCLI_APPDATA_DIR` | ✅ `BITWARDENCLI_APPDATA_DIR` |
| Direct account access | ✅ `bwp`, `bww`, `bwa` | ❌ Not implemented |
| Multi-account status | ✅ Displays all three | ❌ No CLI |
| Login automation | ✅ Via env vars | ⚠️ Implemented but no entry point |
| Session security | ✅ Relies on `data.json` encryption | ❌ Stores sessions in plaintext JSON |
| Password security | ✅ Env vars only | ❌ Has `AccountPasswords` field for disk storage |
| bws integration | ⚠️ CLI shelling only | ⚠️ CLI shelling only (no SDK) |
| Completeness | ✅ Fully functional | ❌ 40% complete |
| Buildable | N/A (shell function) | ❌ No `main()` |

---

## Recommendations

### Immediate (Must Fix)

1. **Remove `AccountPasswords` from `State`** — Never persist master passwords to disk
2. **Remove session key fields from `State`** — Rely on `data.json` for session management
3. **Add `main.go` with CLI entry point** — At minimum, support: `bw-plugin status`, `bw-plugin login <account>`, `bw-plugin get <account> <item>`
4. **Filter `BW_SESSION` in `bwEnv()`** — Prevent cross-account session leakage

### Short-term (Should Fix)

5. **Add Secrets Manager SDK dependency** — Use native Go SDK instead of shelling to `bws`
6. **Fix `doAPIKeyLogin` to call `bw unlock`** — API key login requires subsequent unlock
7. **Use `golang.org/x/term` for password input** — Hidden input, not `fmt.Scanln`
8. **Add command timeouts** — 30s default timeout on all `bw`/`bws` calls
9. **Add `go.sum` and dependencies** — `go mod tidy` after adding SDK

### Medium-term (Nice to Have)

10. **Implement `serve` mode** — If PID management is intentional, build the daemon
11. **Add tests** — Unit tests for config loading, env building, JSON parsing
12. **Cache config after first load** — Don't read disk on every `getAccount()` call
13. **Support `bw send` operations** — Ephemeral secret sharing
14. **Add shell completion generation** — For the CLI

### Architectural Decision: Go Binary vs. Shell Function

**Recommendation:** Keep the shell `bwc` for interactive use. The Go plugin should be a **library** or **daemon** that provides:

- A `bw-plugin serve` mode that runs as a background process
- A local API (HTTP or Unix socket) that other tools can query
- Native SDK integration for Secrets Manager operations
- Session health monitoring and auto-refresh

The shell `bwc` is faster to iterate, easier to debug, and already functional. The Go plugin should complement it, not replace it.

---

## SDK Integration Plan

Based on the [official SDK docs](https://bitwarden.com/help/secrets-manager-sdk/), here's how to integrate:

```go
// go.mod additions
require github.com/bitwarden/sdk-go/v2 v2.x.x

// Example usage
import "github.com/bitwarden/sdk-go/v2"

func newBWSClient(accessToken string) (*sdk.BitwardenClient, error) {
    apiURL := "https://api.bitwarden.com"
    identityURL := "https://identity.bitwarden.com"
    
    client, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
    if err != nil {
        return nil, err
    }
    
    stateFile := os.ExpandEnv("$HOME/.config/bws/state")
    err = client.AccessTokenLogin(accessToken, &stateFile)
    if err != nil {
        return nil, err
    }
    
    return client, nil
}

func listSecrets(client *sdk.BitwardenClient, orgID string) ([]sdk.SecretResponse, error) {
    return client.Secrets().List(orgID)
}

func getSecret(client *sdk.BitwardenClient, secretID string) (*sdk.SecretResponse, error) {
    return client.Secrets().Get(secretID)
}
```

**Key SDK advantages over CLI shelling:**
- No subprocess overhead
- Direct access to `Secrets.Sync()` for incremental sync
- Native error types (not parsing stderr)
- Connection pooling
- State file management handled by SDK

---

## Security Checklist

| Check | Status | Notes |
|-------|--------|-------|
| Master passwords not persisted to disk | ❌ FAIL | `AccountPasswords` field exists |
| Session keys encrypted at rest | ❌ FAIL | Stored in plaintext `state.json` |
| No credential leakage in env vars | ⚠️ PARTIAL | `BW_SESSION` not filtered in `bwEnv()` |
| Password input hidden from terminal | ❌ FAIL | Uses `fmt.Scanln` (echoes) |
| Binary paths validated | ⚠️ PARTIAL | Checks existence, not executability |
| Timeouts on external processes | ❌ FAIL | No timeouts on `exec.Command` |
| State file permissions | ✅ PASS | `chmod 0600` on state file |
| Config directory permissions | ✅ PASS | `chmod 0700` on config dir |
| No hardcoded credentials | ✅ PASS | No secrets in source code |
| Error messages don't leak secrets | ⚠️ PARTIAL | `doLogin` returns full output in error |

---

## Conclusion

The `bw-plugin` codebase has a **solid foundation** for account configuration and `bw` subprocess management but is **not production-ready**. The most critical issues are the missing `main()` function, the contradictory state management approach, and the security risk of persisting master passwords.

**The shell-based `bwc` (~/bin/bwc) is currently the more complete and functional solution.** The Go plugin should be developed as a complementary daemon/SDK integration layer, not as a replacement for the shell wrapper.
