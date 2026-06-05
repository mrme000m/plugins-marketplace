---
name: research-deep-dive
description: Use this skill when the user asks for thorough Perplexity research, current best practices, comparative analysis, technical investigation, deep research, source-grounded answers, Space-scoped research, or follow-up chains. Triggers on "research thoroughly", "deep dive", "latest", "current state", "compare", "investigate", "source-grounded", "Perplexity research".
version: 1.2.0
---

# Research Deep Dive

Run grounded, cost-aware research with Perplexity while preserving context window budget.

## Default Workflow

1. Define the exact question and what would count as an answer.
2. Check whether the answer should come from local files first.
3. Look for a relevant Perplexity Space for project-specific context.
4. Search in `auto` mode unless the user requests deeper/paid modes.
5. Chain follow-ups with `backend_uuid` instead of repeating searches.
6. Synthesize findings, uncertainty, and next research options.
7. Offer to upload the synthesis to a Space when it will be reused.

## Mode Discipline

- `auto`: default for quick facts, version checks, docs verification.
- `pro`: user asks for cited/deeper source analysis.
- `reasoning`: user asks for step-by-step reasoning.
- `deep_research`: user explicitly requests an exhaustive report.

## Query Shape

Use precise queries:

`[library/product] [version/date] [API/topic] [specific question]`

Bad: `react hooks`
Good: `React 19 useActionState API signature server action pending state current docs`

## Output Contract

Return:
- Short answer
- Key findings
- Evidence/source notes when available
- Confidence and caveats
- `backend_uuid`/thread slug if useful
- Suggested follow-ups

## CLI Fallback

Use the `cli-integration` skill and scripts when MCP is unavailable.

```bash
pplx search "<query>" --mode auto
pplx search "<query>" --mode pro --raw
pplx follow-up "<question>" <backend_uuid>
```
