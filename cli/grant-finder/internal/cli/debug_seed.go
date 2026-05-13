package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

type seedFixture struct {
	Opportunities []seedOpportunity `json:"opportunities"`
}

type seedOpportunity struct {
	SourceID    string                  `json:"source_id"`
	RawID       string                  `json:"raw_id"`
	SourceURL   string                  `json:"source_url"`
	Opportunity grantfinder.Opportunity `json:"opportunity"`
}

func newDebugSeedCmd() *cobra.Command {
	var dbPath string
	var fixturePath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "seed-fixture",
		Short: "Seed deterministic fixture opportunities into the ledger",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fixturePath == "" {
				return fmt.Errorf("--fixture is required")
			}
			data, err := os.ReadFile(fixturePath)
			if err != nil {
				return err
			}
			var fixture seedFixture
			if err := json.Unmarshal(data, &fixture); err != nil {
				return err
			}
			store, err := openStore(cmd.Context(), dbPath)
			if err != nil {
				return err
			}
			defer store.Close()
			runID, err := store.StartRun(cmd.Context(), map[string]any{"command": "debug seed-fixture", "fixture": fixturePath})
			if err != nil {
				return err
			}
			counts := map[string]int{"seeded": 0, "new": 0, "updated": 0, "unchanged": 0}
			for _, item := range fixture.Opportunities {
				status, err := store.UpsertOpportunity(cmd.Context(), runID, item.SourceID, item.RawID, item.SourceURL, item.Opportunity, item)
				if err != nil {
					return err
				}
				counts["seeded"]++
				counts[status]++
			}
			if err := store.FinishRun(cmd.Context(), runID, counts); err != nil {
				return err
			}
			if asJSON {
				out := map[string]any{"run_id": runID, "counts": counts}
				return printJSON(cmd.OutOrStdout(), out)
			}
			printRows(cmd.OutOrStdout(), [][]string{
				{"run_id", fmtAny(runID)},
				{"seeded", fmtInt(counts["seeded"])},
				{"new", fmtInt(counts["new"])},
				{"updated", fmtInt(counts["updated"])},
				{"unchanged", fmtInt(counts["unchanged"])},
			})
			return nil
		},
	}
	addDBFlag(cmd, &dbPath)
	addJSONFlag(cmd, &asJSON)
	cmd.Flags().StringVar(&fixturePath, "fixture", "", "Fixture JSON path")
	return cmd
}
