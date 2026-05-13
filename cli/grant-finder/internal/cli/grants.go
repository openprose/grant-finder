// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newGrantsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grants",
		Short: "grants commands",
	}
	cmd.AddCommand(newGrantssearchCmd())
	cmd.AddCommand(newGrantsfetchCmd())
	cmd.AddCommand(newGrantsxmlCmd())

	return cmd
}
