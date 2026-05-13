# Grant Radar — OpenProse example

Type a paragraph describing your startup; get back a ranked markdown report
of matching non-dilutive funding opportunities, with sources cited.

**No API keys.** Runs on free public data via the `grant-finder` CLI. The
only LLM cost is whatever your own Prose VM agent uses to translate the brief
and format the report.

## Try it

The example ships with a real sample brief — for polySpectra, an industrial
3D printing materials company — at `fixtures/polyspectra.brief.txt`:

```bash
prose run examples/openprose/src/grant-radar.prose.md \
  --startup_brief "$(cat examples/openprose/fixtures/polyspectra.brief.txt)"
```

From inside this example's directory:

```bash
cd examples/openprose
prose run src/grant-radar.prose.md \
  --startup_brief "$(cat fixtures/polyspectra.brief.txt)"
```

Or pass your own brief:

```bash
prose run src/grant-radar.prose.md \
  --startup_brief "A US small business making <your tech>. Looking for
    non-dilutive R&D funding to <your goal>. Focus areas: <your areas>."
```

You get back four bindings:

- `research_assignment` — schema-valid JSON, reusable for follow-up runs
- `research_packet` — the deterministic CLI output (ranked grants, evidence,
  coverage)
- `top_pick_explanations` — per-recommendation evidence and provenance
- `markdown_report` — human-readable summary for the founder, formatted as
  markdown

## What it does

```
startup_brief (paragraph)
        │
        ▼
resolve-assignment    ← brief → schema-valid Research Assignment JSON
        │
        ▼
run-research          ← `grant-finder research --assignment -` (subprocess)
        │
        ▼
explain-top-picks     ← `grant-finder explain <rec-id>` for each high-fit grant
        │
        ▼
format-report         ← Research Packet + explanations → markdown
        │
        ▼
research_packet + top_pick_explanations + markdown_report
```

## Prerequisites

Two-step install — both are required for this example to wire cleanly:

```bash
# 1. The grant-finder CLI on PATH
go install github.com/openprose/grant-finder/cli/grant-finder/cmd/grant-finder@latest
grant-finder version

# 2. The grant-finder host-harness skill, installed for your agent harness.
#    From this repo:
ln -s "$PWD/skills/grant-finder" ~/.claude/skills/grant-finder
#    Codex:  ~/.codex/skills/grant-finder
#    Gemini: ~/.agents/skills/grant-finder
```

The skill is wired into the OpenProse system via `### Skills: - grant-finder`.
Forme refuses to wire the system if the skill is not installed (fail-closed
via `skill_unresolved`) — so missing the skill produces a clear error at
wiring time, not a half-run failure in the middle of `run-research`.

Optional: `usearch` on `PATH` enables faster semantic retrieval; without it,
the CLI falls back to SQLite FTS5 automatically.

### Sandbox invocation

`prose run` uses the codex-sdk harness by default, which sandboxes the
spawned agent to read-only `$HOME` and blocks outbound network. The CLI
needs both: it writes a SQLite ledger at `~/.local/share/grant-finder/`
and fetches from Grants.gov and the Federal Register. Pick one of the
invocations below depending on your prose version.

**Granular (recommended once your prose CLI supports the env passthrough):**

```bash
PROSE_CODEX_SANDBOX_MODE=workspace-write \
PROSE_CODEX_APPROVAL_POLICY=never \
PROSE_CODEX_ADD_DIR=$HOME/.local/share/grant-finder \
PROSE_CODEX_NETWORK=true \
prose run src/grant-radar.prose.md \
  --startup_brief "$(cat fixtures/polyspectra.brief.txt)"
```

**Fallback (works on any prose version, including 0.13.1):**

```bash
PROSE_CODEX_SANDBOX_MODE=danger-full-access \
PROSE_CODEX_APPROVAL_POLICY=never \
prose run src/grant-radar.prose.md \
  --startup_brief "$(cat fixtures/polyspectra.brief.txt)"
```

The granular form is strictly less broad — it grants only the specific
filesystem path and outbound network access this system declares in its
`### Environment` block. The system file documents the full permission
shape (filesystem.write, network.outbound, exec); Forme does not yet
enforce that schema, but the documentation is forward-compatible.

## Environment

- `GRANT_FINDER_BIN` — optional override for the `grant-finder` executable
  path. Defaults to whatever resolves on `PATH`.
- `GRANT_FINDER_DB` — optional persistent SQLite ledger path. Sharing the
  ledger across runs makes subsequent research packets faster and surfaces
  changes between runs.

## How it's structured

The example demonstrates the OSS-as-give-away pattern: the OSS path is the
whole thing, self-runnable. The `grant-finder` CLI does the deterministic
work (ledger, dedupe, FTS5/usearch retrieval, source-lane coverage); the
OpenProse system handles the agent-side translation between human brief and
structured assignment and formats the report at the end. LLM work is bounded
to two phases: `resolve-assignment` and `format-report`. Everything between
is the CLI.

For the architectural rationale (and the load-bearing constraints behind each
service's `### Shape.prohibited` list) see the parent repo's
[`AGENTS.md`](../../AGENTS.md).

## Hosted version

The OpenProse team operates a hosted version of this exact flow under the
name *Grant Radar*. The hosted service handles source freshness, ingestion
scheduling, monitoring, and reliability so founders never have to look at
the substrate. The OSS version in this repo is the same idea — just operated
by you instead of us. See <https://openprose.ai>.

## License

Same as the parent repository (`grant-finder`): MIT.
