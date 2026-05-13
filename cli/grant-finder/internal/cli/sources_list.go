package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newSourceslistCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured broad grant/fellowship sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources, err := grantfinder.Sources()
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), sources)
			}
			rows := [][]string{{"ID", "CATEGORY", "SURFACE", "URL"}}
			for _, s := range sources {
				rows = append(rows, []string{s.ID, s.Category, s.Surface, s.URL})
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addJSONFlag(cmd, &asJSON)
	return cmd
}
