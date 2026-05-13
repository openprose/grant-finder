package grantfinder

import (
	"archive/zip"
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpportunityFromFeedItemAvoidsHyphenPhraseAsOpportunityNumber(t *testing.T) {
	feed := Feed{ID: "darpa", Name: "DARPA opportunities RSS", Type: "rss", Canonicality: "authoritative", Signals: []string{"sbir"}}
	item := FeedItem{
		SourceID:  feed.ID,
		SourceURL: "https://example.test/feed.xml",
		RawID:     "1",
		Title:     "SBIR: Compact Wideband Tunable Filters",
		URL:       "https://example.test/opportunities",
		Summary:   "A power-efficient mixed-signal thin-film opportunity.",
	}
	op := OpportunityFromFeedItem(feed, item)
	if op.OpportunityNumber != "" {
		t.Fatalf("expected no guessed opportunity number, got %q", op.OpportunityNumber)
	}
	if got := DedupeKey(op); got == "" || got[:10] != "title-url:" {
		t.Fatalf("expected title-url dedupe key, got %q", got)
	}
}

func TestStoreSearchRoundTrip(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	runID, err := store.StartRun(ctx, map[string]any{"test": true})
	if err != nil {
		t.Fatal(err)
	}
	op := Opportunity{
		RecordType: "grant",
		Title:      "SBIR battery manufacturing grant",
		Sponsor:    "Example Agency",
		URL:        "https://example.test/grant",
		Summary:    "Non-dilutive funding for battery manufacturing.",
		RawSignals: []string{"sbir", "grant"},
	}
	if _, err := store.UpsertOpportunity(ctx, runID, "test-source", "raw-1", "https://example.test/feed", op, op); err != nil {
		t.Fatal(err)
	}
	results, err := store.Search(ctx, "battery", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Title != op.Title {
		t.Fatalf("unexpected title %q", results[0].Title)
	}
}

func TestResearchBuildsDeterministicPacket(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "research.sqlite")
	store, err := OpenStore(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	runID, err := store.StartRun(ctx, map[string]any{"test": true})
	if err != nil {
		t.Fatal(err)
	}
	op := Opportunity{
		RecordType:        "grant",
		Title:             "SBIR autonomous vehicle fleet charging infrastructure grant",
		Sponsor:           "U.S. National Science Foundation",
		URL:               "https://example.test/sbir-autonomy",
		DeadlineText:      "2026-08-01",
		OpportunityNumber: "NSF-SBIR-TEST-1",
		Canonicality:      "authoritative",
		Summary:           "Non-dilutive funding for autonomous vehicle robotics and EV charging infrastructure.",
		Eligibility:       "Small businesses developing advanced manufacturing and robotics technology may apply.",
		RawSignals:        []string{"sbir", "robotics", "ev infrastructure"},
	}
	if _, err := store.UpsertOpportunity(ctx, runID, "test-source", "raw-1", "https://example.test/feed", op, op); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	assignment := Assignment{
		AssignmentID:     "test-assignment",
		ResearchQuestion: "Find grants for autonomous fleet servicing",
		CompanyProfile: CompanyProfile{
			Description:  "Startup building autonomous vehicle fleet servicing infrastructure.",
			Technologies: []string{"autonomous vehicle", "robotics", "ev infrastructure"},
		},
		FocusAreas:        []string{"autonomous vehicles", "robotics", "ev infrastructure"},
		TargetGeographies: []string{"United States"},
		KnownGrants:       []KnownGrant{},
	}
	packet, err := Research(ctx, ResearchOptions{DBPath: dbPath, Refresh: "off", Semantic: "off", Limit: 5, Now: time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)}, assignment)
	if err != nil {
		t.Fatal(err)
	}
	if !packet.Retrieval.NoLLM {
		t.Fatal("expected packet to record no_llm=true")
	}
	if len(packet.Grants) != 1 {
		t.Fatalf("expected one recommendation, got %d", len(packet.Grants))
	}
	if packet.Grants[0].EligibilityFit.Level != "high" {
		t.Fatalf("expected high fit, got %s", packet.Grants[0].EligibilityFit.Level)
	}
	if len(packet.Grants[0].Evidence) == 0 {
		t.Fatal("expected evidence")
	}
}

func TestResearchFiltersInactiveOpportunitiesByDefault(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "activity.sqlite")
	store, err := OpenStore(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	runID, err := store.StartRun(ctx, map[string]any{"test": true})
	if err != nil {
		t.Fatal(err)
	}
	active := Opportunity{
		RecordType:        "grant",
		Title:             "Foundational Research in Robotics",
		Sponsor:           "U.S. National Science Foundation",
		URL:               "https://example.test/nsf-frr",
		DeadlineText:      "2026-09-30",
		Published:         "2025-06-18",
		OpportunityNumber: "PD-20-144Y",
		Canonicality:      "authoritative",
		Summary:           "Foundational robotics research program for autonomous systems and fleet infrastructure.",
		RawSignals:        []string{"grants.gov", "posted", "robotics"},
	}
	archived := Opportunity{
		RecordType:        "grant",
		Title:             "Archived autonomous EV robotics grant",
		Sponsor:           "Department of Energy",
		URL:               "https://example.test/archived",
		DeadlineText:      "2024-01-01",
		Published:         "2023-01-01",
		OpportunityNumber: "DE-FOA-ARCHIVED",
		Canonicality:      "authoritative",
		Summary:           "Archived autonomous vehicle robotics and EV infrastructure funding.",
		RawSignals:        []string{"grants.gov", "archived", "robotics", "ev infrastructure"},
	}
	pastDue := Opportunity{
		RecordType:        "grant",
		Title:             "Past-due autonomous fleet charging robotics grant",
		Sponsor:           "Department of Transportation",
		URL:               "https://example.test/past-due",
		DeadlineText:      "2024-11-26",
		Published:         "2024-01-15",
		OpportunityNumber: "DOT-PAST-DUE",
		Canonicality:      "authoritative",
		Summary:           "Posted autonomous vehicle fleet charging and robotics opportunity.",
		RawSignals:        []string{"grants.gov", "posted", "robotics", "ev infrastructure"},
	}
	closedLoopActive := Opportunity{
		RecordType:        "grant",
		Title:             "Closed-loop battery robotics deployment grant",
		Sponsor:           "Example Energy Agency",
		URL:               "https://example.test/closed-loop-active",
		DeadlineText:      "2026-10-01",
		Published:         "2026-03-01",
		OpportunityNumber: "ENERGY-CLOSED-LOOP",
		Canonicality:      "authoritative",
		Summary:           "Posted opportunity for closed-loop battery logistics and robotics.",
		RawSignals:        []string{"grants.gov", "posted", "robotics", "ev infrastructure"},
	}
	for i, op := range []Opportunity{active, archived, pastDue, closedLoopActive} {
		if _, err := store.UpsertOpportunity(ctx, runID, "test-source", op.OpportunityNumber, "https://example.test/feed", op, op); err != nil {
			t.Fatalf("upsert %d: %v", i, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	assignment := Assignment{
		AssignmentID:     "activity-assignment",
		ResearchQuestion: "Find robotics and EV infrastructure grants",
		CompanyProfile: CompanyProfile{
			Description:  "Startup building autonomous fleet charging and robotics infrastructure.",
			Technologies: []string{"robotics", "ev infrastructure", "autonomous vehicle"},
		},
		FocusAreas:        []string{"robotics", "ev infrastructure"},
		TargetGeographies: []string{"United States"},
	}
	now := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	packet, err := Research(ctx, ResearchOptions{DBPath: dbPath, Refresh: "off", Semantic: "off", Limit: 10, Now: now}, assignment)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]GrantRecommendation{}
	for _, rec := range packet.Grants {
		got[rec.ProgramName] = rec
		if rec.ActivityStatus.Level != "active" {
			t.Fatalf("default research returned inactive recommendation %q: %+v", rec.ProgramName, rec.ActivityStatus)
		}
	}
	if _, ok := got[active.Title]; !ok {
		t.Fatalf("expected active opportunity %q in recommendations; got %v", active.Title, keysOfRecommendations(got))
	}
	if _, ok := got[closedLoopActive.Title]; !ok {
		t.Fatalf("expected closed-loop active opportunity not to be filtered by generic prose; got %v", keysOfRecommendations(got))
	}
	if _, ok := got[archived.Title]; ok {
		t.Fatalf("archived opportunity should be filtered by default")
	}
	if _, ok := got[pastDue.Title]; ok {
		t.Fatalf("past-due opportunity should be filtered by default")
	}

	withInactive, err := Research(ctx, ResearchOptions{DBPath: dbPath, Refresh: "off", Semantic: "off", Limit: 10, IncludeInactive: true, Now: now}, assignment)
	if err != nil {
		t.Fatal(err)
	}
	inactive := map[string]GrantRecommendation{}
	for _, rec := range withInactive.Grants {
		inactive[rec.ProgramName] = rec
	}
	if rec, ok := inactive[archived.Title]; !ok || rec.ActivityStatus.Level != "inactive" {
		t.Fatalf("expected archived historical comp with inactive status, got %+v", rec)
	}
	if rec, ok := inactive[pastDue.Title]; !ok || rec.ActivityStatus.Level != "inactive" {
		t.Fatalf("expected past-due historical comp with inactive status, got %+v", rec)
	}
}

func TestAssessActivityDateParsingAndRollingWindows(t *testing.T) {
	now := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		rec  OpportunityRecord
		want string
	}{
		{
			name: "future normalized date",
			rec:  OpportunityRecord{Opportunity: Opportunity{DeadlineText: "2026-09-30"}},
			want: "active",
		},
		{
			name: "past US date",
			rec:  OpportunityRecord{Opportunity: Opportunity{DeadlineText: "11/26/2024"}},
			want: "inactive",
		},
		{
			name: "rolling accepted anytime",
			rec:  OpportunityRecord{Opportunity: Opportunity{DeadlineText: "Applications accepted anytime"}},
			want: "active",
		},
		{
			name: "stale publication with no active signal",
			rec:  OpportunityRecord{Opportunity: Opportunity{Published: "2020-01-01"}},
			want: "inactive",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := AssessActivity(tt.rec, now)
			if got.Level != tt.want {
				t.Fatalf("expected %s, got %+v", tt.want, got)
			}
		})
	}
}

func keysOfRecommendations(values map[string]GrantRecommendation) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestCandidateRecordsCanUseUsearchMode(t *testing.T) {
	if _, err := exec.LookPath("usearch"); err != nil {
		t.Skip("usearch not installed")
	}
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "semantic.sqlite")
	store, err := OpenStore(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	runID, err := store.StartRun(ctx, map[string]any{"test": true})
	if err != nil {
		t.Fatal(err)
	}
	op := Opportunity{
		RecordType: "grant",
		Title:      "Autonomous fleet EV charging robotics grant",
		Sponsor:    "Example Agency",
		URL:        "https://example.test/semantic",
		Summary:    "Funds robotics for fleet servicing and electric charging depots.",
		RawSignals: []string{"robotics", "ev infrastructure"},
	}
	if _, err := store.UpsertOpportunity(ctx, runID, "semantic-source", "raw-1", "https://example.test/feed", op, op); err != nil {
		t.Fatal(err)
	}
	records, backend, err := CandidateRecords(ctx, store, dbPath, "autonomous vehicle depot charging robots", "usearch", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("expected usearch-mode candidate record")
	}
	if !strings.Contains(backend, "usearch") {
		t.Fatalf("expected usearch backend or fallback, got %q", backend)
	}
}

func TestOpportunityFromFederalRegisterHydratesCanonicalFields(t *testing.T) {
	base := Opportunity{
		RecordType:       "opportunity",
		Title:            "Weak feed title",
		URL:              "https://www.federalregister.gov/documents/2026/01/02/test-doc",
		DocumentNumber:   "2026-00001",
		PublicationBasis: "rss",
		RawSignals:       []string{"grant"},
	}
	hydrated := OpportunityFromFederalRegister(base, map[string]any{
		"title":            "Notice of Funding Opportunity for Clean Energy Infrastructure",
		"html_url":         "https://www.federalregister.gov/documents/2026/01/02/2026-00001/test-doc",
		"dates":            "Applications due August 1, 2026.",
		"publication_date": "2026-01-02",
		"document_number":  "2026-00001",
		"abstract":         "Funding opportunity number DE-FOA-0000001 supports clean energy infrastructure.",
		"agencies": []any{
			map[string]any{"name": "Department of Energy"},
		},
	})
	if hydrated.RecordType != "grant" {
		t.Fatalf("expected grant record type, got %q", hydrated.RecordType)
	}
	if hydrated.Sponsor != "Department of Energy" {
		t.Fatalf("expected hydrated sponsor, got %q", hydrated.Sponsor)
	}
	if hydrated.OpportunityNumber != "DE-FOA-0000001" {
		t.Fatalf("expected extracted opportunity number, got %q", hydrated.OpportunityNumber)
	}
	if hydrated.PublicationBasis != "federal_register" {
		t.Fatalf("expected federal_register basis, got %q", hydrated.PublicationBasis)
	}
}

func TestParseGrantsXMLZipFiltersAndNormalizes(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("GrantsDBExtract.xml")
	if err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<Grants>
  <OpportunitySynopsisDetail_1_0>
    <OpportunityID>123</OpportunityID>
    <OpportunityTitle>Autonomous EV depot charging grant</OpportunityTitle>
    <OpportunityNumber>DE-FOA-0000002</OpportunityNumber>
    <AgencyName>Department of Energy</AgencyName>
    <AgencyCode>DOE</AgencyCode>
    <OpportunityCategory>D</OpportunityCategory>
    <FundingInstrumentType>Grant</FundingInstrumentType>
    <CategoryOfFundingActivity>Energy</CategoryOfFundingActivity>
    <Description>Supports robotics and EV infrastructure.</Description>
    <PostDate>05012026</PostDate>
    <CloseDate>08/01/2026</CloseDate>
  </OpportunitySynopsisDetail_1_0>
  <OpportunitySynopsisDetail_1_0>
    <OpportunityID>456</OpportunityID>
    <OpportunityTitle>Arts education grant</OpportunityTitle>
    <AgencyName>Example Agency</AgencyName>
    <PostDate>2026-04-01</PostDate>
  </OpportunitySynopsisDetail_1_0>
</Grants>`
	if _, err := w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	records, err := ParseGrantsXMLZip(buf.Bytes(), []string{"EV infrastructure"}, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one matching XML record, got %d", len(records))
	}
	if records[0].OpportunityNumber != "DE-FOA-0000002" {
		t.Fatalf("unexpected opportunity number %q", records[0].OpportunityNumber)
	}
	if records[0].PostDate != "2026-05-01" || records[0].DeadlineText != "2026-08-01" {
		t.Fatalf("dates not normalized: post=%q deadline=%q", records[0].PostDate, records[0].DeadlineText)
	}
}

func TestEmbeddedManifestsLoad(t *testing.T) {
	sources, err := Sources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("expected embedded sources")
	}
	feeds, err := Feeds()
	if err != nil {
		t.Fatal(err)
	}
	if len(feeds) == 0 {
		t.Fatal("expected embedded feeds")
	}
	if len(OPML()) == 0 {
		t.Fatal("expected embedded OPML")
	}
}
