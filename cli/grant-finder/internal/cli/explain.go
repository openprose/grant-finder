package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newExplainCmd() *cobra.Command {
	var dbPath string
	var asJSON bool
	var compact bool
	var selectFields string
	cmd := &cobra.Command{
		Use:   "explain <recommendation-id|opportunity-id|dedupe-key>",
		Short: "Explain evidence and provenance for a recommendation",
		Long:  "Explain evidence and provenance for a recommendation or opportunity. The command is deterministic and does not call an LLM.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			key := args[0]
			if len(key) > 4 && key[:4] == "rec-" {
				key = key[4:]
			}
			packet, err := grantfinder.Explain(cmd.Context(), dbPath, key)
			if err != nil {
				return err
			}
			if asJSON || compact || selectFields != "" {
				return printJSONWithOptions(cmd.OutOrStdout(), packet, outputOptions{Select: selectFields, Compact: compact})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "program\t%s\nagency\t%s\nurl\t%s\n", packet.Opportunity.Title, packet.Opportunity.Sponsor, packet.Opportunity.URL)
			for _, ev := range packet.Evidence {
				fmt.Fprintf(cmd.OutOrStdout(), "evidence\t%s\t%s\t%s\n", ev.SourceID, ev.URL, truncate(ev.Claim, 90))
			}
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addJSONFlag(cmd, &asJSON)
	addSelectFlag(cmd, &selectFields)
	addCompactFlag(cmd, &compact)
	return cmd
}
