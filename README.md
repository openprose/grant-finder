# Grant Finder

Agent-facing grant deep-research substrate. Open source. Self-hostable.

Grant Finder helps an upstream agent answer a funding-opportunity request for a
specific startup. It reuses deterministic collectors, a local SQLite Opportunity
Ledger with FTS5, optional `usearch` semantic retrieval, source provenance, and
change detection so the upstream LLM does not rebuild grant research from
scratch on every request.

The upstream LLM uses the CLI. **The CLI itself must not call an LLM.** It
operates entirely on public free APIs (Grants.gov, Federal Register) and public
agency feeds.

## Standing Goal

> Keep a startup's funding pipeline curated, provenance-backed, and ready for
> agent-driven follow-up — without reinventing source discovery every time.

## Agent CLI Surface

The CLI is not a human source-operations console. The public surface is:

- `research` — accept a Research Assignment, return a Research Packet
- `explain` — show evidence and provenance for one recommendation
- `status` — report ledger freshness and source coverage

Source plumbing (sync, feeds, federal-register, grants, sql, export) lives
under `debug` and is not the headline interface. See
`docs/adr/0001-agent-facing-grant-deep-research.md`.

## Quick Start

Build the CLI:

```bash
cd cli/grant-finder
go build -o /tmp/grant-finder ./cmd/grant-finder
```

Seed the deterministic fixture used by the harness:

```bash
/tmp/grant-finder debug seed-fixture \
  --fixture ../../fixtures/acme-deeptech-opportunities.sample.json \
  --db /tmp/grant-finder-demo.sqlite \
  --json
```

Ask for a research packet:

```bash
/tmp/grant-finder research \
  --assignment ../../fixtures/acme-deeptech-assignment.sample.json \
  --db /tmp/grant-finder-demo.sqlite \
  --refresh off \
  --semantic usearch \
  --json
```

The `research`, `explain`, and `status` commands are deterministic and report
`no_llm: true`. Semantic retrieval prefers local `usearch`; when the local
corpus is not indexed or `usearch` is unavailable, the CLI falls back to
SQLite FTS5. Default `research` output filters out closed, archived, expired,
and past-due opportunities. Pass `--include-inactive` for historical comps.

## Harness

```bash
make validate              # fast repo check + go test ./...
make validate-product-cli  # stricter product-surface contract gate
make dogfood-agent         # exercise CLI as an upstream agent would
```

## Hosted Service

Running Grant Finder yourself is the whole point of this repo — fork it, run it,
maintain it. If you would rather have someone else operate it (source freshness,
monitoring, ingestion, reliability, support), the OpenProse team offers a
hosted Grant Radar service. See `https://openprose.ai` for details.

## Credits

This CLI was scaffolded with
[CLI Printing Press](https://github.com/mvanhorn/cli-printing-press) and is now
maintained directly in this repo. The Printing Press meta-artifacts that
explain the product thesis live under `printing-press/`.

## License

MIT. See `LICENSE`.
