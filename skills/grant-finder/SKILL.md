---
name: grant-finder
description: Use the `grant-finder` CLI to return source-cited candidate non-dilutive funding opportunities for a research lab, startup, nonprofit research group, or technical team. Returns evidence-backed Research Packets and per-candidate provenance. Triggers on `grant-finder research`, `grant-finder explain`, `grant-finder status`, or any need to find federal-grant, SBIR/STTR, topic-page, or public-source funding opportunities matching an organization brief.
---

# grant-finder

A deterministic Go CLI that turns a Research Assignment JSON into an
evidence-backed candidate list of non-dilutive funding opportunities. The CLI
does not call an LLM; it operates on a local SQLite ledger refreshed from
Grants.gov, the Federal Register, public agency RSS feeds, and configured
public source pages. Final ranking and recommendation judgment belong to the
calling agent.

## When to use

- An upstream task gave you a lab, startup, nonprofit, or technical-project
  brief, or a Research Assignment JSON, and you need to surface matching funding
  opportunities.
- You need source-cited evidence for one candidate (`explain`).
- You need to check ledger freshness or source-lane coverage (`status`).

## Public commands

```bash
# Return candidate opportunities for a Research Assignment
grant-finder research --assignment <path|-> --json

# Show evidence and provenance for one candidate
grant-finder explain <recommendation-id|opportunity-id> --json

# Report ledger freshness and source-lane coverage
grant-finder status --assignment <path> --json

# Self-check: SQLite/FTS5/usearch availability + manifest counts
grant-finder doctor --json
```

## Common flags

```
--db <path>            SQLite ledger path (default ~/.local/share/grant-finder/grant-finder.sqlite)
--json                 JSON output (default for agent flows)
--compact              Compact JSON
--select <fields>      Project top-level paths (comma-separated, dot-pathed)
--limit N              Result count (default 10 for research)
--refresh auto|off     Refresh stale source lanes before answering. Default: auto
--semantic auto|usearch|off   Default: auto (usearch when available, FTS5 fallback)
--include-inactive     Include closed/archived/past-due records (off by default)
```

## Notes

- Pipe the assignment JSON via stdin with `--assignment -` rather than writing
  to a temp file when possible.
- Always verify `retrieval.no_llm == true` (or `no_llm: true` on explain) in
  the response — drift guard against future regressions.
- The CLI is deterministic; the same assignment against the same ledger
  produces the same packet.

## Where to read more

- Top-level README: <https://github.com/openprose/grant-finder/blob/main/README.md>
- Contributor docs: <https://github.com/openprose/grant-finder/blob/main/AGENTS.md>
- Schemas: `schemas/research-assignment.schema.json`, `schemas/research-packet.schema.json` in the repo.

## Install

```bash
# Binary
go build -o "$HOME/.local/bin/grant-finder" ./cli/grant-finder/cmd/grant-finder

# Skill (this file)
ln -s "$PWD/skills/grant-finder" ~/.claude/skills/grant-finder
# (or your harness's equivalent skills dir for Codex / Gemini)
```
