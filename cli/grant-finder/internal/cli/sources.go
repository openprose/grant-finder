// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newSourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "sources commands",
	}
	cmd.AddCommand(newSourcessmokeCmd())
	cmd.AddCommand(newSourceslistCmd())

	return cmd
}
