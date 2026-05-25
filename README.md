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
| [bitwarden-vault](./plugins/bitwarden-vault) | security | Multi-account Bitwarden password manager, TOTP, secret injection, vault export |

## Plugin Structure

Each plugin follows the [Claude Code Plugin Format](https://code.claude.com/docs/en/plugins) (Dec 2025):

```
plugins/<plugin-name>/
├── .claude-plugin/
│   └── plugin.json          # Plugin manifest (required)
├── skills/
│   └── <skill-name>/
│       └── SKILL.md         # Skill instructions
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
