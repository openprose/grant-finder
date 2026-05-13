package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openprose/grant-finder/cli/grant-finder/internal/grantfinder"
)

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type outputOptions struct {
	Select  string
	Compact bool
}

func printJSONWithOptions(w io.Writer, v any, opts outputOptions) error {
	out := v
	if strings.TrimSpace(opts.Select) != "" {
		out = selectJSONFields(v, opts.Select)
	}
	enc := json.NewEncoder(w)
	if !opts.Compact {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(out)
}

func selectJSONFields(v any, selectExpr string) map[string]any {
	var raw any
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"error": "select failed: " + err.Error()}
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]any{"error": "select failed: " + err.Error()}
	}
	out := map[string]any{}
	for _, part := range strings.Split(selectExpr, ",") {
		path := strings.TrimSpace(part)
		if path == "" {
			continue
		}
		out[path] = valuesAtPath(raw, strings.Split(path, "."))
	}
	return out
}

func valuesAtPath(v any, parts []string) any {
	if len(parts) == 0 {
		return v
	}
	switch cur := v.(type) {
	case map[string]any:
		next, ok := cur[parts[0]]
		if !ok {
			return nil
		}
		return valuesAtPath(next, parts[1:])
	case []any:
		values := make([]any, 0, len(cur))
		for _, item := range cur {
			values = append(values, valuesAtPath(item, parts))
		}
		return values
	default:
		return nil
	}
}

func printRows(w io.Writer, rows [][]string) {
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}

func addDBFlag(cmd *cobra.Command, dbPath *string) {
	cmd.Flags().StringVar(dbPath, "db", grantfinder.DefaultDBPath(), "SQLite database path")
}

func addJSONFlag(cmd *cobra.Command, asJSON *bool) {
	cmd.Flags().BoolVar(asJSON, "json", false, "Print JSON output")
}

func addSelectFlag(cmd *cobra.Command, selectFields *string) {
	cmd.Flags().StringVar(selectFields, "select", "", "Comma-separated JSON fields to return")
}

func addCompactFlag(cmd *cobra.Command, compact *bool) {
	cmd.Flags().BoolVar(compact, "compact", false, "Print compact JSON")
}

func addLimitFlag(cmd *cobra.Command, limit *int, value int) {
	cmd.Flags().IntVar(limit, "limit", value, "Maximum rows to return")
}

func openStore(ctx context.Context, dbPath string) (*grantfinder.Store, error) {
	return grantfinder.OpenStore(ctx, dbPath)
}

func ensureReadOnlySQL(query string) error {
	q := strings.TrimSpace(strings.ToLower(query))
	if strings.HasPrefix(q, "select") || strings.HasPrefix(q, "with") || strings.HasPrefix(q, "pragma") {
		return nil
	}
	return fmt.Errorf("only read-only SQL is allowed (SELECT, WITH, PRAGMA)")
}

func queryRows(ctx context.Context, db *sql.DB, query string, args ...any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		record := map[string]any{}
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				record[col] = string(v)
			default:
				record[col] = v
			}
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func fmtInt(v int) string {
	return strconv.Itoa(v)
}

func fmtAny(v any) string {
	return fmt.Sprint(v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
