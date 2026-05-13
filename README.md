# grant-finder

Find non-dilutive funding for your U.S. startup from the command line.
SBIR/STTR solicitations, federal agency grants, state economic development
programs — ranked by fit, with every recommendation cited back to its source.

**No API keys.** Runs on free public data (Grants.gov, the Federal Register,
public agency RSS feeds). Self-hostable.

**For founders, the engineers working with them, and the AI agents driving
both.** Describe your startup in a paragraph; get back a ranked list of
funding opportunities with deadlines, fit rationale, and application outlines,
backed by source-cited evidence.

> Looking for a hosted, fully-managed version? See
> [Hosted service](#hosted-service) below.

## Current maturity

The agent-facing CLI surface is usable today: `research`, `explain`, `status`,
`doctor`, `agent-context`, and `version` are the supported top-level commands,
with source plumbing kept under `debug`. The harness verifies the fixture-driven
research flow end to end.

The project is still hardening as an operated product. Source coverage,
freshness policy, and long-running ingestion schedules are intentionally called
out in [Limitations](#limitations); the hosted service handles those operational
concerns for users who do not want to run the ledger themselves.

## Try it in 60 seconds

```bash
# 1. Get the repo
git clone https://github.com/openprose/grant-finder.git
cd grant-finder

# 2. Build the CLI
cd cli/grant-finder && go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder
cd ../..

# 3. Run it against the sample assignment
grant-finder research --assignment fixtures/acme-deeptech-assignment.sample.json
```

On first run the CLI refreshes its local ledger from public sources
(~30 seconds), then prints a ranked table:

```
FIT     PROGRAM                                              AGENCY                       DEADLINE     URL
high    SBIR autonomous vehicle fleet charging infrastr…    U.S. National Science Found… 2026-08-01   https://…
medium  America's Seed Fund Phase I (FY2026)                 U.S. National Science Found… 2026-07-15   https://…
medium  DOE EERE Vehicle Technologies SBIR                   U.S. Department of Energy    2026-09-30   https://…
…
```

Pass `--json` for the machine-readable Research Packet that an agent would
consume, including per-recommendation evidence, provenance, deadline
certainty, effort estimate, and source-lane coverage (including explicit
negative-evidence rows like *"no current ARPA-E programs match"*).

## What you can ask it

```bash
# Rank opportunities for a specific startup context
grant-finder research --assignment my-startup.json --json

# Show evidence and provenance for one recommendation
grant-finder explain rec-12 --json

# Check ledger freshness and source-lane coverage
grant-finder status --assignment my-startup.json --json

# Inspect health
grant-finder doctor --json
```

A `my-startup.json` looks like this — see the schema at
[`schemas/research-assignment.schema.json`](./schemas/research-assignment.schema.json):

```json
{
  "assignment_id": "acme-deeptech-2026-05-13",
  "research_question": "Find non-dilutive funding for autonomous EV fleet infrastructure.",
  "company_profile": {
    "name": "Acme Deep-Tech",
    "description": "A deep-tech startup building autonomous fleet-servicing infrastructure...",
    "stage": "startup",
    "location": "United States",
    "technologies": ["autonomous vehicles", "ev infrastructure", "robotics"]
  },
  "focus_areas": ["ev infrastructure", "autonomous vehicles", "robotics"],
  "target_geographies": ["United States", "California"],
  "known_grants": []
}
```

If you want an AI agent to fill this in from a paragraph-length brief, see
[the OpenProse example](./examples/openprose/) — it shows a complete
brief-to-report flow.

## How it works

```
your assignment (JSON)
        │
        ▼
 ┌────────────────────────────────────────────────────────────┐
 │  grant-finder research                                     │
 │   ├─ refresh stale source lanes (Grants.gov, Fed. Reg.,    │
 │   │   public agency RSS) — only if your local ledger       │
 │   │   is empty or stale                                    │
 │   ├─ retrieve candidates (usearch semantic when available, │
 │   │   FTS5 fallback)                                       │
 │   ├─ dedupe against your known grants                      │
 │   ├─ score fit, effort, deadline certainty, activity       │
 │   ├─ filter out closed / archived / past-due (by default)  │
 │   └─ rank                                                  │
 └────────────────────────────────────────────────────────────┘
        │
        ▼
 ranked recommendations + per-result evidence + source-lane coverage
```

The CLI keeps a local SQLite ledger at
`~/.local/share/grant-finder/grant-finder.sqlite` (or `$XDG_DATA_HOME` if set).
Repeat queries reuse it; nothing leaves your machine except the public-API
fetches the CLI makes against Grants.gov, the Federal Register, and configured
RSS feeds.

## Design choices worth knowing

- **The CLI doesn't call an LLM.** Ranking, fit scoring, dedupe, and
  source-lane coverage are deterministic for a fixed assignment, options, and
  ledger state. Fields such as `generated_at` and results after
  `--refresh auto` can change as time passes and public sources update.
- **No paid API keys.** Grants.gov, Federal Register, and public agency feeds
  cover the core federal sources. The CLI will work offline against a
  populated ledger.
- **Provenance over completeness.** Every recommendation cites its source.
  When a must-check lane (like ARPA-E) has no current match, the CLI says so
  explicitly rather than silently omitting the lane.
- **Self-hostable. Always.** This repo is the whole product. Hosting is the
  upsell, not feature parity.

## Install

```bash
# Most common: go install (binary lands in $GOPATH/bin)
go install github.com/openprose/grant-finder/cli/grant-finder/cmd/grant-finder@latest

# From source
git clone https://github.com/openprose/grant-finder.git
cd grant-finder/cli/grant-finder
go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder
```

Requires Go 1.25+ (see `cli/grant-finder/go.mod`). Optional:
install [`usearch`](https://github.com/unum-cloud/usearch) on `PATH` for
faster semantic retrieval; without it the CLI falls back to SQLite FTS5
automatically.

### Driving grant-finder from an AI agent (optional second step)

If you want an AI agent (Claude Code, Codex, Gemini) to drive `grant-finder`
end-to-end through the bundled OpenProse example, install the host-harness
skill so the agent's container resolves the binary correctly:

```bash
# From your local clone of this repo
ln -s "$PWD/skills/grant-finder" ~/.claude/skills/grant-finder
#   Codex:  ~/.codex/skills/grant-finder
#   Gemini: ~/.agents/skills/grant-finder
```

The skill teaches the agent the CLI's command surface. Combined with the
binary install above, this is what `examples/openprose/` needs to wire
cleanly. You do **not** need the skill if you're calling the CLI directly
from a shell or a script.

## What's in the box

| Path | What |
|---|---|
| `cli/grant-finder/` | The Go CLI source |
| `schemas/` | JSON Schema for assignment input and Research Packet output |
| `fixtures/` | Sample assignment + opportunities (generic deep-tech startup) |
| `examples/openprose/` | A runnable OpenProse system that turns a natural-language brief into a ranked report by driving the CLI |
| [`AGENTS.md`](./AGENTS.md) | Architecture and conventions for contributors and AI agents working on this repo |

## Limitations

- **U.S.-focused.** The source manifest covers federal and select U.S. state
  programs. International funding is not in scope yet.
- **Federal sources are the strongest lane.** State, foundation, and
  commercial grant databases are partial or absent.
- **Freshness is local unless hosted.** `--refresh auto` updates stale or empty
  local ledgers, but this repo does not run a background scheduler for you.
  Use your own scheduled job if you need continuously fresh self-hosted data.
- **SAM.gov is off by default.** It requires an API key, which the public
  binary intentionally does not configure.
- **No web scraping.** The CLI only reads structured sources (APIs and RSS).
  Programs that only publish via a JavaScript-rendered page or a paid
  database won't be picked up.
- **Eligibility decisions are yours.** The CLI rates eligibility *fit*
  conservatively but does not adjudicate. Always read the official source
  before applying.

## FAQ

**Q: Do I need an OpenAI / Anthropic / SAM.gov / Exa API key?**
A: No. None of the above. The CLI uses only free public APIs (Grants.gov,
Federal Register) and public RSS feeds.

**Q: Can I use this without an AI agent?**
A: Yes. The CLI takes a JSON file as input — you can write one by hand using
the schema in `schemas/`. The AI-agent integration just makes the
brief-to-assignment translation easier; the CLI itself is fully usable
standalone.

**Q: How fresh is the data?**
A: As fresh as the last `--refresh auto` run. A hosted version (see below)
keeps the ledger continuously fresh; if you're running it yourself, schedule
your own refresh.

**Q: Why isn't \<favorite source\> included?**
A: The default manifest is at
`cli/grant-finder/internal/grantfinder/data/{sources.json,feeds.json}`. PRs
that add public, key-free sources are welcome.

**Q: Will this work for non-U.S. startups?**
A: Not well, today. The source manifest is U.S.-focused. International
expansion is a known limitation, not a permanent decision.

## Hosted service

Running grant-finder yourself is the whole point of this repo — fork it,
build it, run it. If you'd rather have someone else handle source freshness,
ingestion scheduling, monitoring, and reliability, the OpenProse team offers
a hosted version of this same CLI as a service. The code is the same; the
operational concerns are theirs. See <https://openprose.ai>.

## Contributing

Issues are welcome. We're a small team and contribute time is scarce, so
substantive PRs are most useful when they come with: a clear motivating use
case, green `make validate` and `make dogfood-agent` runs, and respect for the
[invariants documented in `AGENTS.md`](./AGENTS.md#invariants-dont-break-these)
(notably: no LLM inside the CLI, no paid API keys, no breaking the public
command surface). Run `make fuzz-smoke FUZZTIME=10s` when changing parsers,
JSON projection, or debug SQL validation. New source manifests are especially
welcome.

## License

MIT. See [`LICENSE`](./LICENSE).
