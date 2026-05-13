# grant-finder CLI

Agent-facing grant deep-research CLI. The upstream LLM agent uses this CLI;
the CLI itself must not call an LLM.

## Build

```bash
go build -o /tmp/grant-finder ./cmd/grant-finder
```

## Public Commands

- `research --assignment <path|->` — return a Research Packet for a resolved
  Research Assignment. Output includes ranked grants, evidence, provenance,
  deadline certainty, effort estimate, coverage rows, and negative evidence
  for must-check sources.
- `explain <recommendation-id|opportunity-id>` — show evidence and provenance
  for a recommendation.
- `status` — report ledger freshness and assignment source-lane coverage.

## Common Flags

- `--db <path>` — SQLite ledger path (default `~/.local/share/grant-finder/grant-finder.sqlite`)
- `--json` — JSON output (preferred for agent flows)
- `--compact` — compact JSON output
- `--select <fields>` — project specific top-level paths (comma-separated, dot-pathed)
- `--limit N` — cap result rows (default 10 for `research`)
- `--refresh auto|off` — refresh stale source lanes before answering (default `auto`)
- `--semantic auto|usearch|off` — semantic retrieval mode (default `auto`)
- `--include-inactive` — include closed, archived, expired, or past-due records

## Debug Surface

Source plumbing lives under `debug` and is for maintainers, not the agent
product interface:

```bash
./grant-finder debug sync --grants-only --keyword SBIR --limit 1 --json
./grant-finder debug feeds smoke --limit 10 --json
./grant-finder debug sources smoke --limit 10 --json
./grant-finder debug sql 'select title, url from opportunities limit 10'
```

## Doctor

```bash
./grant-finder doctor --json
```

Reports manifest counts, SQLite/FTS5 status, and `usearch` availability.

## Notes

- SAM.gov ingestion is not enabled; it requires an API key the public binary
  does not configure.
- Feed hits are leads, not eligibility decisions. Always inspect source URLs.
