# AGENTS.md

For AI coding agents and contributors working **on** this repo (as opposed to
*using* the CLI). If you are evaluating grant-finder as a user, read
[`README.md`](./README.md) first.

## What this repo is

A Go CLI (`grant-finder`) that an upstream AI agent calls to turn a startup's
context into a deterministic, evidence-backed Research Packet of ranked
non-dilutive funding opportunities. The CLI does the ledger work; agent
judgment lives on the caller side, never inside the CLI.

## Where things live

| Path | What |
|---|---|
| `cli/grant-finder/` | The Go CLI source. Module: `github.com/openprose/grant-finder/cli/grant-finder` |
| `cli/grant-finder/cmd/grant-finder/` | Binary entry point |
| `cli/grant-finder/internal/cli/` | Cobra command tree (`research`, `explain`, `status`, `debug` subtree) |
| `cli/grant-finder/internal/grantfinder/` | Domain logic: SQLite store, FTS5/usearch retrieval, Grants.gov/Federal Register collectors, ranking |
| `schemas/` | JSON Schema for Research Assignment input and Research Packet output |
| `fixtures/` | Sample assignment + opportunities (generic deep-tech startup) used by the harness |
| `docs/adr/` | Architecture Decision Records. `0001-agent-facing-grant-deep-research.md` is the foundational one — read it before changing the public command surface |
| `printing-press/` | Product-thesis docs: brief, absorb manifest, plan, ship checklist, current status. This is the *why* behind the public command surface |
| `examples/openprose/` | A runnable OpenProse `kind: system` system that drives the CLI end-to-end |
| `skills/grant-finder/` | Host-harness skill (SKILL.md) that wires the CLI as a Forme dependency. Users symlink this into `~/.claude/skills/`, `~/.codex/skills/`, or `~/.agents/skills/` |
| `scripts/` | Validation harness (`validate_product_surface.py`, `validate_agent_dogfood.py`) |
| `CONTEXT.md` | Domain glossary. The vocabulary in here is load-bearing — use it verbatim in code, commits, docs |

## Invariants (don't break these)

1. **No LLM inside the CLI.** Every public command reports `no_llm: true`. The
   CLI must remain deterministic and replayable. If you find yourself reaching
   for an LLM library inside `internal/`, stop and reconsider the design.
2. **No API keys.** The CLI uses only free public APIs (Grants.gov, Federal
   Register) and public agency RSS feeds. SAM.gov is intentionally gated off
   pending a key path. Do not introduce a service that requires Exa, OpenAI,
   Anthropic, Stripe, browser automation, or any paid third party.
3. **Agent-facing command surface stays minimal.** Agent-facing product
   commands are `research`, `explain`, and `status`. Public utility commands
   are `doctor`, `agent-context`, and `version`. Source plumbing lives under
   `debug`. See `docs/adr/0001-...md`.
4. **Provenance is non-negotiable.** Every opportunity in the ledger has source
   refs; every recommendation has evidence items; explain returns the source
   trail. Do not introduce paths that produce ranked output without provenance.
5. **Schemas are the contract.** `schemas/research-assignment.schema.json` and
   `schemas/research-packet.schema.json` are the boundary between the CLI and
   its callers. Bump the schema version before changing required fields.

## Verification gates

```bash
make validate              # go test ./...
make validate-product-cli  # build CLI + python3 scripts/validate_product_surface.py (checks command surface)
make dogfood-agent         # end-to-end: seed fixture → research → explain → status
make fuzz-smoke            # optional Go fuzz smoke for parsers/projection/read-only SQL guard
```

The first three gates must pass before merging anything that touches the public
command surface or domain logic. `dogfood-agent` validates schema-shaped
Research Packet output plus the Aeseon-style behavioral contract: ranked
opportunities, provenance-bearing evidence, ARPA-E negative evidence, and
`no_llm: true`.

Run `make fuzz-smoke FUZZTIME=10s` when changing assignment parsing, feed/XML
parsing, Federal Register hydration, JSON projection, or debug SQL validation.

## Debug surface (maintainer-only)

These commands live under `debug` because they're substrate, not product:

```bash
grant-finder debug sync --grants-only --keyword SBIR --limit 1 --json
grant-finder debug feeds smoke --limit 10 --json
grant-finder debug sources smoke --limit 10 --json
grant-finder debug sql 'select title, url from opportunities limit 10'
grant-finder debug seed-fixture --fixture fixtures/acme-deeptech-opportunities.sample.json --db /tmp/test.sqlite
```

Do not promote any of these to the top-level command tree.

## Source manifests

The CLI embeds a source manifest at
`cli/grant-finder/internal/grantfinder/data/{feeds.json,sources.json,grant-finder-feeds.opml}`.
To add a new source lane:

1. Add an entry to the relevant JSON file.
2. Run `make validate` to confirm the manifest parses.
3. Run `make dogfood-agent` to confirm no regressions in the behavioral fixture.
4. Add coverage notes if the lane is must-check (see `BuildCoverage()` in
   `internal/grantfinder/research.go`).

## Mycelium notes

Use `mycelium.sh` for git-notes-based design notes that travel with the code:

```bash
mycelium.sh read <file>            # retrieve notes on a file
mycelium.sh refs <file>            # find all notes pointing at the file
mycelium.sh find decision          # find all decision-kind notes
mycelium.sh note <file> -k decision -t "Title" -m "Why this matters"
```

Retrieve all notes with `mycelium.sh dump`. Notable ones:

- `examples/openprose/src/grant-radar.prose.md` — skill-wrapped CLI dep + no-API-key invariant
- `examples/openprose/src/format-report.prose.md` — render-only constraint (no re-ranking)
- `examples/openprose/fixtures/polyspectra.brief.txt` — canonical sample brief, public info only
- `cli/grant-finder/internal/grantfinder/research.go` — evidenceScore weights + academic-stage routing
- `cli/grant-finder/internal/grantfinder/grants.go` — forecasted|posted default
- `cli/grant-finder/internal/grantfinder/store.go` — CoverageMatch ledger-level check
- `skills/grant-finder/SKILL.md` — agent-facing contract for the CLI

When you make a non-obvious design decision or recover from a mistake, leave a
mycelium note so the next agent does not relearn the lesson.

## OpenProse example

`examples/openprose/` is a runnable OpenProse `kind: system` system
(`grant-radar`) that demonstrates how an upstream AI agent drives the CLI.
Four services: `resolve-assignment`, `run-research`, `explain-top-picks`,
`format-report`, plus the top-level system that wires them.

The CLI binary is wired as a Forme dependency via the `grant-finder`
**host-harness skill** at `skills/grant-finder/SKILL.md`, declared in the
top-level system's `### Skills` block and in the two services that shell out
(`run-research`, `explain-top-picks`). Forme's `skill_unresolved` path makes
this fail-closed at wiring time — if the skill is not installed in the host
harness's skills dir, the system refuses to run rather than failing mid-flow
inside a service.

This is the workaround for Forme not having a first-class `### Binaries`
section (see contract-markdown.md:127-162). The earlier soft-doc
`requires: grant-finder CLI tool in PATH` frontmatter approach was
*documentation* and did not enforce; switching to skill-wrapped fixed a
real bug where `prose run` would fail mid-execution with no clear signal.

## The grant-finder host-harness skill

`skills/grant-finder/SKILL.md` is shipped with this repo so users can
symlink it into their agent harness's skills directory. It teaches an agent
the CLI's command surface — what subcommands matter (`research`, `explain`,
`status`), key flags, the `no_llm: true` drift guard, and where to read
more. The README's "Driving grant-finder from an AI agent" section
documents the install step.

If you change the CLI's public command surface, update the skill SKILL.md
in the same change — the skill is the agent-facing contract.

## Renaming or restructuring

If you rename the binary or change the Go module path, you must:

1. Update `go.mod` and every import statement.
2. Update `cmd/<binary>/main.go` path.
3. Update the Makefile build targets.
4. Update `workflow_verify.yaml` (fixture paths).
5. Update `printing-press/product-surface.json` (`prototype_cli.path`).
6. Update both READMEs and `examples/openprose/src/grant-radar.prose.md`
   (the `## Prerequisites` install commands).
7. Run all three `make` gates.

Single source of truth: `go.mod` declares the module path; everything else
references it. Avoid hardcoded path strings outside of necessary places.

## Coding conventions

- Apache headers were stripped; current copyright is `// Copyright 2026
  OpenProse contributors. Licensed under MIT. See LICENSE.` Keep this exact
  string on every new Go file.
- Use `modernc.org/sqlite` (the pure-Go driver). Do not add CGO dependencies.
- New CLI subcommands follow the pattern in `internal/cli/*.go`: one file per
  command, register in `root.go` for public commands or `debug.go` for
  maintainer ones.
- JSON output must include `no_llm: true` (or equivalent on the retrieval
  block) for any deterministic command surface.

## Hosted service positioning

This repo is the OSS half of an OSS-then-hosted product. The hosted service
runs the same code with operational concerns handled (source freshness,
ingestion scheduling, monitoring, support). Do not gate features behind a
tier in this repo — the OSS path must run end-to-end on free public APIs
forever. Differentiation is operational, not by feature.
