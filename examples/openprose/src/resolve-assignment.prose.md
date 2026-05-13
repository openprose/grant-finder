---
name: resolve-assignment
kind: service
---

# Resolve Assignment

### Description

Turn a free-form `startup_brief` into a schema-valid Research Assignment JSON
that the `grant-finder` CLI can consume. This is the agent-side translation
step: the founder talks in sentences, the CLI takes structured input.

The Research Assignment schema lives at
`schemas/research-assignment.schema.json` in the public grant-finder repo. The
service must validate its output against that schema before publishing it.

### Requires

- `startup_brief`: free-form description of the startup, its technology focus,
  geography, stage, and funding question

### Ensures

- `research_assignment`: JSON conforming to
  `schemas/research-assignment.schema.json`, with these fields filled
  conservatively:
  - `assignment_id`: a stable slug derived from the company name and date
  - `research_question`: a single sentence restating the funding question
  - `company_profile`: `{ name, description, stage, location, technologies, constraints }`
  - `focus_areas`: 2–8 technology/program lanes derived from the brief
  - `target_geographies`: jurisdictions the company can credibly apply in
  - `known_grants`: any grants the brief explicitly says the founder already
    knows about (excluded from CLI ranking)

### Shape

- `self`: parse the brief, extract entities, fill the assignment fields,
  validate against the schema, publish the assignment
- `prohibited`: inventing technology areas the brief does not support;
  inferring jurisdictions the company has no presence in; coining a company
  name when the brief does not provide one; emitting an assignment that does
  not validate against the schema

### Strategies

- Read `schemas/research-assignment.schema.json` once before drafting so field
  names, required keys, and types are exact.
- Pull `focus_areas` from explicit nouns in the brief, not adjacent
  associations. *"EV charging"* belongs; *"clean energy"* belongs only if the
  brief actually uses those words or a near synonym.
- For `target_geographies`, include `United States` plus any state-level
  jurisdictions the brief names. Do not add states by association.
- For `constraints`, surface anything explicit: *"non-dilutive only"*, *"no
  defense work"*, *"avoid grants requiring matching funds"*. Leave the list
  empty if the brief is silent.
- Validate before publishing. If validation fails, fix the offending field
  and re-validate rather than relaxing the schema.
