# Current CLI Status

Date: 2026-05-08

The Go CLI in `cli/grant-finder` now has the initial agent-facing product
surface:

- `research` accepts a resolved Research Assignment JSON and returns a Research
  Packet.
- `explain` returns deterministic evidence and provenance for a recommendation
  or opportunity.
- `status` reports ledger freshness and assignment source-lane coverage.
- Source plumbing is under `debug`, not exposed as the headline interface.

It is still not accepted as ship-ready.

## Why

The original command tree exposed deterministic source machinery as user-facing
workflow:

- `sync`
- `search`
- `sources smoke`
- `feeds smoke`
- `grants xml`
- `federal-register hydrate`
- `sql`
- `export opml`

Those capabilities now belong under `debug` or behind automatic refresh. The
product interface is agent-facing Grant Deep Research. The public commands now
center on:

- `research`
- `explain`
- `status`

## Current Acceptance Results

Passing:

- `make validate`
- `make validate-product-cli`
- `make dogfood-agent`
- `printing-press workflow-verify --dir cli/grant-finder --json`

Behaviorally verified:

- The deep-tech-startup fixture produces a Research Packet with ranked opportunities,
  evidence, deadline certainty, application outlines, source coverage, and
  ARPA-E negative evidence.
- Default `research` output filters out closed, archived, expired, and
  past-due records. `--include-inactive` keeps those records available for
  historical comps.
- `--semantic usearch` is accepted and falls back to FTS5 when the local
  `usearch` corpus is not indexed.
- `--select` and `--compact` narrow agent output without model calls.
- A live `debug sync --grants-only --keyword SBIR --limit 1` probe ingested one
  Grants.gov API record into a fresh ledger.

Remaining hold reasons:

- `printing-press dogfood` still warns that the sync/search pipeline does not
  look like a canonical generated API CLI and that its static workflow mapper
  cannot understand fixture paths or extracted workflow variables.
- `printing-press scorecard` is still low because this is an agent substrate
  rather than a spec-generated API wrapper with MCP surface, endpoint examples,
  and standard generated cache patterns.
- Grants.gov API ingestion exists, feed-derived Federal Register links are
  hydrated during sync, and bounded Grants.gov XML extract ingestion has been
  ported from the Python POC into the Go collector path.

## Rework Gate

Before ship acceptance, the implementation must be checked against:

- `CONTEXT.md`
- `docs/adr/0001-agent-facing-grant-deep-research.md`
- `printing-press/research/grant-finder-brief.md`
- `printing-press/research/grant-finder-absorb-manifest.md`
- `printing-press/product-surface.json`

The current CLI can be mined for useful code: source manifests, SQLite store
patterns, `usearch` corpus generation, FTS5 fallback, read-only command helpers,
and POC parity work. The behavioral Research Packet fixture now passes, but ship
acceptance remains blocked on the remaining hold reasons above.
