---
name: grant-radar
kind: system
requires: grant-finder CLI tool in PATH (>= 0.1.0)
---

# Grant Radar

### Description

Turn a natural-language startup brief into a deterministic, evidence-backed
Research Packet of ranked non-dilutive funding opportunities — by driving the
public `grant-finder` Go CLI as the deterministic engine. The CLI does the
ledger work, source ingestion, dedupe, FTS5/usearch retrieval, and ranking.
This OpenProse system handles only the agent-side work: turning a human brief
into a Resolved Research Assignment, invoking the CLI, explaining the top
picks, and formatting a human-readable report.

The CLI itself never calls an LLM. All LLM work happens in this system, on the
agent side. That boundary — agent-language in, deterministic ledger work, then
agent-language out — is the point of the example.

### Requires

- `startup_brief`: free-form description of the startup, its technology focus,
  geography, stage, and funding question. Anything an agent would already
  know after a conversation with a founder.

### Ensures

- `research_assignment`: schema-valid Research Assignment JSON, ready to feed
  back into the CLI on later runs without re-resolving the brief
- `research_packet`: the deterministic Research Packet returned by
  `grant-finder research` — ranked grants with evidence, provenance, deadline
  certainty, fit rationale, effort estimate, coverage rows, and negative
  evidence for must-check sources
- `top_pick_explanations`: per-recommendation evidence and provenance for the
  top high-fit grants, returned by `grant-finder explain`
- `markdown_report`: human-readable summary of the packet — for showing the
  founder or pasting into a Notion/Linear doc

### Services

- `resolve-assignment`
- `run-research`
- `explain-top-picks`
- `format-report`

### Invariants

- **No API keys.** This system must run end-to-end with zero third-party API
  credentials beyond whatever the host harness already provides for the BYO
  Prose VM agent itself. The `grant-finder` CLI uses only free public APIs
  (Grants.gov, Federal Register) and public agency RSS feeds. No SAM.gov key,
  no Exa key, no OpenAI/Anthropic keys inside any service, no browser
  automation. If a future service wants to add an API-keyed source, it must
  be opt-in and the system must still run without it.
- **No LLM inside the CLI.** Every service that invokes `grant-finder`
  validates `retrieval.no_llm == true` (or `no_llm == true` on the explain
  packet) before publishing the result. The CLI is the deterministic engine;
  agent judgment lives in `resolve-assignment` and `format-report` only.
- `resolve-assignment` validates output against
  `schemas/research-assignment.schema.json` before publishing it. The CLI
  rejects invalid assignments at the boundary; this system rejects them at
  composition time.
- `run-research` passes the assignment to the CLI via stdin
  (`--assignment -`) and reads JSON from stdout. The system never writes the
  resolved assignment to a shared location it does not control.
- The CLI is invoked with `--refresh auto --semantic auto` by default. The
  system never forces `--include-inactive` unless the brief explicitly asks
  for historical comparable awards.

## Prerequisites

The `grant-finder` Go CLI must be available on `PATH`. Install with one of:

```bash
go install github.com/openprose/grant-finder/cli/grant-finder/cmd/grant-finder@latest
```

Or build from a local clone:

```bash
git clone https://github.com/openprose/grant-finder
cd grant-finder/cli/grant-finder
go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder
```

Confirm the binary resolves:

```bash
grant-finder version
```

Optional: `usearch` on `PATH` enables local semantic retrieval. Without it,
the CLI falls back to SQLite FTS5 automatically. No API keys are required.

### Environment

- `GRANT_FINDER_BIN`: optional override for the `grant-finder` executable path.
  When unset, services resolve `grant-finder` from `PATH`.
- `GRANT_FINDER_DB`: optional path to a persistent SQLite ledger. When unset,
  the CLI uses `~/.local/share/grant-finder/grant-finder.sqlite`. Sharing the
  ledger across runs makes subsequent research packets faster and surfaces
  changes between runs.
