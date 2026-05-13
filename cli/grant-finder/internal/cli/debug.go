package cli

import "github.com/spf13/cobra"

func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Maintainer-only source and ledger operations",
		Long:  "Maintainer-only source and ledger operations. These are deterministic substrate tools, not the public agent interface.",
	}
	cmd.AddCommand(newSyncCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newChangesCmd())
	cmd.AddCommand(newSqlCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newFederalRegisterCmd())
	cmd.AddCommand(newFeedsCmd())
	cmd.AddCommand(newGrantsCmd())
	cmd.AddCommand(newSourcesCmd())
	cmd.AddCommand(newDebugSeedCmd())
	return cmd
}
