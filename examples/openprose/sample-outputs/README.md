# Sample outputs

Real outputs from running `grant-radar` against the three example briefs.
Captured 2026-05-13 against live Grants.gov, Federal Register, and public
agency RSS sources.

Each example directory contains the five service bindings in execution order:

```
01-startup_brief.md          # the human-language brief (caller input)
02-research_assignment.md    # resolved Research Assignment JSON
03-research_packet.md        # deterministic grant-finder research output
04-top_pick_explanations.md  # per-pick evidence + provenance
05-markdown_report.md        # final human-readable report
```

## What to look at first

**For each example, open `05-markdown_report.md`** — that's the human-readable
artifact at the end of the chain. The other four files are the structured
intermediate bindings.

## The three examples

| Example | Brief | Outcome |
|---|---|---|
| [`polyspectra/`](./polyspectra/) | US small business making rugged photopolymer resins for industrial 3D printing | 10 ranked opportunities; 0 high-fit; top-3 fallback = real currently-open SBIR records from ACL and NIH |
| [`cypris/`](./cypris/) | Berkeley-based advanced materials company developing structural color coatings | Same shape as polySpectra — 0 high-fit, top-3 SBIR fallback with provenance |
| [`enact-lab/`](./enact-lab/) | Yale academic clinical psychiatry lab studying psychedelics | 0 high-fit; the agent flagged the top-3 fallback as **exclusion evidence** rather than recommendations, because the brief says "academic group — SBIR/STTR is not the right vehicle" |

## A real bug the ENACT run surfaced

The ENACT run produced a thoughtful caveat that's worth reading. Even though
the assignment constraints say "no SBIR/STTR," the deterministic CLI ranking
still returned SBIR records as the top-3 because **constraints are not part
of candidate exclusion** in `grant-finder` today — they shape the assignment
text fed to FTS5 but don't filter records by record-type. The agent honored
the constraint at the report layer; the underlying fix belongs in
`grant-finder` itself. See the run's mycelium notes.

## All three runs proved `no_llm: true`

Every `research_packet.md` and every record in `top_pick_explanations.md`
carries `no_llm: true`. The drift guard in `run-research.prose.md` and
`explain-top-picks.prose.md` would have rejected any record that didn't.

## Reproducing these outputs

```bash
# Recommended (granular sandbox, requires patched prose CLI):
PROSE_CODEX_SANDBOX_MODE=workspace-write \
PROSE_CODEX_APPROVAL_POLICY=never \
PROSE_CODEX_ADD_DIR=$HOME/.local/share/grant-finder \
PROSE_CODEX_NETWORK=true \
prose run src/grant-radar.prose.md \
  --startup_brief "$(cat fixtures/polyspectra.brief.txt)"

# Fallback (any prose version):
PROSE_CODEX_SANDBOX_MODE=danger-full-access \
PROSE_CODEX_APPROVAL_POLICY=never \
prose run src/grant-radar.prose.md \
  --startup_brief "$(cat fixtures/polyspectra.brief.txt)"
```

Your live outputs will differ from these samples — Grants.gov posts and
closes opportunities every week, so the ranked list and deadlines shift.
The shape of the report, however, should look the same.
