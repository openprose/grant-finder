package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newExportopmlCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "opml",
		Short: "Write the grouped grant-finder OPML feed list",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := grantfinder.OPML()
			if output == "" || output == "-" {
				_, err := cmd.OutOrStdout().Write(data)
				return err
			}
			return os.WriteFile(output, data, 0o644)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "-", "Output path, or - for stdout")
	return cmd
}
