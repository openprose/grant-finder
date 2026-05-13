---
name: format-report
kind: service
---

# Format Report

### Description

Compose a human-readable markdown report from the deterministic Research
Packet and the per-recommendation explanations. This is the only place where
LLM judgment shapes output that a founder will read — and even here, the job
is faithful rendering, not new analysis.

The CLI has already done ranking, fit assessment, deadline certainty, effort
estimation, and coverage. This service must not re-rank, re-score, or invent
fit rationales. Its job is to write English over the structured facts.

### Requires

- `research_packet`: Research Packet from `run-research`
- `top_pick_explanations`: per-recommendation explanations from `explain-top-picks`

### Ensures

- `markdown_report`: a markdown document with these sections:
  - `# Funding Pipeline — <company name>` header
  - `## Summary`: total potential funding, count of high-fit grants, nearest
    deadline, one-line note on retrieval backend and `no_llm` status
  - `## Top Picks`: per recommendation, a subsection with program name, agency,
    amount, deadline + certainty, fit rationale (verbatim from
    `eligibility_fit.explanation`), effort estimate, application outline (if
    high-fit), and an `<details>` block holding the explanation's evidence
    list with source URLs
  - `## Coverage`: a table mirroring `research_packet.coverage` — one row per
    source lane, with status and note columns
  - `## Negative Evidence`: an explicit callout listing coverage rows whose
    status is `checked_no_match`, especially the ARPA-E row when present
  - `## Provenance`: a line stating the retrieval backend, refresh policy, and
    `generated_at` timestamp from the packet

### Shape

- `self`: render the structured packet and explanations into markdown using
  the sections above, in the order specified
- `prohibited`: introducing claims not backed by `evidence` items;
  rewriting `eligibility_fit.explanation` strings; re-ranking grants;
  adjusting `deadline_certainty` language; hiding negative-evidence rows;
  paraphrasing source claims when a direct quote would do

### Strategies

- Use `research_packet.summary.notes` verbatim in the Summary section. Those
  notes are part of the CLI's contract with the upstream agent.
- For each top pick, link the program name to `grants[i].url`. Show the
  apply-URL when present as a separate "Apply:" line.
- Render `application_outline` as a numbered list under each high-fit pick.
  When the field is empty, omit the section entirely rather than emitting a
  placeholder.
- For the Negative Evidence section, surface every coverage row where
  `status == "checked_no_match"` and `note` is non-empty. The CLI specifically
  encodes "no current ARPA-E programs match" as a load-bearing absence; do not
  filter it out.
- Keep the entire report under 800 lines for legibility. If the packet has
  more than 10 grants and 5+ high-fit, summarize the long tail in a single
  closing bullet rather than expanding every record.
