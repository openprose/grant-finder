package cli

import "github.com/spf13/cobra"

func newStatsCmd() *cobra.Command {
	var dbPath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Print ledger counts and last-run metadata",
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
			if asJSON {
				return printJSON(cmd.OutOrStdout(), stats)
			}
			rows := [][]string{
				{"runs", fmtAny(stats.Runs)},
				{"raw_items", fmtAny(stats.RawItems)},
				{"opportunities", fmtAny(stats.Opportunities)},
				{"opportunity_sources", fmtAny(stats.OpportunitySources)},
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
	return cmd
}
