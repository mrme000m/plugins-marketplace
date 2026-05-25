# bw-plugin Multi-Agent Integration Design

**Date:** 2026-05-25
**Status:** Implementation-ready

## Executive Summary

This document defines how `bw-plugin` integrates with AI agent platforms:
- **Claude Code** (Anthropic) — via `SKILL.md` + hooks
- **Codex CLI** (OpenAI) — via MCP server
- **Opencode CLI** — via TS plugin + hooks
- **Gemini CLI** (Google) — via MCP server
- **Any MCP-compatible client** (Cursor, Zed, etc.)

## Architecture Decision

| Platform | Integration Method | Rationale |
|----------|-------------------|-----------|
| Claude Code | SKILL.md + MCP server | Native skill triggers + universal tool access |
| Codex CLI | MCP server | Codex adopted MCP as primary tool mechanism |
| Opencode CLI | TS Plugin + MCP server | Plugin hooks for UI + MCP for tools |
| Gemini CLI | MCP server | Gemini extensions wrap MCP servers |
| Cursor/Zed/etc | MCP server | Universal compatibility |

**MCP is the common denominator.** All major platforms adopted or are adopting MCP. Building an MCP server first covers 90% of use cases. Platform-specific wrappers (SKILL.md, TS plugin) add conveniences like auto-triggering and UI hooks.

## MCP Server Design

### Transport

- **Primary:** `stdio` (for local CLI integration)
- **Secondary:** Streamable HTTP (for remote/IDE integration)

### Tools (token-efficient, focused)

| Tool | Input | Output | Tokens |
|------|-------|--------|--------|
| `bitwarden_status` | `{account?}` | `{account, status, email}` | ~50 |
| `bitwarden_login` | `{account, method}` | `{success, message}` | ~30 |
| `bitwarden_unlock` | `{account}` | `{success, session_key}` | ~30 |
| `bitwarden_search` | `{account?, query}` | `{items: [{name, username, uri}]}` | ~200 |
| `bitwarden_get` | `{account?, item_name, field?}` | `{value}` | ~50 |
| `bitwarden_totp` | `{account?, item_name}` | `{code}` | ~30 |
| `bitwarden_inject` | `{account?, item_name}` | `{env_vars}` | ~100 |
| `bitwarden_export` | `{account?, encrypt?}` | `{file_path}` | ~50 |

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
- "bitwarden", "bw", "bwc", "vault"
- "TOTP", "2FA code", "authentication code"
- "inject secrets", "env vars from vault"
- "export vault", "backup passwords"

### Hooks

| Hook | Event | Action |
|------|-------|--------|
| `SessionStart` | New session | Load active account, show status |
| `PreToolUse` | Before file write | If path looks like `.env`, warn about secrets |
| `Stop` | Session end | Lock all vaults |

## Codex CLI Integration

```toml
# ~/.codex/config.toml
[mcp.servers.bitwarden]
command = ["bw-plugin-mcp"]
env = { BWP_PASSWORD = "", BWW_PASSWORD = "", BWA_PASSWORD = "" }
```

## Opencode CLI Plugin

```typescript
// .opencode/plugins/bw-plugin/index.ts
export function bwPlugin() {
  return {
    tool: {
      "execute.before": ({ input }) => {
        if (input.tool === "write" && input.args.path?.endsWith('.env')) {
          console.warn("⚠️ Writing to .env — consider using bw-plugin inject instead");
        }
      },
    },
    session: {
      start: () => {
        // Auto-check vault status on session start
      },
    },
  };
}
```

## Token Budget Analysis

| Operation | Old (shell scripts) | New (MCP + focused tools) | Savings |
|-----------|--------------------|--------------------------|---------|
| Status check | ~500 (full bw status JSON × 3) | ~150 (structured summary) | 70% |
| Search | ~2000 (full items JSON) | ~300 (top-K summary) | 85% |
| Get password | ~500 (full item JSON) | ~50 (just the value) | 90% |
| Inject | ~1500 (env dump + item JSON) | ~200 (just env var names) | 87% |

## Implementation Priority

1. **P0:** MCP server (stdio transport, 8 tools)
2. **P1:** Updated Claude Code SKILL.md (auto-triggering + hooks)
3. **P2:** Opencode TS plugin
4. **P3:** Streamable HTTP transport for remote access
