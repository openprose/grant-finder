# ADR 0001: Build an agent-facing grant deep-research substrate

Date: 2026-05-07

## Status

Accepted

## Context

The first prototype drifted toward a human-facing source-operations CLI:
`feeds smoke`, `sources smoke`, `grants xml`, and `federal-register hydrate`
were treated as product commands. That framing is wrong for OpenProse.

The upstream OpenProse agent already knows the startup, customer, technology,
geography, constraints, and current user goal. Grant Finder exists so that
agent can answer "find grants for this startup" quickly from reusable evidence
and deterministic source work, rather than rebuilding a grant research process
from scratch on every request.

A representative deep-tech-startup grant-radar contract defines the product
shape: scan funding sources, surface evidence-backed candidate opportunities,
and let the upstream agent compile a prioritized report with deadlines, fit,
effort, links, application outlines, and explicit negative evidence for
must-check sources.

## Decision

Grant Finder is an agent-facing Grant Deep Research substrate, not a source API
wrapper and not a human operator console.

The public product interface accepts a Resolved Agent Request or Research
Assignment and returns a Research Packet:

- candidate funding opportunities in retrieval order
- preliminary eligibility signals and explanation
- amount and deadline with confirmed/projected status
- evidence and provenance for each candidate
- negative evidence for must-check sources
- freshness and coverage notes
- application outline hints for high-fit candidates

The upstream LLM agent uses this CLI. The CLI itself does not call an LLM. It
may use deterministic retrieval and local semantic search, with `usearch` as the
preferred semantic backend and SQLite FTS5 as fallback. Final ranking,
rejection, and "no good match" judgment belong in the upstream agent program,
not in the CLI.

Deterministic collectors, resolvers, smoke checks, source hydration, API/XML
ingestion, dedupe, FTS5 indexing, and change detection are background machinery.
They should run automatically when stale or be available under debug/internal
commands, but they are not the primary product surface.

## Consequences

- The headline command is `research`, not `sync` or `federal-register hydrate`.
- The binary must not require API keys for LLM providers or send assignment data
  to a model provider.
- Semantic search belongs inside retrieval. It accelerates the upstream agent's
  work but does not make final fit judgments.
- Source-specific actions are internal capabilities unless they are needed for
debugging the research substrate.
- The CLI must not ask the human user to provide the company profile. The
upstream agent passes that context in the assignment payload.
- Product acceptance requires a sample Research Packet, not just successful
source ingestion.
- The existing generated Go CLI is retained only as a prototype ledger substrate
until it is regenerated or reworked against this ADR.

## Alternatives Considered

- **Human source operations CLI**: useful for debugging, but it makes the user or
  upstream agent operate collectors instead of asking for the answer.
- **Single API wrapper**: too narrow; the value comes from reconciling many
  sources into a provenance ledger.
- **Pure prompt/skill**: repeats source discovery and loses deterministic
  evidence, freshness, dedupe, and change tracking across requests.
