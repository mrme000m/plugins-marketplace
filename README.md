# Plugins Marketplace

A standardized repository for sharing and installing agent skills.

## Installation in Claude Code

To add this repository as a marketplace, run:
```bash
/plugin marketplace add <this-repo-url>
```

To install specific plugins from this marketplace:
```bash
/plugin install <plugin-name>@plugins-marketplace
```

## Available Skills

| Skill | Description | Trigger Keywords |
|-------|-------------|------------------|
| [bitwarden-vault](./skills/bitwarden-vault) | Multi-account Bitwarden password manager, TOTP, secret injection, vault export | `bitwarden`, `password`, `TOTP`, `2FA`, `inject secrets`, `vault`, `bw` |

## Contributing Skills

1.  **Create a skill folder**: Create a new folder in `/skills/`.
2.  **Add SKILL.md**: Copy the contents from `/template/SKILL.md` and customize the YAML frontmatter and instructions.
3.  **Register**: Add the path to your skill in `.claude-plugin/marketplace.json`.

## Agent Standards
This repository follows the [Agent Skills](https://agentskills.io) specification. AI agents should refer to `CLAUDE.md` for specific mandates and workflows.
