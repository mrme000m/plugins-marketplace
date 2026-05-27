# bw-plugin Multi-Agent Integration Design

**Date:** 2026-05-25
**Status:** Implementation-ready

## Executive Summary

This document defines how `bw-plugin` integrates with AI agent platforms:
- **Claude Code** (Anthropic) — via `SKILL.md` + hooks
- **Codex CLI** (OpenAI) — via MCP server
- **Gemini CLI** (Google) — via MCP server
- **Any MCP-compatible client** (Cursor, Zed, etc.)

## Architecture Decision

| Platform | Integration Method | Rationale |
|----------|-------------------|-----------|
| Claude Code | SKILL.md + MCP server | Native skill triggers + universal tool access |
| Codex CLI | MCP server | Codex adopted MCP as primary tool mechanism |
| Gemini CLI | MCP server | Gemini extensions wrap MCP servers |
| Cursor/Zed/etc | MCP server | Universal compatibility |

**MCP is the common denominator.** All major platforms adopted or are adopting MCP. Building an MCP server first covers 90% of use cases. Platform-specific wrappers (SKILL.md, TS plugin) add conveniences like auto-triggering and UI hooks.

## MCP Server Design

### Transport

- **stdio** (for local CLI integration)

### Tools (token-efficient, focused)

All vault tools support automatic authentication — no separate login/unlock step needed. Credentials are sourced from environment variables → `.env` file → macOS Keychain.

| Tool | Input | Output | Tokens |
|------|-------|--------|--------|
| `bitwarden_status` | `{account?}` | `{account, status, email}` | ~50 |
| `bitwarden_search` | `{account?, query}` | `{items: [{name, username, uri}]}` | ~200 |
| `bitwarden_get` | `{account?, item_name, field?}` | `{value}` | ~50 |
| `bitwarden_login` | `{account?}` | `{status, session}` | ~100 |
| `bitwarden_unlock` | `{account?}` | `{status}` | ~30 |
| `bitwarden_lock` | `{account?}` | `{status}` | ~30 |
| `bitwarden_logout` | `{account?}` | `{status}` | ~30 |
| `bitwarden_list_accounts` | `{}` | `{accounts: [{name, email, server}]}` | ~100 |

**Token efficiency principles applied:**
- Each tool does ONE thing
- Outputs are minimal JSON (no full vault dumps)
- `account` defaults to active account (reduces input tokens)
- Search returns top-K with summary, not full items

### Resource URIs

- `bitwarden://status` — Current account status
- `bitwarden://accounts` — List configured accounts
- `bitwarden://config` — Current configuration

### Prompts

- `bitwarden://prompts/login-guide` — How to login to each account
- `bitwarden://prompts/security` — Security best practices

## Claude Code SKILL.md

### Auto-triggering keywords

The SKILL.md description field targets these conversation patterns:
- "get password", "retrieve credential", "fetch API key"
- "bitwarden", "bw", "vault"
- "TOTP", "2FA code", "authentication code"
- "inject secrets", "env vars from vault"
- "export vault", "backup passwords"

### Hooks

| Hook | Event | Action |
|------|-------|--------|
| `PreToolUse` | Before `Edit\|Write` | Guard credential/session files from accidental writes |
| `PostToolUse` | After `Bash` | Log security-relevant commands to audit log |

## Codex CLI Integration

```toml
# ~/.codex/config.toml
[mcp.servers.bitwarden]
command = ["bw-plugin-mcp"]
# Credentials from macOS Keychain (via bw-plugin auth setup)
# or set env vars as fallback:
# env = { BWP_CLIENTID = "", BWP_CLIENTSECRET = "" }
```

## Token Budget Analysis

| Operation | Old (shell scripts) | New (MCP + auto-auth) | Savings |
|-----------|--------------------|--------------------------|---------|
| Status check | ~500 (full bw status JSON x 3) | ~150 (structured summary) | 70% |
| Search | ~2000 (full items JSON + manual login) | ~200 (auto-auth + top-K summary) | 90% |
| Get password | ~500 (manual unlock + full item JSON) | ~50 (auto-auth + just the value) | 90% |
| Inject | ~1500 (manual login + env dump) | ~200 (auto-auth + env var names) | 87% |
| Auth test | N/A | ~100 (structured per-account results) | New |

## Implementation Priority

1. **P0:** MCP server (stdio transport, 8 tools with auto-auth)
2. **P1:** Updated Claude Code SKILL.md (auto-triggering + hooks)
3. **P2:** Streamable HTTP transport for remote access
