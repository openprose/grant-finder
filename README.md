# Grant Radar — OpenProse Example

An OpenProse system that turns a natural-language startup brief into a
deterministic, evidence-backed Research Packet of ranked non-dilutive funding
opportunities — by driving the public `grant-finder` Go CLI as the
deterministic engine.

The example demonstrates the OSS-then-hosted pattern: this is the whole thing,
self-runnable. The `grant-finder` CLI does the ledger work, source ingestion,
dedupe, FTS5/usearch retrieval, and ranking; the OpenProse system handles the
agent-side translation between human brief and structured assignment, and
formats a human-readable report at the end.

## What it does

```
startup_brief
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

## Why it's structured this way

- **The CLI is the engine.** The deterministic, replayable, no-LLM work
  (ledger, dedupe, FTS5/usearch retrieval, source-lane coverage, ARPA-E
  negative evidence) is the CLI's job. This system never re-implements it.
- **LLM work is bounded to two phases.** `resolve-assignment` translates a
  brief into structured JSON; `format-report` renders the structured result
  into markdown. Everything between those phases is the CLI.
- **The OSS path is the whole thing.** This example does not gate features
  behind a tier; it gives away the working flow. The hosted upsell is
  operational — keeping source manifests current, refreshing on a schedule,
  reacting when a feed parser breaks — not feature parity.

## Prerequisites

**Required.** The `grant-finder` Go CLI must be on `PATH`:

```bash
go install github.com/openprose/grant-finder/cli/grant-finder/cmd/grant-finder@latest
grant-finder version
```

**Optional.** `usearch` on `PATH` enables local semantic retrieval; without
it, the CLI falls back to SQLite FTS5 automatically. No API keys required —
the CLI runs entirely on free public APIs (Grants.gov, Federal Register) and
public agency RSS feeds.

## No API keys

This example runs end-to-end with **zero third-party API credentials** beyond
whatever the host harness already provides for the BYO Prose VM agent itself.
The `grant-finder` CLI uses only free public APIs (Grants.gov, Federal
Register) and public agency RSS feeds. No SAM.gov key, no Exa key, no
OpenAI/Anthropic keys inside any service, no browser automation. See the
top-level system's `### Invariants` for the load-bearing constraint.

## Running it

The only required input is `--startup_brief`. A real sample brief — for
polySpectra, an industrial 3D printing materials company — is checked in at
`fixtures/polyspectra.brief.txt`:

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

Or pass your own brief inline:

```bash
prose run src/grant-radar.prose.md \
  --startup_brief "A US small business making <your tech>. Looking for
    non-dilutive R&D funding to <your goal>. Focus areas: <your areas>."
```

The system produces:

- `research_assignment` — schema-valid JSON, reusable for follow-up runs
- `research_packet` — the deterministic CLI output
- `top_pick_explanations` — per-recommendation evidence/provenance
- `markdown_report` — human-readable summary for the founder

## Environment

- `GRANT_FINDER_BIN`: optional override for the `grant-finder` executable
  path. Defaults to whatever resolves on `PATH`.
- `GRANT_FINDER_DB`: optional path to a persistent SQLite ledger. Sharing the
  ledger across runs makes subsequent research packets faster and surfaces
  changes between runs.

## Hosted Service

The OpenProse team operates a hosted version of this exact flow under the
name *Grant Radar*. The hosted service handles source freshness, ingestion
scheduling, monitoring, and reliability so founders never have to look at
the substrate. The OSS version in this repo is the same idea — just operated
by you instead of us.

## License

Same as the parent repository (`grant-finder`): MIT.
