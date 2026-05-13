package cli

import (
	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newSyncCmd() *cobra.Command {
	var dbPath string
	var limit int
	var keyword string
	var asJSON bool
	var feedsOnly bool
	var grantsOnly bool
	var grantsXML bool
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Run read-only ingestion into the local provenance ledger",
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := grantfinder.Sync(cmd.Context(), grantfinder.SyncOptions{
				DBPath:        dbPath,
				Limit:         limit,
				Keyword:       keyword,
				IncludeFeeds:  !grantsOnly,
				IncludeGrants: !feedsOnly,
				IncludeXML:    grantsXML,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), report)
			}
			printRows(cmd.OutOrStdout(), [][]string{
				{"run_id", fmtAny(report.RunID)},
				{"feeds_checked", fmtInt(report.FeedsChecked)},
				{"grants_checked", fmtInt(report.GrantsChecked)},
				{"xml_checked", fmtInt(report.XMLChecked)},
				{"items", fmtInt(report.Items)},
				{"new", fmtInt(report.New)},
				{"updated", fmtInt(report.Updated)},
				{"unchanged", fmtInt(report.Unchanged)},
				{"errors", fmtInt(report.Errors)},
			})
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addLimitFlag(cmd, &limit, 25)
	cmd.Flags().StringVar(&keyword, "keyword", "SBIR", "Grants.gov keyword")
	cmd.Flags().BoolVar(&feedsOnly, "feeds-only", false, "Only ingest configured feed/page sources")
	cmd.Flags().BoolVar(&grantsOnly, "grants-only", false, "Only query Grants.gov")
	cmd.Flags().BoolVar(&grantsXML, "grants-xml", false, "Also ingest matching rows from the Grants.gov XML extract")
	addJSONFlag(cmd, &asJSON)
	return cmd
}
