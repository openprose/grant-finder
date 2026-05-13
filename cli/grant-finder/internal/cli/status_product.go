package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newStatusCmd() *cobra.Command {
	var dbPath string
	var assignmentPath string
	var asJSON bool
	var compact bool
	var selectFields string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report grant research substrate freshness and coverage",
		Long:  "Report grant research substrate freshness and coverage for an optional assignment. The command is deterministic and does not call an LLM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context(), dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			stats, err := store.Stats(cmd.Context())
			if err != nil {
				return err
			}
			out := map[string]any{
				"no_llm": true,
				"stats":  stats,
			}
			if assignmentPath != "" {
				assignment, err := readAssignment(assignmentPath)
				if err != nil {
					return err
				}
				out["assignment_id"] = assignment.AssignmentID
				out["coverage"] = grantfinder.BuildCoverage(assignment, nil)
			}
			if asJSON || compact || selectFields != "" {
				return printJSONWithOptions(cmd.OutOrStdout(), out, outputOptions{Select: selectFields, Compact: compact})
			}
			rows := [][]string{
				{"opportunities", fmtAny(stats.Opportunities)},
				{"raw_items", fmtAny(stats.RawItems)},
				{"changes", fmtAny(stats.Changes)},
				{"fts_rows", fmtAny(stats.FTSRows)},
				{"last_run_started_at", stats.LastRunStartedAt},
				{"last_run_finished_at", stats.LastRunFinishedAt},
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addJSONFlag(cmd, &asJSON)
	addSelectFlag(cmd, &selectFields)
	addCompactFlag(cmd, &compact)
	cmd.Flags().StringVar(&assignmentPath, "assignment", "", "Optional research assignment JSON path, or '-' for stdin")
	return cmd
}
