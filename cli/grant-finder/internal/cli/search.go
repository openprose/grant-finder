package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var dbPath string
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search normalized opportunities with SQLite FTS5",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openStore(cmd.Context(), dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			results, err := store.Search(cmd.Context(), strings.Join(args, " "), limit)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), results)
			}
			rows := [][]string{{"TITLE", "SPONSOR", "DEADLINE", "URL"}}
			for _, r := range results {
				rows = append(rows, []string{truncate(r.Title, 70), truncate(r.Sponsor, 30), r.DeadlineText, r.URL})
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addLimitFlag(cmd, &limit, 10)
	addJSONFlag(cmd, &asJSON)
	return cmd
}
