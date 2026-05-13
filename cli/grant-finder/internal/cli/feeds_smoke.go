package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newFeedssmokeCmd() *cobra.Command {
	var limit int
	var timeout int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "smoke",
		Short: "Fetch configured feed/page sources and count parseable items",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := grantfinder.SmokeFeeds(cmd.Context(), limit, time.Duration(timeout)*time.Second)
			if err != nil {
				return err
			}
			summary := grantfinder.SmokeSummary(results)
			if asJSON {
				return printJSON(cmd.OutOrStdout(), summary)
			}
			rows := [][]string{{"ID", "OK", "STATUS", "ITEMS", "ERROR"}}
			for _, r := range results {
				rows = append(rows, []string{r.ID, fmtAny(r.OK), fmtInt(r.StatusCode), fmtInt(r.ItemCount), truncate(r.Error, 80)})
			}
			printRows(cmd.OutOrStdout(), rows)
			return nil
		},
	}
	addLimitFlag(cmd, &limit, 20)
	cmd.Flags().IntVar(&timeout, "timeout", 15, "Per-source timeout in seconds")
	addJSONFlag(cmd, &asJSON)
	return cmd
}
