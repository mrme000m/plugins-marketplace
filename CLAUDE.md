# Agent Instructions: Plugins Marketplace

This file serves as the primary source of truth for all AI agents (Claude, Gemini, OpenCode, etc.) interacting with this repository.

## Project Vision
To provide a modular, standardized marketplace for "Agent Skills" following the `agentskills.io` specification.

## Marketplace Architecture
- **`.claude-plugin/marketplace.json`**: The core registry. All new plugins and skills MUST be registered here to be discoverable by Claude Code.
- **`skills/`**: Each subdirectory is a standalone skill.
- **`template/`**: Contains the `SKILL.md` boilerplate for new skills.

## Core Mandates for Agents

### 1. Creating New Skills
- **Directory**: Always create a new folder under `skills/`.
- **Metadata**: Every skill MUST have a `SKILL.md` file with valid YAML frontmatter containing `name` and `description`.
- **Validation**: Ensure the `name` is lowercase with hyphens.
- **Registration**: After creating a skill folder, you MUST add its relative path (e.g., `"./skills/my-new-skill"`) to the appropriate plugin group in `.claude-plugin/marketplace.json`.

### 2. Marketplace Maintenance
- **Integrity**: Never delete the `.claude-plugin` directory or its contents without explicit user directive.
- **Versioning**: Increment the version in `marketplace.json` when adding or removing plugins/skills.
- **Documentation**: Update the root `README.md` if the installation or discovery process changes.

### 3. Agent Interoperability
- Use `CLAUDE.md` as the master instruction set.
- Respect symlinks (`GEMINI.md`, `OPENCODE.md`) that point to this file.

## Quality Standards
- **Skill Instructions**: Instructions inside `SKILL.md` should be concise, imperative, and focused on behavioral logic.
- **Examples**: Always include at least two concrete examples in `SKILL.md`.
- **Guidelines**: Specify constraints clearly to prevent agent hallucination or misuse of tools.

## Distribution
Users can install plugins from this repository using:
```bash
/plugin marketplace add <repo-url>
/plugin install <plugin-name>@<marketplace-name>
```
