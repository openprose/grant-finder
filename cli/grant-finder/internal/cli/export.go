// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export commands",
	}
	cmd.AddCommand(newExportopmlCmd())

	return cmd
}
