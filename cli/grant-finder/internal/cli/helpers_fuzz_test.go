package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func FuzzSelectJSONFields(f *testing.F) {
	f.Add("retrieval.backend,grants.program_name")
	f.Add("coverage.source_lane,missing.path")
	f.Add(",,grants.evidence.claim")

	packet := map[string]any{
		"retrieval": map[string]any{
			"backend": "fts5",
			"no_llm":  true,
		},
		"grants": []map[string]any{
			{
				"program_name": "Example SBIR",
				"evidence": []map[string]any{
					{"claim": "official source"},
				},
			},
		},
		"coverage": []map[string]any{
			{"source_lane": "ARPA-E", "status": "checked"},
		},
	}

	f.Fuzz(func(t *testing.T, selectExpr string) {
		if len(selectExpr) > 4096 {
			return
		}
		out := selectJSONFields(packet, selectExpr)
		if out == nil {
			t.Fatal("selectJSONFields returned nil")
		}
		if _, ok := out[""]; ok {
			t.Fatal("selectJSONFields emitted an empty path")
		}
		if _, err := json.Marshal(out); err != nil {
			t.Fatalf("selected output is not JSON-marshalable: %v", err)
		}
	})
}

func FuzzEnsureReadOnlySQL(f *testing.F) {
	f.Add("select * from opportunities")
	f.Add("with rows as (select 1) select * from rows")
	f.Add("pragma table_info(opportunities)")
	f.Add("select 1; drop table opportunities")
	f.Add("selective")
	f.Add("delete from opportunities")

	f.Fuzz(func(t *testing.T, query string) {
		if len(query) > 4096 {
			return
		}
		err := ensureReadOnlySQL(query)
		if err != nil {
			return
		}
		trimmed := strings.TrimSpace(query)
		token, ok := firstSQLToken(trimmed)
		if !ok {
			t.Fatalf("accepted query without a first token: %q", query)
		}
		switch strings.ToLower(token) {
		case "select", "with", "pragma":
		default:
			t.Fatalf("accepted non-read-only first token %q in query %q", token, query)
		}
		if hasStackedSQLStatement(trimmed) {
			t.Fatalf("accepted stacked SQL statement: %q", query)
		}
	})
}
