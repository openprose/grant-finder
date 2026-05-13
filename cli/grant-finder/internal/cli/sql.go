package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func newSqlCmd() *cobra.Command {
	var dbPath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "sql <query>",
		Short: "Run read-only SQL against the local SQLite store",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			if err := ensureReadOnlySQL(query); err != nil {
				return err
			}
			store, err := openStore(cmd.Context(), dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			rows, err := queryRows(cmd.Context(), store.DB, query)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), rows)
			}
			return printJSON(cmd.OutOrStdout(), rows)
		},
	}
	addDBFlag(cmd, &dbPath)
	addJSONFlag(cmd, &asJSON)
	return cmd
}
