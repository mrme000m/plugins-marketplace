# Bitwarden Vault Plugin — Installation

## Quick Install

```bash
# From this plugin directory
cd src
make install
```

## Manual Install

```bash
# 1. Build the binary
cd src
GOROOT=/opt/homebrew/Cellar/go/$(go version | awk '{print $3}' | sed 's/go//')/libexec \
  go build -ldflags "-s -w" -o bw-plugin .

# 2. Copy to PATH
cp bw-plugin ~/bin/

# 3. Create aliases (symlinks)
ln -sf ~/bin/bw-plugin ~/bin/bwp
ln -sf ~/bin/bw-plugin ~/bin/bww
ln -sf ~/bin/bw-plugin ~/bin/bwa
```

## Requirements

- `bw` — Bitwarden Password Manager CLI (`brew install bitwarden-cli`)
- `bws` — Bitwarden Secrets Manager CLI (optional, for `bws` commands)
- macOS Keychain (for `auth setup` credential storage)
- Go 1.26+ (for building from source)

## First-Time Setup

### 1. Store Credentials

```bash
bw-plugin auth setup
```

This interactively prompts for each account's master password and (optionally) API key credentials, storing them in macOS Keychain. Press Enter to skip any field and keep existing values.

### 2. Login to Each Account

```bash
bw-plugin auth login
```

Performs interactive login for all accounts. If Bitwarden requires device verification (OTP sent to email), this command prompts for the code. After device verification, auto-auth works without manual intervention.

To login a single account:
```bash
bw-plugin auth login personal
```

### 3. Verify

```bash
bw-plugin auth test
```

Tests the full auto-auth flow for all accounts. If successful, shows session keys for each account and confirms vault operations will work.

## Credential Configuration

There are three ways to provide credentials, used in this priority order:

### Option A: macOS Keychain (Recommended)

```bash
bw-plugin auth setup
```

Credentials are stored securely in macOS Keychain with service names like `bw-plugin.personal.password`. No shell configuration needed.

### Option B: Environment Variables (Fallback)

Set password environment variables in your shell profile:

```bash
export BWP_PASSWORD="your-personal-password"
export BWW_PASSWORD="your-work-password"
export BWA_PASSWORD="your-api-vault-password"
```

For API key auth, also set:
```bash
export BWA_CLIENTID="user.xxxx"
export BWA_CLIENTSECRET="xxxx"
```

### Option C: Config File (Account Customization)

To customize accounts, create `~/.config/bw-plugin/config.json`:

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

## Managing Stored Credentials

```bash
bw-plugin auth show               # View what's stored (masked)
bw-plugin auth test               # Test all accounts
bw-plugin auth clean              # Remove all credentials from Keychain
```

## Cross-Compilation

```bash
cd src
make build-all    # Darwin, Linux, Windows
make build-darwin # macOS (AMD64 + ARM64)
make build-linux  # Linux (AMD64 + ARM64)
make build-windows # Windows AMD64
```

## MCP Server (Optional)

For MCP-compatible clients (Cursor, Zed, Codex CLI, etc.):

```bash
cd mcp-server
go build -o bw-plugin-mcp .
cp bw-plugin-mcp ~/bin/
```

Configure in your MCP client:
```json
{
  "mcpServers": {
    "bitwarden": {
      "command": "bw-plugin-mcp"
    }
  }
}
```
