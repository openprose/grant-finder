# Grant Finder Prototype Shipcheck Status

Date: 2026-05-08

## Verdict

Hold. The old shipcheck remains invalidated as product acceptance evidence, but
the replacement agent-facing harness now passes.

The previous shipcheck treated the generated Go CLI as shippable because it
compiled, ran smoke commands, and exercised source plumbing. That was the wrong
acceptance target. The product is an agent-facing Grant Deep Research interface,
not a human-facing source operations CLI.

## What Remains Useful

- The generated code can be mined for SQLite, FTS5, source manifests, read-only
  helpers, and POC parity work.
- The manual smoke results remain useful as source reachability evidence.
- The prototype demonstrated that deterministic source work can be ported into
  a Go CLI.

## Why It Is Not Shippable

- Grants.gov API ingestion exists, feed-derived Federal Register links are
  hydrated during sync, and bounded Grants.gov XML extract ingestion has been
  ported into the Go collector path. The remaining work is scheduling/freshness
  policy and richer source-lane coverage, not parser existence.
- `printing-press dogfood` still warns on generated-CLI assumptions: sync/search
  do not look like the canonical generated API pipeline, and the static workflow
  mapper cannot understand fixture paths or extracted variables even though
  `workflow-verify` runs the workflow successfully.
- `printing-press scorecard` remains low because the project is an agent-facing
  substrate, not yet a canonical spec-generated CLI with MCP surface, endpoint
  examples, and standard generated cache patterns.

## Current Passing Gates

```bash
make validate
make validate-product-cli
make dogfood-agent
printing-press workflow-verify --dir cli/grant-finder --json
```

The live Grants.gov API/XML ingestion path was also smoke-tested with:

```bash
grant-finder debug sync --grants-only --grants-xml --keyword SBIR --limit 1 --json
```

Result: one official Grants.gov API record and one matching XML extract record
ingested into a fresh ledger with zero errors.

## Replacement Acceptance Gate

Use:

```bash
make validate
make validate-product-cli
make dogfood-agent
```

Then run a behavioral fixture where a deep-tech-startup Research Assignment produces
a Research Packet with ranked opportunities, evidence, provenance, deadline
certainty, effort estimate, application outline for high-fit matches, known-grant
dedupe, and ARPA-E negative evidence when applicable.
