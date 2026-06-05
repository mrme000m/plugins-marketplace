---
name: pplx-research
aliases: [perplexity-research, pplx-deep-research, p-research]
description: Execute grounded Perplexity research with mode discipline, Space-aware search, follow-up chaining, and CLI fallback
allowed-tools: [mcp__perplexity__search, mcp__perplexity__follow_up, mcp__perplexity__list_spaces, mcp__perplexity__search_in_space, mcp__perplexity__list_space_files, Bash]
argument-hint: "<query> [--mode auto|pro|reasoning|deep_research] [--space <name>] [--sources web,scholar,social]"
---

# /pplx-research

Run a Perplexity-backed research workflow with safe defaults.

## Decision Tree

1. **Local codebase question?** Read local files first; do not search the web for facts present in the repo.
2. **Current/external fact needed?** Use Perplexity.
3. **Project-specific knowledge exists?** List Spaces and prefer `search_in_space`.
4. **No Space match?** Use general search.
5. **Need more detail?** Use `follow_up` with `backend_uuid`; do not repeat the same search.

## Mode Rules

| User intent | Mode |
|---|---|
| Quick fact, current version, docs check | `auto` |
| User asks for cited/deeper analysis | `pro` |
| User asks for step-by-step reasoning | `reasoning` |
| User explicitly asks for exhaustive report/deep research | `deep_research` |

Never silently escalate to a paid mode. Ask or clearly note the mode change.

## MCP Workflow

1. Enhance vague queries with library, version, API name, error text, and project context when available.
2. Use `answer_only: true` for quick verification and follow-ups.
3. Use `answer_only: false` only when the user needs sources/citations.
4. Capture and report `backend_uuid` when useful for follow-up continuity.
5. Present findings as: answer, key evidence, confidence, risks/unknowns, follow-up options.

## CLI Fallback

If MCP tools are unavailable:

```bash
pplx search "<query>" --mode auto
pplx search "<query>" --mode pro --raw
pplx follow-up "<question>" <backend_uuid>
```

Use `bash <plugin>/scripts/pplx-health.sh --no-search` before diagnosing auth issues.
