# Grant Finder Absorb Manifest

## Scope

This manifest absorbs the OpenProse Grant Radar contract and the existing Python
POC. It corrects the public CLI surface: source operations are implementation
machinery, while the product surface is an agent-facing research interface.

## Absorbed

| # | Feature | Best Source | Our Implementation | Added Value | Status |
|---|---------|-------------|--------------------|-------------|--------|
| 1 | Scan canonical funding sources | OpenProse grant-radar strategies | Background collectors for Grants.gov, SBIR/STTR, DOE EERE, NSF, ARPA-E, state/local incentives, EV infrastructure, and founder-support feeds | Runs deterministically before research when stale | shipping |
| 2 | Match opportunities to company profile | OpenProse `eligibility-matcher` | `research` consumes assignment JSON from the upstream agent and scores fit | The CLI does not ask the human for context the agent already has | shipping |
| 3 | Compile prioritized report | OpenProse `grant-compiler` | `research --json` emits a Research Packet sorted by urgency, fit, then amount | Agent-ready structured output with evidence | shipping |
| 4 | Include program name and agency | OpenProse report contract | Normalized opportunity fields plus official source provenance | Preserves canonical names and source IDs | shipping |
| 5 | Include amount/funding range | OpenProse report contract | Extract amount text plus normalized range when parseable | Keeps raw evidence when normalization is uncertain | shipping |
| 6 | Include deadline and urgency | OpenProse report contract | Deadline field with `confirmed` or `projected` certainty | Avoids false precision around awaiting NOFOs | shipping |
| 7 | Conservative eligibility fit | OpenProse strategy | Fit score and rationale stored per assignment/opportunity | High/medium/low labels remain explainable | shipping |
| 8 | Effort estimate | OpenProse report contract | Low/medium/high estimate from evidence about LOI, proposal, match, and partnerships | Helps the upstream agent prioritize work | shipping |
| 9 | Application outline for high-fit opportunities | OpenProse report contract | `research` includes outline sections for high-fit rows | Turns research into next action, not just discovery | shipping |
| 10 | Known grants exclusion | OpenProse `known_grants` input | Assignment payload includes known grants; dedupe before ranking | Prevents repeated recommendations | shipping |
| 11 | ARPA-E must-check negative evidence | OpenProse strategy | Research Packet includes source coverage row: `No current ARPA-E programs match` when appropriate | Makes absence explicit instead of silent | shipping |
| 12 | State-specific checks per target geography | OpenProse strategy | Target geography expands source lanes and evidence coverage expectations | Does not rely only on federal pass-through sources | shipping |
| 13 | SQLite opportunity ledger | Python POC | Go store with raw observations, normalized opportunities, provenance, changes, and FTS5 | Reusable across repeated agent requests | shipping |
| 14 | Grants.gov XML reconciliation | Python POC | Background collector for nightly XML extract and Applicant API sample lookups | Canonical bulk source, not a user-facing XML command | shipping |
| 15 | Federal Register resolution | Python POC | Resolver linked to source observations | Formal notices hydrate automatically, not as top-level workflow | shipping |
| 16 | Idempotent sync and change rows | Python POC | Background refresh writes changes only when content changes | Enables "what changed" follow-ups | shipping |
| 17 | Semantic and FTS5 retrieval | User direction + Python POC | `usearch` semantic retrieval over local opportunity corpus, with FTS5 fallback | The upstream LLM agent gets fast candidate recall without the CLI calling an LLM | shipping |
| 18 | OPML/feed/source manifests | Python POC | Versioned source manifests remain inputs | Manifests support collectors, not product commands | shipping |

## Transcendence

| # | Feature | Command | Why Only We Can Do This |
|---|---------|---------|-------------------------|
| 1 | Research Packet compiler | `research --assignment assignment.json --json` | Requires joining startup context, ledger records, evidence, source coverage, fit, and next actions into one agent-ready response |
| 2 | Automatic deterministic refresh | `research --refresh auto` | Hides source maintenance behind the answer path so agents do not operate collectors by hand |
| 3 | Local semantic retrieval | `research --semantic auto --json` | Uses `usearch` over the local opportunity corpus for high-recall candidate finding without any LLM call inside the CLI |
| 4 | Negative evidence audit | `research --include-coverage --json` | Requires knowing must-check sources and recording when each source was checked and yielded no match |
| 5 | Evidence-backed recommendation explanation | `explain <recommendation-id> --json` | Requires provenance links from recommendation to raw observations, source snapshots, and fit rationale |
| 6 | Known-grant aware dedupe | `research --assignment assignment.json --json` | Requires combining assignment-specific exclusions with canonical opportunity identifiers and local ledger aliases |
| 7 | Freshness and change-aware prioritization | `research --since last-run --json` | Requires local change history across sources, not a single live search call |
| 8 | Source-lane coverage contract | `status --assignment assignment.json --json` | Requires evaluating coverage against the assignment's focus areas and geographies |

## Public Agent Surface

| Command | Purpose |
|---------|---------|
| `research` | Primary entrypoint. Takes a resolved assignment and returns the Research Packet. |
| `explain` | Returns evidence, provenance, fit rationale, and source trail for one recommendation or opportunity. |
| `status` | Reports ledger freshness and source coverage for an assignment. |

## Internal Or Debug Surface

The following capabilities may exist for diagnostics but must not be the product
interface: feed smoke checks, source smoke checks, Grants.gov XML fetch, Federal
Register hydration, SQL inspection, OPML export, raw sync, and raw FTS search.

## Acceptance Notes

- The current generated Go CLI does not satisfy this manifest because it exposes
  source plumbing as the primary interface and lacks `research`.
- Implementation cannot be considered shippable until a sample deep-tech-startup
  assignment produces a valid Research Packet with ranked grants, evidence, and
  negative evidence.
