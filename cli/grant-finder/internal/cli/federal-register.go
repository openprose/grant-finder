// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newFederalRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "federal-register",
		Short: "federal-register commands",
	}
	cmd.AddCommand(newFederalregisterhydrateCmd())

	return cmd
}
