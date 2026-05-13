---
name: grant-radar
kind: system
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

### Skills

- grant-finder

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

This system declares the `grant-finder` host-harness skill (see `### Skills`
above). Forme refuses to wire the system if the skill is not installed,
returning `skill_unresolved` before any service runs. That guarantees the
dependency is satisfied at wiring time rather than failing mid-run.

**Two-step install:**

```bash
# 1. Build the grant-finder CLI binary onto PATH
git clone https://github.com/openprose/grant-finder
cd grant-finder/cli/grant-finder
go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder

# 2. Install the grant-finder host-harness skill (symlink from the repo)
cd ../..
ln -s "$PWD/skills/grant-finder" ~/.claude/skills/grant-finder
#   Codex: ~/.codex/skills/grant-finder
#   Gemini / other agent harnesses: ~/.agents/skills/grant-finder
```

Confirm both halves are wired:

```bash
grant-finder version
ls -l ~/.claude/skills/grant-finder/SKILL.md
```

Optional: `usearch` on `PATH` enables local semantic retrieval. Without it,
the CLI falls back to SQLite FTS5 automatically. No API keys are required.

### Sandbox invocation

The default `codex-sdk` harness for `prose run` sandboxes the spawned agent
to a read-only `$HOME` and blocks outbound network. The CLI cannot run
under those defaults — it needs to create its SQLite ledger under
`~/.local/share/grant-finder/` and reach public APIs (Grants.gov, Federal
Register, agency RSS).

**Recommended (granular permissions** — requires
[openprose/prose#78](https://github.com/openprose/prose/pull/78) (or any
prose release that includes it) for the `PROSE_CODEX_ADD_DIR` /
`PROSE_CODEX_NETWORK` env-passthrough):

```bash
PROSE_CODEX_SANDBOX_MODE=workspace-write \
PROSE_CODEX_APPROVAL_POLICY=never \
PROSE_CODEX_ADD_DIR=$HOME/.local/share/grant-finder \
PROSE_CODEX_NETWORK=true \
prose run examples/openprose/src/grant-radar.prose.md \
  --startup_brief "$(cat examples/openprose/fixtures/polyspectra.brief.txt)"
```

**Fallback (no sandbox** — works on any prose version, including 0.13.1):

```bash
PROSE_CODEX_SANDBOX_MODE=danger-full-access \
PROSE_CODEX_APPROVAL_POLICY=never \
prose run examples/openprose/src/grant-radar.prose.md \
  --startup_brief "$(cat examples/openprose/fixtures/polyspectra.brief.txt)"
```

The granular form is strictly less broad — it grants only the specific
filesystem path and outbound network access this system declares in
`### Environment`. Use it as soon as your prose CLI supports the
passthrough env vars.

### Environment

**Env vars (read by the system services):**

- `GRANT_FINDER_BIN`: optional override for the `grant-finder` executable path.
  When unset, services resolve `grant-finder` from `PATH`.
- `GRANT_FINDER_DB`: optional path to a persistent SQLite ledger. When unset,
  the CLI uses `~/.local/share/grant-finder/grant-finder.sqlite`. Sharing the
  ledger across runs makes subsequent research packets faster and surfaces
  changes between runs.

**Host harness sandbox requirements (soft documentation today; intended to
become Forme-enforced):**

The `grant-finder` CLI requires permissions that a stock sandboxed Prose VM
agent run does not grant by default. Forme today does not parse a structured
permission schema here — the entries below are documentation. Once Forme
gains the schema, they should become enforced.

```
filesystem.write:
  - ~/.local/share/grant-finder/    # SQLite ledger
  - ${GRANT_FINDER_DB%/*}/          # if GRANT_FINDER_DB is set, its parent

network.outbound:
  - api.grants.gov                  # Grants.gov search + fetchOpportunity
  - www.federalregister.gov         # Federal Register document hydration
  - grants.gov                      # XML bulk extract page
  # plus the per-feed RSS URLs declared in cli/grant-finder/internal/grantfinder/data/feeds.json

exec:
  - grant-finder                    # the CLI binary itself
```

If the host harness does not grant these (e.g., the default codex-sdk
sandbox), the run reaches `run-research` and stalls trying to create the
ledger or fetch sources. See `## Prerequisites` for the sandbox invocation.
