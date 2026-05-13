# grant-finder

The CLI binary. Returns ranked non-dilutive funding opportunities for a
startup, with citations. Top-level repo docs are at
[`../../README.md`](../../README.md); contributor docs at
[`../../AGENTS.md`](../../AGENTS.md).

## Build

```bash
go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder
```

## Commands

| Command | What it does |
|---|---|
| `research` | Returns ranked grants for a startup assignment, with evidence and provenance |
| `explain` | Shows the source trail behind one recommendation |
| `status` | Reports ledger freshness and source-lane coverage |
| `doctor` | Checks the CLI's local health (SQLite, FTS5, optional `usearch`) |
| `agent-context` | Prints a JSON description of the command surface — for AI agents introspecting before use |
| `version` | Prints the version |

## Common flags

| Flag | Effect |
|---|---|
| `--db <path>` | SQLite ledger path. Default: `~/.local/share/grant-finder/grant-finder.sqlite` |
| `--json` | JSON output (default human output is a table) |
| `--compact` | Compact JSON output |
| `--select <fields>` | Project specific top-level JSON paths (comma-separated, dot-pathed) |
| `--limit N` | Cap result rows (default: 10 for `research`) |
| `--refresh auto\|off` | Refresh stale source lanes before answering. Default: `auto` |
| `--semantic auto\|usearch\|off` | Semantic retrieval mode. Default: `auto` |
| `--include-inactive` | Include closed, archived, expired, or past-due records (off by default) |

## Maintainer commands

Source plumbing — sync, smoke checks, raw SQL — lives under `debug`. These
are substrate, not the product interface:

```bash
grant-finder debug sync --grants-only --keyword SBIR --limit 1 --json
grant-finder debug feeds smoke --limit 10 --json
grant-finder debug sources smoke --limit 10 --json
grant-finder debug sql 'select title, url from opportunities limit 10'
```

See [`../../AGENTS.md`](../../AGENTS.md) for the full debug surface and
verification gates.

## Notes

- SAM.gov ingestion is not enabled. It requires an API key the public binary
  intentionally does not configure.
- Feed hits are leads, not eligibility decisions. Always inspect the official
  source URL before applying.
- Results are deterministic — ranking, fit scoring, dedupe, and coverage
  don't use an LLM.
