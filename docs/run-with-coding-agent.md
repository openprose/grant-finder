# Run Grant Finder With Claude Code or Codex

This page is for people who want a coding assistant to run Grant Finder for
them. You are using the product; you are not contributing to the repo.

Paste the block below into Claude Code, Codex, or another coding agent. The
agent will install the local CLI, run the OpenProse Grant Radar example, and
return a source-cited funding report.

Before you start, have a paragraph ready that describes the research lab,
startup, nonprofit team, or technical project you want funding for. Grant Finder
is currently U.S.-focused and does not apply to grants, email sponsors, or
submit anything on your behalf.

## Copy-Paste Prompt

````markdown
You are helping me run OpenProse Grant Finder as a product user.

My goal is to get a source-cited report of non-dilutive funding opportunities
for my research lab, startup, or project. I am not asking you to contribute code
to the repo.

Follow these rules:

- Use the public `openprose/grant-finder` repo.
- Do not add API keys or paid services.
- Do not wire Exa, SAM.gov, browser automation, OpenAI, Anthropic, Stripe, or
  any other paid or keyed service into Grant Finder retrieval.
- Do not put an LLM inside the Go CLI.
- Treat the CLI as deterministic retrieval and provenance machinery.
- Use the OpenProse agent layer to judge fit and write the final report.
- If there are no credible matches, say that directly. Do not force weak
  recommendations.
- Do not apply, email, submit, register, purchase, or mutate any upstream
  service.

First, check prerequisites:

```bash
command -v git
command -v go
command -v prose
```

If any of those are missing, stop and tell me exactly which prerequisite is
missing. Do not fake a report.

Then clone or update Grant Finder:

```bash
WORKDIR="${HOME}/openprose-grant-finder-run"
mkdir -p "$WORKDIR"
cd "$WORKDIR"

if [ -d grant-finder/.git ]; then
  cd grant-finder
  git pull --ff-only
else
  git clone https://github.com/openprose/grant-finder.git
  cd grant-finder
fi
```

Build the CLI and make sure it is on PATH:

```bash
mkdir -p "$HOME/.local/bin"
cd "$WORKDIR/grant-finder/cli/grant-finder"
go build -o "$HOME/.local/bin/grant-finder" ./cmd/grant-finder
export PATH="$HOME/.local/bin:$PATH"
grant-finder version
```

Install the host-harness skill for whichever agent harness is available. Do not
overwrite an existing custom skill directory; if a path already exists and is
not a symlink, stop and tell me.

```bash
cd "$WORKDIR/grant-finder"

install_skill_link() {
  skills_dir="$1"
  target="$skills_dir/grant-finder"
  mkdir -p "$skills_dir"
  if [ -e "$target" ] && [ ! -L "$target" ]; then
    echo "Existing non-symlink skill path: $target"
    echo "Leaving it untouched. Ask me before changing it."
    return 1
  fi
  ln -sfn "$PWD/skills/grant-finder" "$target"
}

install_skill_link "$HOME/.claude/skills" || exit 1
install_skill_link "$HOME/.codex/skills" || exit 1
install_skill_link "$HOME/.agents/skills" || exit 1
```

Now get my brief:

- If I included a research-lab, startup, nonprofit, or project brief in this chat, save it to
  `$WORKDIR/funding-brief.txt`.
- If I did not include a brief, ask me for one before running anything else.

Run the OpenProse Grant Radar example from the repo root:

```bash
cd "$WORKDIR/grant-finder"

PROSE_CODEX_SANDBOX_MODE=workspace-write \
PROSE_CODEX_APPROVAL_POLICY=never \
PROSE_CODEX_ADD_DIR="$HOME/.local/share/grant-finder" \
PROSE_CODEX_NETWORK=true \
prose run examples/openprose/src/grant-radar.prose.md \
  --startup_brief "$(cat "$WORKDIR/funding-brief.txt")"
```

If that fails only because the installed `prose` version does not support the
granular sandbox env vars, retry once with the broader fallback:

```bash
cd "$WORKDIR/grant-finder"

PROSE_CODEX_SANDBOX_MODE=danger-full-access \
PROSE_CODEX_APPROVAL_POLICY=never \
prose run examples/openprose/src/grant-radar.prose.md \
  --startup_brief "$(cat "$WORKDIR/funding-brief.txt")"
```

When the run finishes, show me:

- the final Markdown report
- where the run artifacts were written
- any credible matches and any rejected weak matches
- any important caveats about source coverage or freshness

Do not summarize unsupported opportunities as good fits. Every recommended
opportunity must have source evidence from the run.
````

## What The Agent Should Return

A good final response from the coding assistant should include the Markdown
funding report, the run path, and a short note about confidence and gaps. If
the report has no credible current opportunities, that is a valid result.

If the agent reports missing prerequisites, install those first and run the same
prompt again.
