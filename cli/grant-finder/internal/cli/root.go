// Copyright 2026 OpenProse contributors. Licensed under MIT. See LICENSE.

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

// Execute runs the CLI.
func Execute() error {
	rootCmd := &cobra.Command{
		Use:          "grant-finder",
		Short:        "Provenance-first grant and founder-opportunity ledger",
		SilenceUsage: true,
		Version:      version,
	}
	rootCmd.SetVersionTemplate("grant-finder {{ .Version }}\n")
	rootCmd.AddCommand(newResearchCmd())
	rootCmd.AddCommand(newExplainCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newDebugCmd())
	rootCmd.AddCommand(newAgentContextCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newVersionCliCmd())

	return rootCmd.Execute()
}

// ExitCode extracts exit code from an error (always 1 for now).
func ExitCode(err error) int {
	return 1
}

func newVersionCliCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("grant-finder %s\n", version)
		},
	}
}

// suggestFlag is a placeholder for flag suggestion support.
var _ = strings.Contains
