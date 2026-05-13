package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newFeedslistCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured RSS/page/newsletter feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			feeds, err := grantfinder.Feeds()
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), feeds)
			}
			rows := [][]string{{"ID", "TYPE", "ACCESS", "CANONICALITY", "URL"}}
			for _, f := range feeds {
				rows = append(rows, []string{f.ID, f.Type, f.Access, f.Canonicality, f.URL})
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addJSONFlag(cmd, &asJSON)
	return cmd
}
