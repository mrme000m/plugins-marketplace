---
name: bw-help
description: Show all available Bitwarden plugin capabilities, commands, skills, and MCP tools in a progressive discovery menu. Use when the user asks "what can you do", "help", "how do I", "capabilities", "commands", or needs to discover available Bitwarden features.
---

# bw-help Command

When the user runs `/bw-help`, `/help bitwarden`, or asks "what can you do" / "help" / "how do I", show a progressive discovery menu of all Bitwarden plugin capabilities.

## Output Format

```
Bitwarden Vault Plugin — Capabilities
======================================

Quick commands:
  /bw-status    — Show vault status for all accounts
  /bws-setup    — Set up Secrets Manager credentials
  /bw-help      — Show this help menu

[1] Vault Operations (SKILL: bitwarden-vault)
    — TOTP generation, credential injection, vault search, export
    — Multi-account switching, API key authentication
    — Trigger: "TOTP", "inject password", "search vault", "export"

[2] CLI CRUD (SKILL: bitwarden-cli)
    — Create, edit, delete vault items, folders, collections
    — Trigger: "create item", "edit folder", "delete password"

[3] Secrets Manager (SKILL: bitwarden-secrets-manager)
    — Manage projects, secrets, access tokens via bws CLI
    — Trigger: "create secret", "list projects", "inject env"

[MCP Server] bitwarden-mcp
    — 8 programmatic tools: status, search, get, login, unlock, lock, logout, list_accounts

Setup:
    bw-plugin auth setup      # Store API key + password in Keychain
    bw-plugin auth login      # Login with API key
    bw-plugin auth test       # Verify auto-auth
```

## Progressive Discovery Logic

1. If the user says "help", "what can you do", "commands": Show the full menu above.
2. If the user says "TOTP", "auth code", "2FA": Suggest Vault Operations ([1]).
3. If the user says "create", "edit", "delete", "CRUD": Suggest CLI CRUD ([2]).
4. If the user says "secret", "project", "bws": Suggest Secrets Manager ([3]).
5. If the user says "programmatic", "API", "tool": Suggest MCP Server.

Always link specific capabilities back to the skill or command that handles them.
