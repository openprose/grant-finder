---
name: explain-top-picks
kind: service
---

# Explain Top Picks

### Description

For each high-fit recommendation in the Research Packet, invoke
`grant-finder explain <id> --json` to retrieve the per-recommendation evidence
and provenance trail. The CLI already includes summary evidence on each grant;
`explain` returns the full source trail (raw observations, source snapshots,
dedupe rationale) that the agent should use when drafting an application.

### Requires

- `research_packet`: Research Packet from `run-research`

### Ensures

- `top_pick_explanations`: array of explanation records for high-fit grants in
  the packet, each containing:
  - `recommendation_id`: matches `grants[i].recommendation_id` in the packet
  - `opportunity`: normalized opportunity record
  - `evidence`: list of `{ source_id, url, claim }` items
  - `sources`: list of `{ source_id, source_url, raw_id }` provenance links
  - `notes`: any notes the CLI emitted (e.g., dedupe rationale)
  - `no_llm`: must be `true` on every record

### Skills

- grant-finder

### Shape

- `self`: select up to 5 high-fit recommendations, invoke `explain` for each,
  collect results, publish the array
- `prohibited`: inventing evidence the CLI did not return; merging or
  paraphrasing evidence across recommendations; explaining recommendations
  the packet did not return

### Strategies

- Filter `research_packet.grants` to entries where
  `eligibility_fit.level == "high"`. Cap at 5 to keep the explanation work
  bounded — the report stays scannable and process time stays predictable.
- If fewer than 2 high-fit grants exist, fall back to the top 3 by `score`
  regardless of fit level. The founder still benefits from provenance on the
  best available leads.
- For each selected grant, invoke:
  ```bash
  grant-finder explain "<recommendation_id>" --json
  ```
  The CLI accepts the `rec-<n>` prefix; pass the field verbatim.
- Run the explain calls in parallel — they are read-only against the local
  ledger and do not contend.
- Validate each response's `no_llm: true` before adding it to the output. Drop
  any record where that flag is missing or false and surface the drop in the
  service log.
