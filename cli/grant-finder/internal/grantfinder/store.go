package grantfinder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const DefaultDBName = "grant-finder.sqlite"

var (
	spaceRE      = regexp.MustCompile(`\s+`)
	oppNumberRE  = regexp.MustCompile(`(?i)\b(?:DE-FOA-[0-9A-Z-]+|[A-Z]{2,}-[A-Z0-9-]{3,}|[0-9]{2,}-[0-9A-Z-]{4,})\b`)
	frDocumentRE = regexp.MustCompile(`/documents/\d{4}/\d{2}/\d{2}/([^/?#]+)`)
)

type Store struct {
	DB *sql.DB
}

type Opportunity struct {
	ID                int64    `json:"id,omitempty"`
	DedupeKey         string   `json:"dedupe_key"`
	RecordType        string   `json:"record_type,omitempty"`
	Title             string   `json:"title,omitempty"`
	Sponsor           string   `json:"sponsor,omitempty"`
	URL               string   `json:"url,omitempty"`
	ApplyURL          string   `json:"apply_url,omitempty"`
	DeadlineText      string   `json:"deadline_text,omitempty"`
	Published         string   `json:"published,omitempty"`
	OpportunityNumber string   `json:"opportunity_number,omitempty"`
	DocumentNumber    string   `json:"document_number,omitempty"`
	Canonicality      string   `json:"canonicality,omitempty"`
	PublicationBasis  string   `json:"publication_basis,omitempty"`
	Summary           string   `json:"summary,omitempty"`
	Eligibility       string   `json:"eligibility,omitempty"`
	RawSignals        []string `json:"raw_signals,omitempty"`
	SourceRefs        []Ref    `json:"source_refs,omitempty"`
}

type Ref struct {
	SourceID  string `json:"source_id"`
	SourceURL string `json:"source_url,omitempty"`
	RawID     string `json:"raw_id,omitempty"`
}

type Change struct {
	ID            int64  `json:"id"`
	OpportunityID int64  `json:"opportunity_id"`
	DedupeKey     string `json:"dedupe_key"`
	ChangeType    string `json:"change_type"`
	ChangedAt     string `json:"changed_at"`
	Summary       string `json:"summary"`
	Title         string `json:"title,omitempty"`
	URL           string `json:"url,omitempty"`
}

type Stats struct {
	Runs               int64  `json:"runs"`
	RawItems           int64  `json:"raw_items"`
	Opportunities      int64  `json:"opportunities"`
	OpportunitySources int64  `json:"opportunity_sources"`
	Changes            int64  `json:"changes"`
	FTSRows            int64  `json:"fts_rows"`
	LastRunStartedAt   string `json:"last_run_started_at,omitempty"`
	LastRunFinishedAt  string `json:"last_run_finished_at,omitempty"`
	LastRunCountsJSON  string `json:"last_run_counts_json,omitempty"`
}

func DefaultDBPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "grant-finder", DefaultDBName)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return DefaultDBName
	}
	return filepath.Join(home, ".local", "share", "grant-finder", DefaultDBName)
}

func OpenStore(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		path = DefaultDBPath()
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating db directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{DB: db}
	if err := s.Init(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) Init(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL`,
		`CREATE TABLE IF NOT EXISTS runs (id INTEGER PRIMARY KEY AUTOINCREMENT, started_at TEXT NOT NULL, finished_at TEXT, command_json TEXT, counts_json TEXT)`,
		`CREATE TABLE IF NOT EXISTS raw_items (id INTEGER PRIMARY KEY AUTOINCREMENT, source_id TEXT NOT NULL, raw_id TEXT, url TEXT, title TEXT, fetched_at TEXT NOT NULL, payload_json TEXT NOT NULL, payload_hash TEXT NOT NULL, UNIQUE(source_id, raw_id))`,
		`CREATE TABLE IF NOT EXISTS opportunities (id INTEGER PRIMARY KEY AUTOINCREMENT, dedupe_key TEXT NOT NULL UNIQUE, record_type TEXT, title TEXT, sponsor TEXT, url TEXT, apply_url TEXT, deadline_text TEXT, published TEXT, opportunity_number TEXT, document_number TEXT, canonicality TEXT, publication_basis TEXT, content_hash TEXT NOT NULL, first_seen TEXT NOT NULL, last_seen TEXT NOT NULL, payload_json TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS opportunity_sources (id INTEGER PRIMARY KEY AUTOINCREMENT, opportunity_id INTEGER NOT NULL, source_id TEXT NOT NULL, source_url TEXT, raw_id TEXT, last_seen TEXT NOT NULL, UNIQUE(opportunity_id, source_id, raw_id))`,
		`CREATE TABLE IF NOT EXISTS changes (id INTEGER PRIMARY KEY AUTOINCREMENT, opportunity_id INTEGER, dedupe_key TEXT NOT NULL, change_type TEXT NOT NULL, changed_at TEXT NOT NULL, old_hash TEXT, new_hash TEXT, summary TEXT, payload_json TEXT, run_id INTEGER)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS opportunity_search USING fts5(title, sponsor, record_type, summary, eligibility, deadline_text, raw_signals, url)`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) StartRun(ctx context.Context, command any) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	commandJSON, _ := json.Marshal(command)
	res, err := s.DB.ExecContext(ctx, `INSERT INTO runs(started_at, command_json) VALUES (?, ?)`, now, string(commandJSON))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishRun(ctx context.Context, runID int64, counts any) error {
	now := time.Now().UTC().Format(time.RFC3339)
	countsJSON, _ := json.Marshal(counts)
	_, err := s.DB.ExecContext(ctx, `UPDATE runs SET finished_at=?, counts_json=? WHERE id=?`, now, string(countsJSON), runID)
	return err
}

func (s *Store) UpsertOpportunity(ctx context.Context, runID int64, sourceID, rawID, sourceURL string, op Opportunity, payload any) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	op.DedupeKey = DedupeKey(op)
	if op.DedupeKey == "" {
		return "", fmt.Errorf("missing dedupe key for %q", op.Title)
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadHash := HashJSON(payload)
	_, _ = s.DB.ExecContext(ctx, `INSERT OR REPLACE INTO raw_items(source_id, raw_id, url, title, fetched_at, payload_json, payload_hash) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sourceID, rawID, sourceURL, op.Title, now, string(payloadJSON), payloadHash)

	recordJSON, _ := json.Marshal(op)
	contentHash := HashJSON(op)
	var existingID int64
	var oldHash string
	err := s.DB.QueryRowContext(ctx, `SELECT id, content_hash FROM opportunities WHERE dedupe_key=?`, op.DedupeKey).Scan(&existingID, &oldHash)
	status := "unchanged"
	switch {
	case err == sql.ErrNoRows:
		res, err := s.DB.ExecContext(ctx, `INSERT INTO opportunities(dedupe_key, record_type, title, sponsor, url, apply_url, deadline_text, published, opportunity_number, document_number, canonicality, publication_basis, content_hash, first_seen, last_seen, payload_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			op.DedupeKey, op.RecordType, op.Title, op.Sponsor, op.URL, op.ApplyURL, op.DeadlineText, op.Published, op.OpportunityNumber, op.DocumentNumber, op.Canonicality, op.PublicationBasis, contentHash, now, now, string(recordJSON))
		if err != nil {
			return "", err
		}
		existingID, _ = res.LastInsertId()
		_, _ = s.DB.ExecContext(ctx, `INSERT INTO changes(opportunity_id, dedupe_key, change_type, changed_at, new_hash, summary, payload_json, run_id) VALUES (?, ?, 'first_seen', ?, ?, ?, ?, ?)`, existingID, op.DedupeKey, now, contentHash, op.Title, string(recordJSON), runID)
		status = "new"
	case err != nil:
		return "", err
	default:
		_, err := s.DB.ExecContext(ctx, `UPDATE opportunities SET record_type=?, title=?, sponsor=?, url=?, apply_url=?, deadline_text=?, published=?, opportunity_number=?, document_number=?, canonicality=?, publication_basis=?, content_hash=?, last_seen=?, payload_json=? WHERE id=?`,
			op.RecordType, op.Title, op.Sponsor, op.URL, op.ApplyURL, op.DeadlineText, op.Published, op.OpportunityNumber, op.DocumentNumber, op.Canonicality, op.PublicationBasis, contentHash, now, string(recordJSON), existingID)
		if err != nil {
			return "", err
		}
		if oldHash != contentHash {
			status = "updated"
			_, _ = s.DB.ExecContext(ctx, `INSERT INTO changes(opportunity_id, dedupe_key, change_type, changed_at, old_hash, new_hash, summary, payload_json, run_id) VALUES (?, ?, 'updated', ?, ?, ?, ?, ?, ?)`, existingID, op.DedupeKey, now, oldHash, contentHash, op.Title, string(recordJSON), runID)
		}
	}
	_, _ = s.DB.ExecContext(ctx, `INSERT OR REPLACE INTO opportunity_sources(opportunity_id, source_id, source_url, raw_id, last_seen) VALUES (?, ?, ?, ?, ?)`, existingID, sourceID, sourceURL, rawID, now)
	if err := s.syncSearch(ctx, existingID, op); err != nil {
		return "", err
	}
	return status, nil
}

func (s *Store) syncSearch(ctx context.Context, id int64, op Opportunity) error {
	_, _ = s.DB.ExecContext(ctx, `DELETE FROM opportunity_search WHERE rowid=?`, id)
	_, err := s.DB.ExecContext(ctx, `INSERT INTO opportunity_search(rowid, title, sponsor, record_type, summary, eligibility, deadline_text, raw_signals, url) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, op.Title, op.Sponsor, op.RecordType, op.Summary, op.Eligibility, op.DeadlineText, strings.Join(op.RawSignals, " "), op.URL)
	return err
}

type SearchResult struct {
	ID           int64   `json:"id"`
	Rank         float64 `json:"rank"`
	Title        string  `json:"title"`
	Sponsor      string  `json:"sponsor,omitempty"`
	RecordType   string  `json:"record_type,omitempty"`
	DeadlineText string  `json:"deadline_text,omitempty"`
	URL          string  `json:"url,omitempty"`
	DedupeKey    string  `json:"dedupe_key"`
}

type OpportunityRecord struct {
	Opportunity
	FirstSeen  string `json:"first_seen,omitempty"`
	LastSeen   string `json:"last_seen,omitempty"`
	SearchRank any    `json:"search_rank,omitempty"`
}

func (s *Store) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT o.id, bm25(opportunity_search) AS rank, o.title, o.sponsor, o.record_type, o.deadline_text, o.url, o.dedupe_key FROM opportunity_search JOIN opportunities o ON opportunity_search.rowid=o.id WHERE opportunity_search MATCH ? ORDER BY rank LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Rank, &r.Title, &r.Sponsor, &r.RecordType, &r.DeadlineText, &r.URL, &r.DedupeKey); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) OpportunityByID(ctx context.Context, id int64) (OpportunityRecord, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, dedupe_key, record_type, title, sponsor, url, apply_url, deadline_text, published, opportunity_number, document_number, canonicality, publication_basis, first_seen, last_seen, payload_json FROM opportunities WHERE id=?`, id)
	return scanOpportunityRecord(ctx, s, row)
}

// CoverageMatch reports whether the ledger contains any opportunity whose
// title, sponsor, or URL contains any of the supplied needles (case-insensitive).
// Empty needles list returns (false, nil) — no needles means no match.
//
// This is the truthful-coverage check: it answers "did this lane contribute
// records to the ledger?" rather than "did a record from this lane survive
// to the top-N ranking?"
func (s *Store) CoverageMatch(ctx context.Context, needles []string) (bool, error) {
	if len(needles) == 0 {
		return false, nil
	}
	var conditions []string
	var args []any
	for _, needle := range needles {
		needle = strings.TrimSpace(needle)
		if needle == "" {
			continue
		}
		pattern := "%" + strings.ToLower(needle) + "%"
		conditions = append(conditions, "(LOWER(title) LIKE ? OR LOWER(sponsor) LIKE ? OR LOWER(url) LIKE ?)")
		args = append(args, pattern, pattern, pattern)
	}
	if len(conditions) == 0 {
		return false, nil
	}
	query := "SELECT 1 FROM opportunities WHERE " + strings.Join(conditions, " OR ") + " LIMIT 1"
	var hit int
	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&hit)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) OpportunityByKey(ctx context.Context, key string) (OpportunityRecord, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id, dedupe_key, record_type, title, sponsor, url, apply_url, deadline_text, published, opportunity_number, document_number, canonicality, publication_basis, first_seen, last_seen, payload_json FROM opportunities WHERE dedupe_key=? OR opportunity_number=? OR document_number=?`, key, key, key)
	return scanOpportunityRecord(ctx, s, row)
}

func (s *Store) AllOpportunityRecords(ctx context.Context, limit int) ([]OpportunityRecord, error) {
	query := `SELECT id, dedupe_key, record_type, title, sponsor, url, apply_url, deadline_text, published, opportunity_number, document_number, canonicality, publication_basis, first_seen, last_seen, payload_json FROM opportunities ORDER BY last_seen DESC, id DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OpportunityRecord
	for rows.Next() {
		rec, err := scanOpportunityRecordFromScanner(ctx, s, rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) OpportunitySources(ctx context.Context, opportunityID int64) ([]Ref, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT source_id, COALESCE(source_url, ''), COALESCE(raw_id, '') FROM opportunity_sources WHERE opportunity_id=? ORDER BY source_id, raw_id`, opportunityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []Ref
	for rows.Next() {
		var ref Ref
		if err := rows.Scan(&ref.SourceID, &ref.SourceURL, &ref.RawID); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

type opportunityScanner interface {
	Scan(dest ...any) error
}

func scanOpportunityRecord(ctx context.Context, s *Store, scanner opportunityScanner) (OpportunityRecord, error) {
	return scanOpportunityRecordFromScanner(ctx, s, scanner)
}

func scanOpportunityRecordFromScanner(ctx context.Context, s *Store, scanner opportunityScanner) (OpportunityRecord, error) {
	var rec OpportunityRecord
	var payloadJSON string
	if err := scanner.Scan(&rec.ID, &rec.DedupeKey, &rec.RecordType, &rec.Title, &rec.Sponsor, &rec.URL, &rec.ApplyURL, &rec.DeadlineText, &rec.Published, &rec.OpportunityNumber, &rec.DocumentNumber, &rec.Canonicality, &rec.PublicationBasis, &rec.FirstSeen, &rec.LastSeen, &payloadJSON); err != nil {
		return rec, err
	}
	var payload Opportunity
	if payloadJSON != "" && json.Unmarshal([]byte(payloadJSON), &payload) == nil {
		if payload.Summary != "" {
			rec.Summary = payload.Summary
		}
		if payload.Eligibility != "" {
			rec.Eligibility = payload.Eligibility
		}
		if payload.RawSignals != nil {
			rec.RawSignals = payload.RawSignals
		}
	}
	refs, err := s.OpportunitySources(ctx, rec.ID)
	if err == nil {
		rec.SourceRefs = refs
	}
	return rec, nil
}

func (s *Store) Changes(ctx context.Context, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, COALESCE(opportunity_id, 0), dedupe_key, change_type, changed_at, COALESCE(summary, '') FROM changes ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Change
	for rows.Next() {
		var c Change
		if err := rows.Scan(&c.ID, &c.OpportunityID, &c.DedupeKey, &c.ChangeType, &c.ChangedAt, &c.Summary); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) RecentChanges(ctx context.Context, limit int) ([]Change, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT c.id, COALESCE(c.opportunity_id, 0), c.dedupe_key, c.change_type, c.changed_at, COALESCE(c.summary, ''), COALESCE(o.title, c.summary, ''), COALESCE(o.url, '') FROM changes c LEFT JOIN opportunities o ON o.id=c.opportunity_id ORDER BY c.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Change
	for rows.Next() {
		var c Change
		if err := rows.Scan(&c.ID, &c.OpportunityID, &c.DedupeKey, &c.ChangeType, &c.ChangedAt, &c.Summary, &c.Title, &c.URL); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	counts := []struct {
		query string
		dest  *int64
	}{
		{`SELECT COUNT(*) FROM runs`, &st.Runs},
		{`SELECT COUNT(*) FROM raw_items`, &st.RawItems},
		{`SELECT COUNT(*) FROM opportunities`, &st.Opportunities},
		{`SELECT COUNT(*) FROM opportunity_sources`, &st.OpportunitySources},
		{`SELECT COUNT(*) FROM changes`, &st.Changes},
		{`SELECT COUNT(*) FROM opportunity_search`, &st.FTSRows},
	}
	for _, c := range counts {
		if err := s.DB.QueryRowContext(ctx, c.query).Scan(c.dest); err != nil {
			return st, err
		}
	}
	_ = s.DB.QueryRowContext(ctx, `SELECT COALESCE(started_at, ''), COALESCE(finished_at, ''), COALESCE(counts_json, '') FROM runs ORDER BY id DESC LIMIT 1`).Scan(&st.LastRunStartedAt, &st.LastRunFinishedAt, &st.LastRunCountsJSON)
	return st, nil
}

func (s *Store) StatsMap(ctx context.Context) (map[string]any, error) {
	st, err := s.Stats(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"runs":                st.Runs,
		"raw_items":           st.RawItems,
		"opportunities":       st.Opportunities,
		"opportunity_sources": st.OpportunitySources,
		"changes":             st.Changes,
		"fts_rows":            st.FTSRows,
	}
	if st.LastRunStartedAt != "" {
		out["last_run"] = map[string]any{
			"started_at":  st.LastRunStartedAt,
			"finished_at": st.LastRunFinishedAt,
			"counts_json": st.LastRunCountsJSON,
		}
	}
	return out, nil
}

func DedupeKey(op Opportunity) string {
	if op.OpportunityNumber != "" {
		return "opportunity-number:" + strings.ToLower(op.OpportunityNumber)
	}
	if op.DocumentNumber != "" {
		return "federal-register:" + strings.ToLower(op.DocumentNumber)
	}
	if op.Title != "" && op.URL != "" {
		return "title-url:" + HashString(strings.ToLower(cleanText(op.Title)+"|"+NormalizeURL(op.URL)+"|"+cleanText(op.Sponsor)+"|"+cleanText(op.Published)))
	}
	if op.URL != "" {
		return "url:" + NormalizeURL(op.URL)
	}
	return ""
}

func HashJSON(v any) string {
	data, _ := json.Marshal(v)
	return HashString(string(data))
}

func HashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func NormalizeURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			q.Del(key)
		}
	}
	u.RawQuery = q.Encode()
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func cleanText(s string) string {
	return spaceRE.ReplaceAllString(strings.TrimSpace(s), " ")
}

func GuessOpportunityNumber(text string) string {
	for _, candidate := range oppNumberRE.FindAllString(text, -1) {
		for _, r := range candidate {
			if r >= '0' && r <= '9' {
				return strings.TrimSpace(candidate)
			}
		}
	}
	return ""
}

func GuessFRDocumentNumber(rawURL string) string {
	m := frDocumentRE.FindStringSubmatch(rawURL)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func SortedSignals(signals []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, signal := range signals {
		signal = strings.TrimSpace(signal)
		if signal != "" && !seen[signal] {
			seen[signal] = true
			out = append(out, signal)
		}
	}
	sort.Strings(out)
	return out
}
