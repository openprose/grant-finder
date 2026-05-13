package cli

import (
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func newDoctorCmd() *cobra.Command {
	var dbPath string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check manifest loading, SQLite, and FTS5 health",
		RunE: func(cmd *cobra.Command, args []string) error {
			feeds, feedsErr := grantfinder.Feeds()
			sources, sourcesErr := grantfinder.Sources()
			store, storeErr := grantfinder.OpenStore(cmd.Context(), dbPath)
			var stats any
			if storeErr == nil {
				defer store.Close()
				stats, _ = store.Stats(cmd.Context())
			}
			usearchPath, usearchErr := exec.LookPath("usearch")
			report := map[string]any{
				"no_llm":             true,
				"go":                 runtime.Version(),
				"os":                 runtime.GOOS,
				"arch":               runtime.GOARCH,
				"agent_interface":    []string{"research", "explain", "status"},
				"semantic_retrieval": "usearch preferred, FTS5 fallback",
				"usearch_path":       usearchPath,
				"usearch_error":      errString(usearchErr),
				"feeds":              len(feeds),
				"feeds_error":        errString(feedsErr),
				"sources":            len(sources),
				"sources_error":      errString(sourcesErr),
				"db_path":            dbPath,
				"db_error":           errString(storeErr),
				"stats":              stats,
			}
			if asJSON {
				return printJSON(cmd.OutOrStdout(), report)
			}
			return printJSON(cmd.OutOrStdout(), report)
		},
	}
	addDBFlag(cmd, &dbPath)
	addJSONFlag(cmd, &asJSON)
	return cmd
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
