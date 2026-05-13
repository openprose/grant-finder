package grantfinder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	tokenRE        = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9_]{2,}`)
	corpusIDRE     = regexp.MustCompile(`opportunity-([0-9]+)\.md$`)
	searchStopword = map[string]bool{
		"and": true, "are": true, "for": true, "from": true, "into": true,
		"non": true, "the": true, "this": true, "that": true, "with": true,
		"without": true, "startup": true, "company": true, "funding": true,
	}
)

func CandidateRecords(ctx context.Context, store *Store, dbPath, query, semanticMode string, limit int) ([]OpportunityRecord, string, error) {
	if limit <= 0 {
		limit = 10
	}
	semanticMode = strings.ToLower(strings.TrimSpace(semanticMode))
	if semanticMode == "" {
		semanticMode = "auto"
	}
	seen := map[int64]bool{}
	var records []OpportunityRecord
	backend := "fts5"

	if semanticMode == "auto" || semanticMode == "usearch" {
		usearchRecords, err := usearchCandidateRecords(ctx, store, dbPath, query, limit)
		if err == nil && len(usearchRecords) > 0 {
			backend = "usearch"
			for _, rec := range usearchRecords {
				if !seen[rec.ID] {
					seen[rec.ID] = true
					records = append(records, rec)
				}
			}
		} else if semanticMode == "usearch" {
			if err != nil {
				backend = "usearch_unavailable_fts5_fallback"
			} else {
				backend = "usearch_empty_fts5_fallback"
			}
		}
	}

	if len(records) < limit {
		fts, err := ftsCandidateRecords(ctx, store, query, limit*2)
		if err != nil && len(records) == 0 {
			return nil, backend, err
		}
		if len(records) > 0 {
			backend += "+fts5"
		}
		for _, rec := range fts {
			if !seen[rec.ID] {
				seen[rec.ID] = true
				records = append(records, rec)
				if len(records) >= limit {
					break
				}
			}
		}
	}

	if len(records) == 0 {
		all, err := store.AllOpportunityRecords(ctx, limit)
		if err != nil {
			return nil, backend, err
		}
		backend = "ledger_recent"
		records = all
	}
	if len(records) > limit {
		records = records[:limit]
	}
	return records, backend, nil
}

func ftsCandidateRecords(ctx context.Context, store *Store, query string, limit int) ([]OpportunityRecord, error) {
	ftsQuery := BuildFTSQuery(query)
	if ftsQuery == "" {
		return store.AllOpportunityRecords(ctx, limit)
	}
	results, err := store.Search(ctx, ftsQuery, limit)
	if err != nil {
		return store.AllOpportunityRecords(ctx, limit)
	}
	records := make([]OpportunityRecord, 0, len(results))
	for _, result := range results {
		rec, err := store.OpportunityByID(ctx, result.ID)
		if err != nil {
			continue
		}
		rec.SearchRank = result.Rank
		records = append(records, rec)
	}
	return records, nil
}

func usearchCandidateRecords(ctx context.Context, store *Store, dbPath, query string, limit int) ([]OpportunityRecord, error) {
	if _, err := exec.LookPath("usearch"); err != nil {
		return nil, err
	}
	corpusDir, err := ExportUsearchCorpus(ctx, store, DefaultUsearchCorpusDir(dbPath))
	if err != nil {
		return nil, err
	}
	args := []string{"-r", "--json", "-k", strconv.Itoa(limit), query, corpusDir}
	cmd := exec.CommandContext(ctx, "usearch", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Unit struct {
			File string `json:"file"`
		} `json:"unit"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	records := make([]OpportunityRecord, 0, len(rows))
	for _, row := range rows {
		id := opportunityIDFromCorpusFile(row.Unit.File)
		if id == 0 {
			continue
		}
		rec, err := store.OpportunityByID(ctx, id)
		if err != nil {
			continue
		}
		rec.SearchRank = row.Score
		records = append(records, rec)
	}
	return records, nil
}

func DefaultUsearchCorpusDir(dbPath string) string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "grant-finder", "usearch", HashString(dbPath)[:16])
}

func ExportUsearchCorpus(ctx context.Context, store *Store, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	records, err := store.AllOpportunityRecords(ctx, 0)
	if err != nil {
		return "", err
	}
	for _, rec := range records {
		path := filepath.Join(dir, fmt.Sprintf("opportunity-%d.md", rec.ID))
		body := opportunityCorpusMarkdown(rec)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func opportunityCorpusMarkdown(rec OpportunityRecord) string {
	parts := []string{
		fmt.Sprintf("id: %d", rec.ID),
		"title: " + rec.Title,
		"sponsor: " + rec.Sponsor,
		"type: " + rec.RecordType,
		"deadline: " + rec.DeadlineText,
		"url: " + rec.URL,
		"summary: " + rec.Summary,
		"eligibility: " + rec.Eligibility,
		"signals: " + strings.Join(rec.RawSignals, ", "),
	}
	for _, ref := range rec.SourceRefs {
		parts = append(parts, "source: "+strings.TrimSpace(ref.SourceID+" "+ref.SourceURL))
	}
	return strings.Join(parts, "\n") + "\n"
}

func opportunityIDFromCorpusFile(path string) int64 {
	base := filepath.Base(path)
	match := corpusIDRE.FindStringSubmatch(base)
	if len(match) != 2 {
		return 0
	}
	id, _ := strconv.ParseInt(match[1], 10, 64)
	return id
}

func BuildFTSQuery(query string) string {
	seen := map[string]bool{}
	var terms []string
	for _, token := range tokenRE.FindAllString(strings.ToLower(query), -1) {
		token = strings.Trim(token, "_")
		if token == "" || searchStopword[token] || seen[token] {
			continue
		}
		seen[token] = true
		terms = append(terms, token)
		if len(terms) >= 16 {
			break
		}
	}
	return strings.Join(terms, " OR ")
}
