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
- Go 1.22+ (for building from source)

## First-Time Setup

### 1. Store Credentials

```bash
bw-plugin auth setup
```

This interactively prompts for each account's API key credentials (Client ID + Client Secret) and master password, storing them in macOS Keychain. Press Enter to skip any field and keep existing values.

**Get your API key at:** vault.bitwarden.com → Settings → My Account → API Key

### 2. Login to Each Account

```bash
bw-plugin auth login
```

Performs API key login for all accounts. No device verification needed — API keys bypass that entirely.

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

Credentials are resolved in this priority order: environment variables → `.env` file → macOS Keychain.

### Option A: Environment Variables (Recommended for Agents)

Set environment variables in your shell profile:

```bash
export BWP_CLIENTID="user.xxxx"
export BWP_CLIENTSECRET="xxxx"
```

For API key auth:
```bash
export BW_CLIENTID="user.xxxx"
export BW_CLIENTSECRET="xxxx"
```

### Option B: `.env` File

Create `~/.config/bw-plugin/.env`:

```bash
BWP_CLIENTID=user.xxxx
BWP_CLIENTSECRET=xxxx
```

### Option C: macOS Keychain (Interactive Setup)

```bash
bw-plugin auth setup
```

Credentials are stored securely in macOS Keychain with service names like `bw-plugin.account.<id>.client_id`. No shell configuration needed.

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
