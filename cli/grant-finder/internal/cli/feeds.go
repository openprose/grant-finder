// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
)

func newFeedsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feeds",
		Short: "feeds commands",
	}
	cmd.AddCommand(newFeedssmokeCmd())
	cmd.AddCommand(newFeedslistCmd())

	return cmd
}
