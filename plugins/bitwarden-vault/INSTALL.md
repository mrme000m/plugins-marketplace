# Bitwarden Vault Plugin — Installation

## Quick Install

```bash
# From this skill directory
cd skills/bitwarden-vault
make install
```

## Manual Install

```bash
# 1. Build the binary
cd src
go build -o bw-plugin .

# 2. Copy to PATH
cp bw-plugin ~/bin/

# 3. Create aliases (symlinks)
ln -s ~/bin/bw-plugin ~/bin/bwp
ln -s ~/bin/bw-plugin ~/bin/bww
ln -s ~/bin/bw-plugin ~/bin/bwa
```

## Requirements

- `bw` — Bitwarden Password Manager CLI (`brew install bitwarden-cli`)
- `bws` — Bitwarden Secrets Manager CLI (optional, for `bws` commands)
- Go 1.21+ (for building from source)

## Configuration

Set password environment variables for non-interactive unlock:

```bash
export BWP_PASSWORD="your-personal-password"
export BWW_PASSWORD="your-work-password"
export BWA_PASSWORD="your-api-vault-password"
```

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