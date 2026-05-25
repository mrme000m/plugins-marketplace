# bw-plugin

A single, cross-platform Go binary that consolidates all Bitwarden CLI multi-account management, vault operations, and Secrets Manager integration.

Replaces the previous collection of zsh functions and helper scripts (`bwc`, `bw-status`, `bw-search`, `bw-inject`, `bw-totp`, `bw-export`) with a unified, compiled tool.

## Features

- **Multi-account management** — Personal Premium, Legacy NodeWarden, API Keys Vault
- **Isolated sessions** — Each account uses its own `BITWARDENCLI_APPDATA_DIR`
- **Keychain credential storage** — Master passwords and API keys stored in macOS Keychain
- **Automatic authentication** — Vault operations auto-login and unlock using stored credentials
- **Multiple auth methods** — API key login (bypasses device verification), master password, env var fallback
- **Timeout-protected** — All auto-auth attempts have 30s timeouts with closed stdin to prevent 2FA hang
- **Vault operations** — Search, inject credentials as env vars, TOTP, export with encryption
- **Secrets Manager** — `bws` passthrough with `--no-inherit-env` default
- **HTTP API server** — `bw serve` management (start/stop/status)
- **Cross-platform** — macOS, Linux, Windows (single static binary, zero dependencies)

## Installation

```bash
# From project directory
make install

# Or manually:
# 1. Build: GOROOT=$(brew --prefix)/Cellar/go/$(go version | awk '{print $3}' | sed 's/go//')/libexec go build -ldflags "-s -w" -o bw-plugin .
# 2. Copy to PATH: cp bw-plugin ~/bin/
# 3. Create symlinks:
#    ln -s ~/bin/bw-plugin ~/bin/bwp
#    ln -s ~/bin/bw-plugin ~/bin/bww
#    ln -s ~/bin/bw-plugin ~/bin/bwa
```

## First-Time Setup

```bash
# 1. Store credentials in macOS Keychain
bw-plugin auth setup

# 2. Login to each account (handles device verification OTP interactively)
bw-plugin auth login

# 3. Verify all accounts authenticate
bw-plugin auth test
```

After setup, all vault operations auto-authenticate transparently.

## Usage

### Auth Management

```bash
bw-plugin auth setup              # Store credentials in macOS Keychain (interactive)
bw-plugin auth login              # Interactive login for all accounts (handles OTP)
bw-plugin auth login personal     # Login to specific account only
bw-plugin auth test               # Test auto-auth flow for all accounts
bw-plugin auth show               # Show stored credentials (masked)
bw-plugin auth clean              # Remove all credentials from Keychain
```

**Credential resolution order:**
1. macOS Keychain (primary — set via `auth setup`)
2. Environment variables (`BWP_PASSWORD`, `BWW_PASSWORD`, `BWA_PASSWORD`)
3. API keys (`BWP_CLIENTID`/`BWP_CLIENTSECRET`, etc.)

**Auto-auth flow** (triggered by any vault operation):
1. Check `bw status` for the target account
2. If **unauthenticated**: try API key login first (bypasses device verification), then password login
3. If **locked**: unlock with stored credentials
4. All attempts: 30s timeout + stdin closed (prevents interactive prompt hangs)

### Status & Account Switching

```bash
bw-plugin                    # Show all accounts and their status
bw-plugin status -j          # JSON output
bw-plugin switch             # Cycle: personal -> work -> api
bw-plugin switch work        # Switch to specific account
bw-plugin work               # Shortcut to switch to work
```

### Authentication (Manual)

```bash
bw-plugin login              # Login (uses Keychain or password env var)
bw-plugin login --apikey     # Login with API key (BW_CLIENTID/BW_CLIENTSECRET)
bw-plugin unlock             # Unlock vault, prints BW_SESSION
bw-plugin unlock --raw       # Output session key only (for shell capture)
bw-plugin lock               # Lock vault
bw-plugin logout             # Destroy session
bw-plugin validate           # Check vault status
```

**Session model:** `bw-plugin` never persists session keys to disk. Vault operations auto-unlock on-demand if credentials are available (Keychain or env var). Manual unlock is rarely needed:

```bash
# Only needed if you want BW_SESSION in the current shell
export BW_SESSION=$(bw-plugin unlock --raw)
```

### Vault Operations

```bash
# Search (auto-authenticates as needed)
bw-plugin search "github"              # Search active account
bw-plugin search -a "github"           # Search ALL accounts
bw-plugin search -p api "cloudflare"   # Search specific account

# Inject credentials into a command
bw-plugin inject cloudflare-api -- ./deploy.sh
# Injects: BW_USERNAME, BW_PASSWORD, BW_ITEM_NAME, BW_ITEM_URL, BW_<CUSTOM_FIELDS>

# TOTP
bw-plugin totp "github.com"            # Print TOTP code
bw-plugin totp "github.com" --copy     # Copy to clipboard

# Export with PIN encryption (AES-256-CBC + PBKDF2, 1M iterations)
bw-plugin export -p personal -e -o ~/Backups
bw-plugin decrypt ~/Backups/bw-personal-20260525-120000.enc
```

### Account Aliases (Symlinks)

```bash
bwp get password "github.com"          # Run as personal account
bww get password "legacy-site"         # Run as work account
bwa get password "cloudflare-api"      # Run as API keys account
```

When invoked as `bwp`, `bww`, or `bwa`, the binary automatically targets that account without needing to switch.

### Secrets Manager (bws)

```bash
bw-plugin bws secret list
bw-plugin bws run -- 'npm run dev'     # Defaults to --no-inherit-env
bw-plugin bws run --project-id <id> -- './deploy.sh'
```

### bw serve

```bash
bw-plugin serve start --port 9000      # Start local HTTP API
bw-plugin serve stop                   # Stop server
bw-plugin serve status                 # Check status
```

### Passthrough

Any unrecognized command is passed directly to `bw` with the active account's context:

```bash
bw-plugin get password "github.com"
bw-plugin generate --length 32 --uppercase
bw-plugin sync
```

## Configuration

Accounts are configured in code with these defaults:

| Account | Email | Server | Env Prefix |
|---------|-------|--------|------------|
| personal | `misterme00@icloud.com` | `vault.bitwarden.com` | `BWP` |
| work | `i@mrme0.store` | `nodewarden.hmmr.workers.dev` | `BWW` |
| api | `i@mrme0.store` | `vault.bitwarden.com` | `BWA` |

To customize, create `~/.config/bw-plugin/config.json`:

```json
{
  "accounts": {
    "personal": {
      "name": "Personal Premium",
      "email": "you@example.com",
      "server": "https://vault.bitwarden.com",
      "env_prefix": "BWP",
      "tag": "PREMIUM"
    }
  }
}
```

State (active account only) is stored in `~/.config/bw-plugin/state.json`. **Session keys and credentials are never persisted** — credentials live in macOS Keychain, sessions are derived on-demand.

## Cross-Compilation

```bash
make build-all    # Build for Darwin, Linux, Windows
make build-darwin # macOS (AMD64 + ARM64)
make build-linux  # Linux (AMD64 + ARM64)
make build-windows # Windows AMD64
```

## Security Notes

- **No session persistence:** Session keys are never written to disk. They are derived on-demand via `bw unlock`.
- **Keychain credential storage:** Master passwords and API keys are stored in macOS Keychain (encrypted, access-controlled by OS).
- **Isolated accounts:** Each account has its own `BITWARDENCLI_APPDATA_DIR` — sessions never cross-contaminate.
- **BW_SESSION stripping:** `bwEnv()` strips any existing `BW_SESSION` before running commands, preventing cross-account leakage.
- **Timeout-protected auth:** All non-interactive login attempts have 30s timeouts with stdin closed to prevent 2FA prompt hangs.
- **API key login priority:** API keys are tried before passwords during auto-auth because they bypass device verification.
- **State file** (`~/.config/bw-plugin/state.json`) tracks only `active_account` — no credentials. `chmod 0600`.
- **Account directories** created with `chmod 0700`.
- `bws run` defaults to `--no-inherit-env` to prevent shell env leakage.
- PIN-encrypted exports use AES-256-CBC with PBKDF2 and 1,000,000 iterations.
- Decrypts legacy `openssl enc` exports for backward compatibility.

## Requirements

- `bw` — Bitwarden Password Manager CLI
- `bws` — Bitwarden Secrets Manager CLI (optional)
- macOS Keychain (for `auth setup` credential storage)

## License

Private use — tailored to the author's Bitwarden setup.
