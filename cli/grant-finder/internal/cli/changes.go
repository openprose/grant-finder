package cli

import "github.com/spf13/cobra"

func newChangesCmd() *cobra.Command {
	var dbPath string
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Show recently first-seen or changed opportunities",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context(), dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			changes, err := store.Changes(cmd.Context(), limit)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), changes)
			}
			rows := [][]string{{"WHEN", "TYPE", "TITLE", "URL"}}
			for _, c := range changes {
				rows = append(rows, []string{c.ChangedAt, c.ChangeType, truncate(c.Summary, 70), c.DedupeKey})
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addLimitFlag(cmd, &limit, 25)
	addJSONFlag(cmd, &asJSON)
	return cmd
}
