package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newFederalregisterhydrateCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "hydrate <document-number>",
		Short: "Hydrate a Federal Register document number with JSON metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := grantfinder.HydrateFederalRegister(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), record)
			}
			return printJSON(cmd.OutOrStdout(), record)
		},
	}
	addJSONFlag(cmd, &asJSON)
	return cmd
}
