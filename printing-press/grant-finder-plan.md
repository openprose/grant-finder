# Grant Finder CLI Implementation Plan

## Product Thesis

Grant Finder is an agent-facing Grant Deep Research substrate for OpenProse.
It helps an upstream agent answer a specific funding-opportunity request for a
specific startup by reusing deterministic collectors, a local Opportunity
Ledger, source provenance, local semantic retrieval, FTS5 fallback, and prior
source coverage.

The CLI is not for humans. It is the boundary between a Resolved Agent Request
and deterministic background research machinery.

## Core Insight

The product is not a grant scraper, a feed reader, or a source-maintenance
console. The useful interface is a **Research Packet compiler** over a
provenance-first Opportunity Ledger.

The upstream agent already knows the customer/startup context. Grant Finder
should not recreate intake. Instead, it receives an assignment payload and does
the deterministic work the agent should not redo: refresh stale sources, search
the ledger, reconcile duplicates, resolve weak records, check must-cover lanes,
rank candidates, and return evidence-backed recommendations.

The upstream LLM uses the CLI; the CLI itself does not call an LLM. Semantic
retrieval should use local `usearch` over the opportunity/evidence corpus, with
SQLite FTS5 as fallback.

Anything deterministic should run automatically or live behind debug tooling.
Commands such as Federal Register hydration and Grants.gov XML ingestion are
background capabilities, not headline product commands.

## Ground Truth Inputs

- Domain context: `CONTEXT.md`
- ADR: `docs/adr/0001-agent-facing-grant-deep-research.md`
- Product surface: `printing-press/product-surface.json`
- Printing Press brief: `printing-press/research/grant-finder-brief.md`
- Absorb manifest: `printing-press/research/grant-finder-absorb-manifest.md`

## Agent-Facing Commands

### `research`

Primary command. Accepts a Research Assignment as JSON via `--assignment` or
stdin and returns a Research Packet.

Required behavior:

- automatically refresh stale deterministic source lanes unless disabled
- retrieve candidates with `usearch` when available, then supplement with FTS5
- exclude known grants from the assignment
- expand focus areas and target geographies into source coverage expectations
- rank opportunities by urgency, eligibility fit, then amount
- include evidence, provenance, deadline certainty, effort estimate, and next action
- include application outlines for high-fit opportunities
- include negative evidence for must-check sources
- support `--json`, `--compact`, `--limit`, and eventually `--select`

### `explain`

Returns the evidence and provenance trail for one recommendation or opportunity.

Required behavior:

- show raw source observations and normalized opportunity fields
- show fit rationale and deadline certainty
- show which sources corroborated the record
- show why known-grant or duplicate logic did or did not apply

### `status`

Reports ledger freshness and source coverage for an optional assignment.

Required behavior:

- show last refresh by source lane
- show missing/stale required lanes for the assignment
- show whether must-check sources have current positive or negative evidence
- support read-only JSON output

## Background Capabilities

These may be implemented as internal packages, scheduled jobs, or debug
commands. They are not the product interface.

- source refresh
- feed/page source parsing
- Grants.gov XML ingestion and Applicant API lookup
- Federal Register hydration
- source/feed smoke checks
- `usearch` semantic retrieval over the local opportunity/evidence corpus
- FTS5 ledger fallback search
- read-only SQL inspection
- OPML export
- change detection

## Data Model

The SQLite store needs these tables:

- assignments: saved Research Assignment payloads and hashes
- runs: one row per refresh/research run
- raw_items: fetched source/feed/API items before normalization
- opportunities: deduped normalized opportunity records
- opportunity_sources: source provenance links
- evidence: source-backed evidence snippets and fields
- fit_assessments: assignment-specific eligibility fit and rationale
- research_packets: emitted packet metadata and summary
- changes: first-seen/updated change records
- opportunity_search: FTS5 fallback over titles, agencies, summaries, eligibility, geography, technology signals, deadlines, URLs, and source IDs
- opportunity corpus files: local markdown/json documents for `usearch` indexing

## First Robust Build Scope

1. Validate the product surface and assignment/packet contracts.
2. Port POC parity into Go for ledger, provenance, FTS5, dedupe, Grants.gov XML/API, Federal Register resolution, and idempotency.
3. Build `research` with automatic refresh and a sample deep-tech-startup acceptance fixture.
4. Build `explain` and `status`.
5. Demote current source-operation commands to debug/internal paths or remove them from the public command tree.

## Deferred

- SAM.gov ingestion until API-key access is explicit.
- Mailbox/listserv ingestion until an approved monitored inbox exists.
- Commercial grant database extraction until access and terms are explicit.
- Broad state/local source-specific parsers beyond what is needed for the first acceptance fixture.

## Verification Targets

Required fast checks:

- `make validate`
- `make validate-product-cli` once the product CLI is regenerated or reworked
- `go test ./...` inside `cli/grant-finder`

Required behavioral checks before ship:

- sample deep-tech-startup assignment returns a valid Research Packet
- Research Packet includes at least one ranked opportunity when fixtures contain matches
- Research Packet includes negative evidence for ARPA-E when no match exists
- known grants are excluded
- repeated refresh is idempotent
- `explain` returns provenance for a recommendation
- `status` reports source coverage and freshness

Live verification remains read-only. No command applies, submits, emails, posts,
registers, purchases, or mutates upstream services.
