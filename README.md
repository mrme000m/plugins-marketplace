# Plugins Marketplace

A standardized repository for sharing and installing Claude Code plugins and agent skills.

## Installation in Claude Code

To add this repository as a marketplace, run:
```bash
/plugin marketplace add https://github.com/mrme000m/plugins-marketplace
```

To install specific plugins from this marketplace:
```bash
/plugin install bitwarden-vault@plugins-marketplace
```

To install locally for development:
```bash
/plugin install /path/to/plugins-marketplace/plugins/bitwarden-vault
```

## Available Plugins

| Plugin | Category | Description |
|--------|----------|-------------|
| [bitwarden-vault](./plugins/bitwarden-vault) | security | Multi-account Bitwarden password manager with 3 bundled skills: vault operations, `bw` CLI CRUD, and Secrets Manager |
| [email-attachments](./plugins/email-attachments) | security | Hybrid email attachment pipeline: Python extractors (PDF, DOCX, images, Office) normalize files to text + metadata before Claude scores for phishing and routes to inbox |
| [email-downloader](./plugins/email-downloader) | productivity | Modular IMAP email downloader with robust filtering and optional integration with the email-attachments parsing pipeline |
| [email-otp](./plugins/email-otp) | security | Modular email OTP extraction with pluggable IMAP providers (Gmail, iCloud, Outlook) and service-specific templates (Bitwarden, GitHub, Google, AWS) |

### Bundled Skills

#### bitwarden-vault
The `bitwarden-vault` plugin includes the following skills under `plugins/bitwarden-vault/skills/`:

| Skill | Description |
|-------|-------------|
| `bitwarden-vault` | Vault operations: TOTP, secret injection, search, export, account switching |
| `bitwarden-cli` | Full CRUD management via `bw` CLI: items, folders, collections, organizations |
| `bitwarden-secrets-manager` | Full CRUD management via `bws` CLI: secrets, projects, injection |

#### email-attachments
The `email-attachments` plugin includes the following skills under `plugins/email-attachments/skills/`:

| Skill | Description |
|-------|-------------|
| `email-attachments` | Pipeline: type detection, Python/Apryse extraction, metadata/risk envelopes, OCR |
| `phishing-audit` | Security assessment across 7 dimensions: spoofing, brand, urgency, URLs, credentials, signals, entities |
| `inbox-organize` | Classification: document category, priority, action items, folder suggestion |

#### email-downloader
The `email-downloader` plugin includes the following skills under `plugins/email-downloader/skills/`:

| Skill | Description |
|-------|-------------|
| `email-downloader` | Download and filter emails via IMAP, save to markdown/raw EML, and optionally parse attachments |

#### email-otp
The `email-otp` plugin includes the following skills under `plugins/email-otp/skills/`:

| Skill | Description |
|-------|-------------|
| `email-otp` | Fetch one-time passwords from email inboxes via IMAP with pluggable providers and templates |

### Slash Commands

#### bitwarden-vault
| Command | Usage |
|---------|-------|
| `/bw-status` | Show Bitwarden vault status for all configured accounts |
| `/bws-setup` | Interactive setup for Bitwarden Secrets Manager credentials |
| `/bw-help` | Progressive discovery — show all capabilities, skills, and tools |

#### email-attachments
| Command | Usage |
|---------|-------|
| `/parse-attachment` | Parse a single file or directory into normalized artifacts |
| `/attachment-audit` | Run phishing/security audit on parsed artifacts |
| `/attachment-help` | Progressive discovery — show all capabilities, parsers, and skills |

#### email-downloader
| Command | Usage |
|---------|-------|
| `/email-download` | Download emails using IMAP with optional filtering |
| `/email-list` | List available mailboxes or summarize emails in a folder |

#### email-otp
| Command | Usage |
|---------|-------|
| `/email-otp` | Fetch OTP codes from email inboxes by service/provider |

### MCP Server

The plugin bundles an MCP server (`bitwarden-mcp`) that provides vault tools via stdio:
- `bitwarden_status` — Show vault status for all configured accounts
- `bitwarden_search` — Search vault items across accounts
- `bitwarden_get` — Get a specific vault item by ID
- `bitwarden_login` — Login and authenticate to vault
- `bitwarden_unlock` / `bitwarden_lock` — Lock/unlock vault
- `bitwarden_logout` — Logout from vault
- `bitwarden_list_accounts` — List configured accounts

### Security Hooks

The plugin includes runtime security protections:

- **`PreToolUse` credential-guard hook**: Blocks accidental `Edit|Write` operations to credential/session files (accounts.json, keychain, Bitwarden data dirs)
- **`PostToolUse` audit-log hook**: Logs security-relevant `Bash` commands (auth, unlock, export, secret operations) to `~/.local/share/bw-plugin/audit.log`

## Plugin Structure

Each plugin follows the [Claude Code Plugin Format](https://code.claude.com/docs/en/plugins) (Dec 2025):

```
plugins/<plugin-name>/
├── .claude-plugin/
│   └── plugin.json          # Plugin manifest (required)
├── hooks/
│   └── hooks.json           # Optional PreToolUse/PostToolUse hooks
├── skills/
│   └── <skill-name>/
│       └── SKILL.md         # Skill instructions with trigger phrases
├── commands/                # Optional slash commands
├── agents/                  # Optional subagents
├── mcp/                     # Optional MCP server
└── README.md
```

## Contributing Plugins

1. **Create a plugin folder** under `plugins/`.
2. **Add `.claude-plugin/plugin.json`** with the manifest (name, version, description, skills, commands, etc.).
3. **Add your skill(s)** under `skills/<name>/SKILL.md` with YAML frontmatter.
4. **Register**: Add the plugin to `.claude-plugin/marketplace.json`.
5. **Test locally**:
   ```bash
   /plugin install ./plugins/your-plugin
   ```

## Validation Checklist

Before publishing, verify:

- [ ] `plugin.json` is valid JSON with `$schema`
- [ ] All `source` paths in `plugin.json` exist
- [ ] Skills have proper YAML frontmatter (`name`, `description`)
- [ ] `marketplace.json` is valid with `$schema`
- [ ] `README.md` explains installation and usage
- [ ] No secrets or credentials in code

## References

- [Claude Code Plugin Docs](https://code.claude.com/docs/en/plugins)
- [Creating Marketplaces](https://code.claude.com/docs/en/plugin-marketplaces)
- [Agent Skills Spec](https://agentskills.io)
- [Anthropic Skills Repo](https://github.com/anthropics/skills)
