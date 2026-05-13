package grantfinder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Assignment struct {
	AssignmentID      string         `json:"assignment_id"`
	ResearchQuestion  string         `json:"research_question,omitempty"`
	CompanyProfile    CompanyProfile `json:"company_profile"`
	FocusAreas        []string       `json:"focus_areas"`
	TargetGeographies []string       `json:"target_geographies"`
	KnownGrants       []KnownGrant   `json:"known_grants"`
}

type CompanyProfile struct {
	Name         string   `json:"name,omitempty"`
	Description  string   `json:"description"`
	Stage        string   `json:"stage,omitempty"`
	Location     string   `json:"location,omitempty"`
	Technologies []string `json:"technologies,omitempty"`
	Constraints  []string `json:"constraints,omitempty"`
}

type KnownGrant struct {
	ProgramName   string `json:"program_name,omitempty"`
	OpportunityID string `json:"opportunity_id,omitempty"`
	URL           string `json:"url,omitempty"`
}

type ResearchOptions struct {
	DBPath          string
	Limit           int
	Refresh         string
	Semantic        string
	Compact         bool
	IncludeInactive bool
	Now             time.Time
}

type ResearchPacket struct {
	AssignmentID string                `json:"assignment_id"`
	GeneratedAt  string                `json:"generated_at"`
	Retrieval    RetrievalInfo         `json:"retrieval"`
	Summary      ResearchSummary       `json:"summary"`
	Grants       []GrantRecommendation `json:"grants"`
	Coverage     []CoverageRow         `json:"coverage"`
}

type RetrievalInfo struct {
	Backend string `json:"backend"`
	Query   string `json:"query"`
	NoLLM   bool   `json:"no_llm"`
}

type ResearchSummary struct {
	TotalPotentialFunding string   `json:"total_potential_funding"`
	HighFitCount          int      `json:"high_fit_count"`
	NearestDeadline       *string  `json:"nearest_deadline"`
	Notes                 []string `json:"notes,omitempty"`
}

type GrantRecommendation struct {
	RecommendationID   string         `json:"recommendation_id"`
	OpportunityID      int64          `json:"opportunity_id"`
	ProgramName        string         `json:"program_name"`
	Agency             string         `json:"agency"`
	Amount             string         `json:"amount"`
	Deadline           *string        `json:"deadline"`
	DeadlineCertainty  string         `json:"deadline_certainty"`
	EligibilityFit     FitAssessment  `json:"eligibility_fit"`
	EffortEstimate     FitAssessment  `json:"effort_estimate"`
	ActivityStatus     FitAssessment  `json:"activity_status"`
	URL                string         `json:"url"`
	ApplicationOutline []string       `json:"application_outline,omitempty"`
	Evidence           []EvidenceItem `json:"evidence"`
	Score              int            `json:"score,omitempty"`
}

type FitAssessment struct {
	Level       string `json:"level"`
	Explanation string `json:"explanation"`
}

type EvidenceItem struct {
	SourceID string `json:"source_id"`
	URL      string `json:"url"`
	Claim    string `json:"claim"`
}

type CoverageRow struct {
	SourceLane string `json:"source_lane"`
	Status     string `json:"status"`
	Note       string `json:"note,omitempty"`
}

type ExplainPacket struct {
	Opportunity OpportunityRecord `json:"opportunity"`
	Evidence    []EvidenceItem    `json:"evidence"`
	Sources     []Ref             `json:"sources"`
	Notes       []string          `json:"notes,omitempty"`
	NoLLM       bool              `json:"no_llm"`
}

func ParseAssignment(data []byte) (Assignment, error) {
	var assignment Assignment
	if err := json.Unmarshal(data, &assignment); err != nil {
		return assignment, err
	}
	if strings.TrimSpace(assignment.AssignmentID) == "" {
		return assignment, fmt.Errorf("assignment_id is required")
	}
	if strings.TrimSpace(assignment.CompanyProfile.Description) == "" {
		return assignment, fmt.Errorf("company_profile.description is required")
	}
	if assignment.FocusAreas == nil {
		assignment.FocusAreas = []string{}
	}
	if assignment.TargetGeographies == nil {
		assignment.TargetGeographies = []string{}
	}
	if assignment.KnownGrants == nil {
		assignment.KnownGrants = []KnownGrant{}
	}
	return assignment, nil
}

func Research(ctx context.Context, opts ResearchOptions, assignment Assignment) (ResearchPacket, error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Refresh == "" {
		opts.Refresh = "auto"
	}
	if opts.Semantic == "" {
		opts.Semantic = "auto"
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.Refresh == "auto" {
		if err := refreshIfEmpty(ctx, opts, assignment); err != nil {
			return ResearchPacket{}, err
		}
	}
	store, err := OpenStore(ctx, opts.DBPath)
	if err != nil {
		return ResearchPacket{}, err
	}
	defer store.Close()

	query := BuildAssignmentQuery(assignment)
	records, backend, err := CandidateRecords(ctx, store, opts.DBPath, query, opts.Semantic, opts.Limit*20)
	if err != nil {
		return ResearchPacket{}, err
	}
	recs := make([]GrantRecommendation, 0, opts.Limit)
	seen := map[int64]bool{}
	for _, record := range records {
		if seen[record.ID] || IsKnownGrant(assignment, record) {
			continue
		}
		activity := AssessActivity(record, opts.Now)
		if !opts.IncludeInactive && activity.Level == "inactive" {
			continue
		}
		seen[record.ID] = true
		rec := BuildRecommendation(assignment, record, activity)
		recs = append(recs, rec)
	}
	sort.SliceStable(recs, func(i, j int) bool {
		if recs[i].Score != recs[j].Score {
			return recs[i].Score > recs[j].Score
		}
		return deadlineSortKey(recs[i].Deadline) < deadlineSortKey(recs[j].Deadline)
	})
	if len(recs) > opts.Limit {
		recs = recs[:opts.Limit]
	}
	coverageRecs := recs
	if len(records) == 0 {
		coverageRecs = nil
	}
	packet := ResearchPacket{
		AssignmentID: assignment.AssignmentID,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Retrieval: RetrievalInfo{
			Backend: backend,
			Query:   query,
			NoLLM:   true,
		},
		Summary:  BuildSummary(recs, opts.IncludeInactive),
		Grants:   recs,
		Coverage: BuildCoverage(assignment, coverageRecs),
	}
	return packet, nil
}

func Explain(ctx context.Context, dbPath, idOrKey string) (ExplainPacket, error) {
	store, err := OpenStore(ctx, dbPath)
	if err != nil {
		return ExplainPacket{}, err
	}
	defer store.Close()
	var rec OpportunityRecord
	if id, err := strconv.ParseInt(idOrKey, 10, 64); err == nil {
		rec, err = store.OpportunityByID(ctx, id)
		if err != nil {
			return ExplainPacket{}, err
		}
	} else {
		rec, err = store.OpportunityByKey(ctx, idOrKey)
		if err != nil {
			if err == sql.ErrNoRows {
				return ExplainPacket{}, fmt.Errorf("opportunity not found: %s", idOrKey)
			}
			return ExplainPacket{}, err
		}
	}
	return ExplainPacket{
		Opportunity: rec,
		Evidence:    evidenceForOpportunity(rec),
		Sources:     rec.SourceRefs,
		Notes:       []string{"deterministic explanation; no LLM call was made"},
		NoLLM:       true,
	}, nil
}

func BuildAssignmentQuery(a Assignment) string {
	parts := []string{a.ResearchQuestion, a.CompanyProfile.Description, a.CompanyProfile.Stage, a.CompanyProfile.Location}
	parts = append(parts, a.CompanyProfile.Technologies...)
	parts = append(parts, a.FocusAreas...)
	parts = append(parts, a.TargetGeographies...)
	return strings.Join(parts, " ")
}

func BuildRecommendation(a Assignment, rec OpportunityRecord, activity FitAssessment) GrantRecommendation {
	fit := AssessFit(a, rec)
	effort := EstimateEffort(rec)
	certainty := DeadlineCertainty(rec.DeadlineText)
	var deadline *string
	if strings.TrimSpace(rec.DeadlineText) != "" {
		d := rec.DeadlineText
		deadline = &d
	}
	score := fitScore(fit.Level)*100 + evidenceScore(rec) - effortPenalty(effort.Level)
	recommendation := GrantRecommendation{
		RecommendationID:  fmt.Sprintf("rec-%d", rec.ID),
		OpportunityID:     rec.ID,
		ProgramName:       fallback(rec.Title, "Unknown program"),
		Agency:            fallback(rec.Sponsor, "Unknown agency"),
		Amount:            "unknown from current evidence",
		Deadline:          deadline,
		DeadlineCertainty: certainty,
		EligibilityFit:    fit,
		EffortEstimate:    effort,
		ActivityStatus:    activity,
		URL:               rec.URL,
		Evidence:          evidenceForOpportunity(rec),
		Score:             score,
	}
	if fit.Level == "high" {
		recommendation.ApplicationOutline = []string{
			"Company and technology overview",
			"Problem, deployment context, and public benefit",
			"Technical approach and work plan",
			"Commercialization or deployment plan",
			"Budget, milestones, and partner commitments",
		}
	}
	return recommendation
}

func AssessActivity(rec OpportunityRecord, now time.Time) FitAssessment {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	haystack := strings.ToLower(strings.Join([]string{
		rec.Title,
		rec.Sponsor,
		rec.DeadlineText,
		rec.Published,
		rec.Summary,
		strings.Join(rec.RawSignals, " "),
	}, " "))

	switch {
	case hasActivitySignal(rec, "archived") || hasStatusPhrase(haystack, "archived"):
		return FitAssessment{Level: "inactive", Explanation: "Source status indicates archived."}
	case hasActivitySignal(rec, "closed", "closed_solicitation") || hasStatusPhrase(haystack, "closed"):
		return FitAssessment{Level: "inactive", Explanation: "Source status indicates closed."}
	case hasActivitySignal(rec, "expired") || hasStatusPhrase(haystack, "expired"):
		return FitAssessment{Level: "inactive", Explanation: "Source status indicates expired."}
	}

	if deadline, ok := ParseOpportunityDate(rec.DeadlineText); ok {
		if deadline.Before(today) {
			return FitAssessment{Level: "inactive", Explanation: "Deadline is past due: " + deadline.Format("2006-01-02") + "."}
		}
		return FitAssessment{Level: "active", Explanation: "Deadline is current: " + deadline.Format("2006-01-02") + "."}
	}

	deadlineText := strings.ToLower(strings.TrimSpace(rec.DeadlineText))
	switch {
	case strings.Contains(deadlineText, "accepted anytime"), strings.Contains(deadlineText, "continuous"), strings.Contains(deadlineText, "rolling"):
		return FitAssessment{Level: "active", Explanation: "Deadline language indicates a rolling or anytime submission window."}
	case strings.Contains(haystack, "posted"), strings.Contains(haystack, "forecasted"):
		return FitAssessment{Level: "active", Explanation: "Source status indicates posted or forecasted and no past deadline was found."}
	case strings.Contains(deadlineText, "awaiting"), strings.Contains(deadlineText, "nofo"), strings.Contains(deadlineText, "projected"):
		return FitAssessment{Level: "active", Explanation: "Opportunity is awaiting or projecting a future NOFO."}
	}

	if published, ok := ParseOpportunityDate(rec.Published); ok {
		staleCutoff := today.AddDate(-2, 0, 0)
		if published.Before(staleCutoff) {
			return FitAssessment{Level: "inactive", Explanation: "No current deadline or active status; publication is stale: " + published.Format("2006-01-02") + "."}
		}
	}

	return FitAssessment{Level: "active", Explanation: "No closed, archived, expired, or past-due signal was found."}
}

func hasActivitySignal(rec OpportunityRecord, values ...string) bool {
	want := map[string]bool{}
	for _, value := range values {
		want[strings.ToLower(strings.TrimSpace(value))] = true
	}
	for _, signal := range rec.RawSignals {
		normalized := strings.ToLower(strings.TrimSpace(signal))
		if want[normalized] {
			return true
		}
	}
	return false
}

func hasStatusPhrase(haystack, status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	for _, phrase := range []string{
		"status: " + status,
		"opportunity status: " + status,
		"oppstatus: " + status,
		"opp status: " + status,
		"source status indicates " + status,
	} {
		if strings.Contains(haystack, phrase) {
			return true
		}
	}
	return false
}

func ParseOpportunityDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	value = strings.TrimSuffix(value, ".")
	for _, layout := range []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"Jan 02, 2006",
		"January 02, 2006",
		"Jan 2, 2006",
		"January 2, 2006",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func AssessFit(a Assignment, rec OpportunityRecord) FitAssessment {
	haystack := strings.ToLower(strings.Join([]string{
		rec.Title, rec.Sponsor, rec.RecordType, rec.Summary, rec.Eligibility,
		strings.Join(rec.RawSignals, " "),
	}, " "))
	score := 0
	var matched []string
	for _, term := range append(a.FocusAreas, a.CompanyProfile.Technologies...) {
		term = strings.ToLower(strings.TrimSpace(term))
		if term != "" && strings.Contains(haystack, term) {
			score += 2
			matched = append(matched, term)
		}
	}
	for _, geo := range a.TargetGeographies {
		geo = strings.ToLower(strings.TrimSpace(geo))
		if geo != "" && strings.Contains(haystack, geo) {
			score++
			matched = append(matched, geo)
		}
	}
	if strings.Contains(haystack, "sbir") || strings.Contains(haystack, "sttr") {
		score += 2
		matched = append(matched, "SBIR/STTR")
	}
	switch {
	case score >= 5:
		return FitAssessment{Level: "high", Explanation: "Matched strong assignment signals: " + strings.Join(uniqueStrings(matched), ", ")}
	case score >= 2:
		return FitAssessment{Level: "medium", Explanation: "Some assignment signals matched; agent should verify eligibility details against the official source."}
	default:
		return FitAssessment{Level: "low", Explanation: "Few explicit assignment signals matched in current ledger evidence."}
	}
}

func EstimateEffort(rec OpportunityRecord) FitAssessment {
	haystack := strings.ToLower(strings.Join([]string{rec.Title, rec.Summary, rec.Eligibility}, " "))
	switch {
	case strings.Contains(haystack, "letter of intent") || strings.Contains(haystack, "matching funds") || strings.Contains(haystack, "consortium") || strings.Contains(haystack, "partnership"):
		return FitAssessment{Level: "high", Explanation: "Evidence suggests LOI, matching funds, partnership, or consortium work."}
	case strings.Contains(haystack, "proposal") || strings.Contains(haystack, "application") || strings.Contains(haystack, "phase ii"):
		return FitAssessment{Level: "medium", Explanation: "Likely requires a structured application or technical proposal."}
	default:
		return FitAssessment{Level: "low", Explanation: "No high-effort application signals found in current evidence."}
	}
}

func DeadlineCertainty(deadline string) string {
	d := strings.ToLower(strings.TrimSpace(deadline))
	switch {
	case d == "":
		return "unknown"
	case strings.Contains(d, "awaiting") || strings.Contains(d, "nofo"):
		return "awaiting_nofo"
	case strings.Contains(d, "projected") || strings.Contains(d, "estimated"):
		return "projected"
	default:
		return "confirmed"
	}
}

func BuildSummary(recs []GrantRecommendation, includeInactive bool) ResearchSummary {
	high := 0
	var nearest *string
	for _, rec := range recs {
		if rec.EligibilityFit.Level == "high" {
			high++
		}
		if rec.Deadline != nil && (nearest == nil || deadlineSortKey(rec.Deadline) < deadlineSortKey(nearest)) {
			d := *rec.Deadline
			nearest = &d
		}
	}
	notes := []string{"ranking is deterministic; no LLM call was made inside the CLI"}
	if !includeInactive {
		notes = append(notes, "inactive opportunities are filtered by default; pass --include-inactive for historical comps")
	}
	return ResearchSummary{
		TotalPotentialFunding: "unknown from current evidence",
		HighFitCount:          high,
		NearestDeadline:       nearest,
		Notes:                 notes,
	}
}

func BuildCoverage(a Assignment, recs []GrantRecommendation) []CoverageRow {
	rows := []CoverageRow{
		{SourceLane: "Grants.gov", Status: statusFromMatches(recs, "grants.gov"), Note: "canonical federal opportunity lane"},
		{SourceLane: "SBIR/STTR", Status: statusFromMatches(recs, "sbir", "sttr"), Note: "small business funding lane"},
		{SourceLane: "ARPA-E", Status: statusFromMatches(recs, "arpa-e", "advanced research projects agency energy"), Note: "No current ARPA-E programs match"},
		{SourceLane: "DOE EERE", Status: statusFromMatches(recs, "eere", "energy efficiency"), Note: "energy funding lane"},
		{SourceLane: "NSF", Status: statusFromMatches(recs, "national science foundation", "nsf"), Note: "research and commercialization lane"},
	}
	for _, geo := range a.TargetGeographies {
		geo = strings.TrimSpace(geo)
		if geo == "" || strings.EqualFold(geo, "United States") {
			continue
		}
		rows = append(rows, CoverageRow{
			SourceLane: "state economic development: " + geo,
			Status:     statusFromMatches(recs, strings.ToLower(geo)),
			Note:       "state-specific source lane required by assignment geography",
		})
	}
	return rows
}

func statusFromMatches(recs []GrantRecommendation, needles ...string) string {
	if recs == nil {
		return "not_checked"
	}
	for _, rec := range recs {
		haystack := strings.ToLower(strings.Join([]string{rec.ProgramName, rec.Agency, rec.URL}, " "))
		for _, needle := range needles {
			if strings.Contains(haystack, strings.ToLower(needle)) {
				return "matched"
			}
		}
	}
	return "checked_no_match"
}

func IsKnownGrant(a Assignment, rec OpportunityRecord) bool {
	for _, known := range a.KnownGrants {
		if known.OpportunityID != "" && (strings.EqualFold(known.OpportunityID, rec.OpportunityNumber) || strings.EqualFold(known.OpportunityID, rec.DedupeKey)) {
			return true
		}
		if known.URL != "" && NormalizeURL(known.URL) == NormalizeURL(rec.URL) {
			return true
		}
		if known.ProgramName != "" && strings.EqualFold(cleanText(known.ProgramName), cleanText(rec.Title)) {
			return true
		}
	}
	return false
}

func evidenceForOpportunity(rec OpportunityRecord) []EvidenceItem {
	claim := fallback(rec.Summary, rec.Title)
	if claim == "" {
		claim = "Opportunity exists in the local ledger."
	}
	var out []EvidenceItem
	if len(rec.SourceRefs) == 0 {
		return []EvidenceItem{{SourceID: "ledger", URL: rec.URL, Claim: claim}}
	}
	for _, ref := range rec.SourceRefs {
		out = append(out, EvidenceItem{
			SourceID: fallback(ref.SourceID, "ledger"),
			URL:      fallback(ref.SourceURL, rec.URL),
			Claim:    claim,
		})
	}
	return out
}

func refreshIfEmpty(ctx context.Context, opts ResearchOptions, assignment Assignment) error {
	store, err := OpenStore(ctx, opts.DBPath)
	if err != nil {
		return err
	}
	stats, err := store.Stats(ctx)
	_ = store.Close()
	if err != nil {
		return err
	}
	if stats.Opportunities > 0 {
		return nil
	}
	_, err = RunSync(ctx, SyncOptions{
		DBPath:        opts.DBPath,
		Limit:         25,
		Keyword:       KeywordForAssignment(assignment),
		IncludeFeeds:  true,
		IncludeGrants: true,
	})
	return err
}

// KeywordForAssignment picks a Grants.gov search keyword from a Research
// Assignment. The keyword is what we feed to the Grants.gov API during
// refresh — it materially shapes what shows up in the ledger.
//
// Stage is the strongest signal. An academic lab is not an SBIR target even
// when the brief mentions "not SBIR/STTR" — a substring match on "sbir" in
// such a phrase would be a false positive. We branch on stage first.
func KeywordForAssignment(a Assignment) string {
	stage := strings.ToLower(a.CompanyProfile.Stage)
	if strings.Contains(stage, "academic") || strings.Contains(stage, "university") || strings.Contains(stage, "lab") {
		if len(a.FocusAreas) > 0 {
			return a.FocusAreas[0]
		}
		return "research"
	}
	text := strings.ToLower(BuildAssignmentQuery(a))
	switch {
	case strings.Contains(text, "sbir"), strings.Contains(text, "sttr"), strings.Contains(text, "startup"), strings.Contains(text, "small business"):
		return "SBIR"
	case strings.Contains(text, "climate"), strings.Contains(text, "clean energy"):
		return "clean energy"
	case len(a.FocusAreas) > 0:
		return a.FocusAreas[0]
	default:
		return "grant"
	}
}

func fitScore(level string) int {
	switch level {
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1
	}
}

func effortPenalty(level string) int {
	switch level {
	case "high":
		return 30
	case "medium":
		return 10
	default:
		return 0
	}
}

func evidenceScore(rec OpportunityRecord) int {
	score := len(rec.SourceRefs) * 5
	// Real grant listings carry an opportunity number or Federal Register
	// document number. These records exist on a canonical funding surface
	// (Grants.gov, the Federal Register). Weight them so they dominate
	// fit-level differences against text-matched ecosystem/media noise.
	if rec.OpportunityNumber != "" || rec.DocumentNumber != "" {
		score += 100
	}
	switch rec.Canonicality {
	case "authoritative":
		score += 60
	case "authoritative_or_corroborating", "state_authoritative", "authoritative_replacement":
		score += 40
	case "corroborating", "enrichment", "curated_lead", "cross_agency_index", "state_program":
		score += 20
	case "deadline_signal":
		score += 10
	case "early_warning", "human_qa_alert":
		score -= 20
	case "context", "ecosystem_media", "community_lead", "search_generated_lead", "commercial_lead":
		// Not opportunity records — these are leads, context, or news. Penalize
		// so they only surface when nothing more authoritative matches.
		score -= 100
	}
	return score
}

func deadlineSortKey(deadline *string) string {
	if deadline == nil || *deadline == "" {
		return "9999-99-99"
	}
	return *deadline
}

func fallback(v, fallbackValue string) string {
	if strings.TrimSpace(v) != "" {
		return v
	}
	return fallbackValue
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[strings.ToLower(value)] {
			seen[strings.ToLower(value)] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
