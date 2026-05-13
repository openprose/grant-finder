package cli

import "github.com/spf13/cobra"

func newAgentContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-context",
		Short: "Describe the command surface for agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			context := map[string]any{
				"name":        "grant-finder",
				"description": "Agent-facing grant deep-research substrate. The upstream LLM uses this CLI; the CLI itself does not call an LLM. Retrieval prefers local usearch semantic search and falls back to SQLite FTS5.",
				"commands": []map[string]string{
					{"command": "research --assignment assignment.json --json", "purpose": "Return a deterministic Research Packet: candidate opportunities, evidence, provenance, preliminary fit signals, deadlines, coverage, and negative evidence."},
					{"command": "research --assignment assignment.json --semantic usearch --json", "purpose": "Force local usearch semantic retrieval over the opportunity corpus, with FTS5 supplement/fallback."},
					{"command": "explain rec-123 --json", "purpose": "Explain the source evidence and provenance behind a candidate opportunity."},
					{"command": "status --assignment assignment.json --json", "purpose": "Report ledger freshness and source-lane coverage for an assignment."},
					{"command": "debug sync --db /tmp/grants.sqlite --limit 30 --json", "purpose": "Maintainer-only deterministic refresh; normal agents should prefer research --refresh auto."},
				},
				"anti_triggers": []string{
					"Do not ask the human user for startup profile fields; the upstream agent provides the assignment JSON.",
					"Do not expect this CLI to make final eligibility judgments; it returns deterministic evidence for the upstream agent.",
					"Do not call an LLM from inside this CLI.",
					"Do not use this CLI to submit grant applications.",
					"Do not use this CLI for SAM.gov keyed ingestion until a key path is configured.",
					"Do not treat source/debug commands as the product interface.",
				},
			}
			return printJSON(cmd.OutOrStdout(), context)
		},
	}
}
