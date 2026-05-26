# Plugin Template

A starter template for creating Claude Code plugins following the official Anthropic format (Dec 2025).

## Usage

1. **Copy this template** into a new folder under `plugins/<your-plugin-name>/`
2. **Edit `plugin.json`** — set name, version, description, and list your skills/commands/agents
3. **Write your skill(s)** in `skills/<skill-name>/SKILL.md` with YAML frontmatter
4. **Add commands** (optional) in `commands/<command-name>.md` with YAML frontmatter
5. **Add hooks** (optional) in `hooks/hooks.json` for PreToolUse/PostToolUse guards
6. **Add agents** (optional) in `agents/<agent-name>.md` for specialized subagents
7. **Register** the plugin in `.claude-plugin/marketplace.json` at repo root
8. **Test locally**:
   ```bash
   /plugin install /path/to/plugins-marketplace/plugins/<your-plugin-name>
   ```

## File Descriptions

| File | Purpose |
|------|---------|
| `plugin.json` | Plugin manifest — required. Defines name, version, skills, commands, agents, MCP servers, hooks |
| `SKILL.md` | Skill instructions with YAML frontmatter (`name`, `description`, trigger phrases) |
| `my-command.md` | Slash command definition with YAML frontmatter (`name`, `description`) |
| `hooks.json` | Pre/post tool execution guards (security, audit, validation) |
| `my-agent.md` | Specialized subagent definition with model and tool restrictions |

## Best Practices

### Progressive Discoverability
- Include trigger phrases in skill `description` so Claude knows when to invoke
- Create a `/help` command listing all capabilities
- Use clear, specific descriptions (not vague summaries)

### Security
- Use `PreToolUse` hooks to guard sensitive files
- Use `PostToolUse` hooks to audit security-relevant commands
- Never store secrets in plugin files — use environment variables

### Single Responsibility
- One skill ≈ one domain of knowledge
- One command ≈ one atomic action
- One agent ≈ one specialized role
