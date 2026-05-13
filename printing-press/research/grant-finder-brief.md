# Grant Finder CLI Brief

## API Identity

- Domain: non-dilutive funding opportunity research for startups and technical companies.
- Users: upstream OpenProse agents that already hold customer/startup context.
- Data profile: multi-source funding records, raw evidence, source provenance, changes, FTS5 indexes, fit assessments, and compiled research packets.

## Product Insight

Grant Finder is not a scraper, a feed reader, or a human console for operating
source plumbing. It is a reusable Grant Deep Research substrate for agents.

The durable insight is that agents should call one interface with a resolved
startup/context/technology assignment and receive a source-backed Research
Packet. Anything deterministic - RSS collection, Grants.gov XML ingestion,
Federal Register resolution, dedupe, FTS5 indexing, source freshness checks, and
known-grant exclusion - should happen automatically behind that interface.

The upstream LLM agent is the caller. The CLI itself should not call an LLM. It
should use deterministic retrieval, with local `usearch` semantic search as the
preferred candidate finder and SQLite FTS5 as fallback.

The CLI sits between a request like "find grants for this startup" and the
underlying collectors/database/resolvers. It lets a specific agent go fast
without reinventing grant source discovery every time.

## Source Of Truth

- Domain glossary: `CONTEXT.md`
- ADR: `docs/adr/0001-agent-facing-grant-deep-research.md`

## Top Workflows

1. Given a Resolved Agent Request, return a ranked Research Packet for a specific startup and funding question.
2. Automatically refresh stale deterministic source lanes before answering, without exposing source-specific commands as the user workflow.
3. Explain why one opportunity was recommended, including source evidence, provenance, deadline certainty, eligibility fit, and negative evidence.
4. Reuse the local Opportunity Ledger across repeated requests so the agent can ask narrower follow-ups without crawling from scratch.
5. Track changes and newly visible opportunities so the agent can say what changed since the last research run.

## Table Stakes

- Must ingest or reuse Grants.gov, SBIR/STTR, DOE EERE, NSF, ARPA-E, state economic development, EV infrastructure, smart-city, and founder-support source lanes.
- Must preserve official identifiers when available and dedupe known grants.
- Must distinguish confirmed deadlines from projected or awaiting-NOFO status.
- Must include effort estimates and conservative eligibility fit.
- Must output JSON suitable for agent consumption.
- Must support local semantic retrieval over normalized opportunity text and evidence, preferring `usearch` and falling back to FTS5.
- Must not call an LLM from inside the CLI.
- Must include negative evidence for must-check sources, especially ARPA-E.

## Data Layer

- Primary entities: assignments, opportunities, raw observations, evidence, provenance links, source snapshots, changes, fit assessments, research packets.
- Sync cursor: per-source last fetched time, ETag/Last-Modified when available, cached artifact hash, and last successful normalized observation.
- Search: `usearch` over a local opportunity/evidence corpus when available; SQLite FTS5 over title, sponsor, agency, opportunity type, eligibility, summary, deadline text, geography, technology signals, and source IDs as fallback.

## Product Thesis

- Name: Grant Finder / Grant Radar.
- Why it should exist: agents need a deterministic research substrate that turns many noisy funding sources into a fast, explainable, replayable recommendation packet for a specific startup context.

## Build Priorities

1. Define and validate the agent-facing assignment input and Research Packet output contracts.
2. Bring POC parity into the Go implementation: SQLite ledger, source provenance, Federal Register resolution, Grants.gov XML/API ingestion, FTS5, dedupe, idempotency.
3. Build `research` as the primary command that automatically refreshes stale deterministic lanes and returns ranked recommendations.
4. Build `explain` for provenance/evidence/negative-evidence inspection of a selected recommendation.
5. Hide source plumbing under debug/internal commands after the agent-facing surface passes acceptance.
