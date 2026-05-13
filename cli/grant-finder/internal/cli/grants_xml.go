package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newGrantsxmlCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "xml",
		Short: "Check the Grants.gov XML extract page and latest ZIP URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := grantfinder.LatestGrantsXMLExtract(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), info)
			}
			return printJSON(cmd.OutOrStdout(), info)
		},
	}
	addJSONFlag(cmd, &asJSON)
	return cmd
}
