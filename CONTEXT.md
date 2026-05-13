# Grant Finder Context

Grant Finder is a reusable grant deep-research substrate for agents. It lets an
upstream agent answer funding-opportunity requests from accumulated evidence,
provenance, and deterministic background work instead of reinventing source
discovery and grant research for every user request.

## Language

**Agent Interface**:
The CLI boundary an upstream agent calls to translate a user request into deterministic searches, syncs, explanations, and recommendations.
_Avoid_: Human CLI, operator console, admin tool

**Upstream User Request**:
The natural-language task given to the agent, such as finding relevant non-dilutive funding for a startup.
_Avoid_: CLI invocation, scraper job

**Resolved Agent Request**:
The agent's already-interpreted task context, including the customer/startup facts, constraints, goals, and preferences it gathered upstream.
_Avoid_: Intake form, CLI profile setup, user questionnaire

**Research Assignment**:
The specific startup, context, technology, and funding question that an upstream agent passes to Grant Finder for one grant deep-research run.
_Avoid_: User profile, source query, scraper configuration

**Background Collector**:
A deterministic scraper, feed reader, API client, or resolver that gathers raw evidence without deciding what the user should do.
_Avoid_: User command, interactive workflow

**Opportunity Ledger**:
The local database of normalized opportunities, evidence, provenance, changes, and search indexes used to answer agent requests.
_Avoid_: Cache, scrape dump, feed reader

**Opportunity Radar**:
The agent-facing product behavior that finds, ranks, explains, and tracks opportunities for a specific user or company context.
_Avoid_: Source monitor, RSS inbox

**Research Packet**:
The structured answer Grant Finder returns to an upstream agent: candidate opportunities, preliminary fit rationale, evidence, provenance, negative evidence, freshness, and suggested next steps. Final ranking and recommendation judgment belong to the caller.
_Avoid_: Search results page, scrape output, raw export

**Grant Deep Research**:
The agent's funding-opportunity investigation workflow over sources, evidence, fit, changes, gaps, and recommended next actions.
_Avoid_: Grant search, source scraping, feed browsing

**Research Substrate**:
The reusable evidence, ledger, collectors, resolvers, and indexes that make **Grant Deep Research** fast across repeated requests.
_Avoid_: One-off research run, temporary cache

**Semantic Retrieval**:
Deterministic local retrieval over opportunity/evidence text using `usearch` as the preferred semantic backend and FTS5 as the structured fallback.
_Avoid_: LLM judgment, prompt completion, web search

**Evidence**:
Source-backed facts used to justify why an opportunity exists, changed, or fits a request.
_Avoid_: Content blob, raw page

**Provenance**:
The link between a ledger record and every source or collector observation that supports it.
_Avoid_: Referrer, scrape metadata

**Resolver**:
A background collector that turns a weak signal into a more canonical or complete record.
_Avoid_: Public CLI command, hydration workflow

**Deterministic Work**:
Any collection, parsing, resolving, deduping, scoring, or bookkeeping step that can run safely without agent judgment.
_Avoid_: User choice, public command surface

**Agent Judgment**:
The non-deterministic interpretation layer that decides relevance, fit, explanation, prioritization, and next action for an upstream user request.
_Avoid_: Scraping, hydration, database maintenance

**Automatic Refresh**:
The agent-interface behavior that runs any stale deterministic collectors or resolvers before answering, without asking the upstream agent to operate source-specific commands.
_Avoid_: Manual sync step, public hydrate command

## Relationships

- An **Upstream User Request** is handled by an agent through the **Agent Interface**.
- The upstream agent already knows the customer or startup context before it calls the **Agent Interface**.
- The **Agent Interface** receives a **Resolved Agent Request** and helps the agent go fast; it must not make the user restate context through CLI flags or prompts.
- A **Research Assignment** is the unit of work for the **Agent Interface**: one startup/context/technology investigation in, one **Research Packet** out.
- The **Agent Interface** reads from and may refresh the **Opportunity Ledger**.
- **Background Collectors** write raw observations and normalized records into the **Opportunity Ledger**.
- A **Resolver** is a type of **Background Collector**.
- **Deterministic Work** belongs behind the **Agent Interface** and should run automatically when needed.
- **Automatic Refresh** is how **Deterministic Work** stays invisible to the upstream agent during normal use.
- **Agent Judgment** should consume **Evidence** and **Provenance** rather than ask the user to operate collectors.
- **Evidence** and **Provenance** explain why an **Opportunity Ledger** record should be trusted.
- The **Research Substrate** prevents every **Grant Deep Research** request from restarting source discovery from scratch.
- **Semantic Retrieval** helps the upstream LLM agent find relevant ledger records quickly, but it does not replace the agent's final judgment.
- The **Opportunity Radar** is the outcome the agent exposes to the user; collectors and resolvers are implementation machinery behind it.
- A **Research Packet** must include negative evidence for must-check sources when no matching opportunity exists.

## Example Dialogue

> **Dev:** "Should `federal-register hydrate` be a top-level command?"
> **Domain expert:** "No. Federal Register hydration is a **Resolver** behind the **Agent Interface**. The agent should ask the ledger what changed, what fits, and what evidence supports it."

## Flagged Ambiguities

- "CLI" can mean a human-facing tool or an **Agent Interface**. Resolved: this project is an agent interface, not a human operator console.
- "Find grants" can mean scraping sources or answering an **Upstream User Request**. Resolved: the product behavior is the answer/ranking/explanation layer; scraping is background collection.
- "Command" can mean a public agent-facing capability or a deterministic maintenance action. Resolved: deterministic work should be automated behind the agent-facing capability unless it is needed for debugging.
- "Profile" can sound like something this CLI gathers from a human. Resolved: customer/startup context belongs upstream with the agent; the CLI receives a **Resolved Agent Request** and returns fast evidence-backed results.
- "Grant finder" can sound like a search box. Resolved: the intended behavior is **Grant Deep Research** backed by a reusable **Research Substrate**.
- "Hydration" can sound like a user workflow. Resolved: source hydration is a **Resolver** and belongs inside **Automatic Refresh** or debug tooling, not the primary product interface.
- "Semantic search" can sound like LLM generation. Resolved: the upstream LLM uses this CLI; the CLI itself uses deterministic retrieval and must not call an LLM.
