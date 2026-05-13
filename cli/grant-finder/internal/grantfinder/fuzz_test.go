package grantfinder

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzParseAssignment(f *testing.F) {
	f.Add([]byte(`{"assignment_id":"fuzz","company_profile":{"description":"robotics startup"},"focus_areas":["robotics"],"target_geographies":["United States"],"known_grants":[]}`))
	f.Add([]byte(`{"assignment_id":"fuzz","company_profile":{"description":"robotics startup"},"focus_areas":null,"target_geographies":null,"known_grants":null}`))
	f.Add([]byte(`{"assignment_id":"","company_profile":{"description":"missing id"}}`))
	f.Add([]byte(`not-json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64<<10 {
			return
		}
		assignment, err := ParseAssignment(data)
		if err != nil {
			return
		}
		if strings.TrimSpace(assignment.AssignmentID) == "" {
			t.Fatal("accepted assignment without assignment_id")
		}
		if strings.TrimSpace(assignment.CompanyProfile.Description) == "" {
			t.Fatal("accepted assignment without company_profile.description")
		}
		if assignment.FocusAreas == nil || assignment.TargetGeographies == nil || assignment.KnownGrants == nil {
			t.Fatal("accepted assignment did not normalize nil slices")
		}
		if _, err := json.Marshal(assignment); err != nil {
			t.Fatalf("accepted assignment is not JSON-marshalable: %v", err)
		}
		_ = BuildAssignmentQuery(assignment)
	})
}

func FuzzParseFeedItems(f *testing.F) {
	f.Add([]byte(`<rss><channel><item><title>SBIR robotics grant</title><link>https://example.test/grant</link><guid>raw-1</guid><description><![CDATA[<p>Funds robotics.</p>]]></description><pubDate>Mon, 01 May 2026 00:00:00 GMT</pubDate></item></channel></rss>`))
	f.Add([]byte(`<feed><entry><title>Challenge prize</title><id>entry-1</id><link rel="alternate" href="https://example.test/challenge"/><summary>Prize funding</summary><updated>2026-05-01</updated></entry></feed>`))
	f.Add([]byte(`<rss><channel><item><title></title><link></link></item></channel></rss>`))
	f.Add([]byte(`not-xml`))

	feed := Feed{
		ID:           "fuzz-feed",
		Name:         "Fuzz Feed",
		URL:          "https://example.test/feed.xml",
		Type:         "rss",
		Canonicality: "authoritative",
		Signals:      []string{"sbir", "grant", "grant"},
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 128<<10 {
			return
		}
		items, err := ParseFeedItems(feed, data)
		if err != nil {
			return
		}
		for _, item := range items {
			if item.SourceID != feed.ID {
				t.Fatalf("item source id = %q, want %q", item.SourceID, feed.ID)
			}
			if item.SourceURL != feed.URL {
				t.Fatalf("item source url = %q, want %q", item.SourceURL, feed.URL)
			}
			if strings.TrimSpace(item.Title) == "" && strings.TrimSpace(item.URL) == "" {
				t.Fatal("parsed item with both empty title and URL")
			}
			if !sort.StringsAreSorted(item.Signals) {
				t.Fatalf("signals are not sorted: %v", item.Signals)
			}
		}
	})
}

func FuzzParseGrantsXMLZip(f *testing.F) {
	f.Add("123", "Autonomous EV depot charging grant", "DE-FOA-0000002", "Department of Energy", "DOE", "D", "Grant", "Energy", "Supports robotics and EV infrastructure.", "05012026", "08/01/2026")
	f.Add("", "", "", "", "", "", "", "", "", "", "")
	f.Add("456", "Arts education grant", "NEA-1", "Example Agency", "NEA", "D", "Grant", "Arts", "Education.", "2026-04-01", "2026-08-01")

	f.Fuzz(func(t *testing.T, id, title, number, agency, agencyCode, status, instrument, category, description, postDate, closeDate string) {
		body := grantsXMLZip(t, map[string]string{
			"OpportunityID":             shortString(id, 64),
			"OpportunityTitle":          shortString(title, 256),
			"OpportunityNumber":         shortString(number, 128),
			"AgencyName":                shortString(agency, 256),
			"AgencyCode":                shortString(agencyCode, 64),
			"OpportunityCategory":       shortString(status, 32),
			"FundingInstrumentType":     shortString(instrument, 64),
			"CategoryOfFundingActivity": shortString(category, 128),
			"Description":               shortString(description, 512),
			"PostDate":                  shortString(postDate, 64),
			"CloseDate":                 shortString(closeDate, 64),
		})
		records, err := ParseGrantsXMLZip(body, nil, 5, 10)
		if err != nil {
			t.Fatalf("structure-aware ZIP should parse: %v", err)
		}
		if len(records) > 5 {
			t.Fatalf("got %d records, want <= 5", len(records))
		}
		for _, record := range records {
			if record.SourceID != "grants-gov-xml-extract" {
				t.Fatalf("source id = %q", record.SourceID)
			}
			if record.RecordType != "grant" {
				t.Fatalf("record type = %q", record.RecordType)
			}
			if record.Canonicality != "authoritative" {
				t.Fatalf("canonicality = %q", record.Canonicality)
			}
			if strings.TrimSpace(record.OpportunityID) == "" && record.URL != "" {
				t.Fatalf("record without opportunity ID has URL %q", record.URL)
			}
			if strings.TrimSpace(record.OpportunityID) != "" && !strings.HasSuffix(record.URL, record.OpportunityID) {
				t.Fatalf("record URL %q does not end with opportunity ID %q", record.URL, record.OpportunityID)
			}
		}
	})
}

func FuzzOpportunityFromFederalRegister(f *testing.F) {
	f.Add("Notice of Funding Opportunity for Clean Energy Infrastructure", "https://www.federalregister.gov/documents/2026/01/02/test-doc", "Applications due August 1, 2026.", "2026-01-02", "2026-00001", "Funding opportunity number DE-FOA-0000001 supports clean energy infrastructure.", "Department of Energy")
	f.Add("Prize Competition", "", "", "", "", "Challenge description.", "")
	f.Add("", "", "", "", "", "", "")

	f.Fuzz(func(t *testing.T, title, htmlURL, dates, publicationDate, documentNumber, abstract, agencyName string) {
		base := Opportunity{
			Title:      shortString(title, 256),
			RawSignals: []string{"seed", "seed"},
		}
		data := map[string]any{
			"title":            shortString(title, 256),
			"html_url":         shortString(htmlURL, 512),
			"dates":            shortString(dates, 512),
			"publication_date": shortString(publicationDate, 64),
			"document_number":  shortString(documentNumber, 64),
			"abstract":         shortString(abstract, 1024),
			"agencies": []any{
				map[string]any{"name": shortString(agencyName, 256)},
			},
		}
		out := OpportunityFromFederalRegister(base, data)
		if out.PublicationBasis != "federal_register" {
			t.Fatalf("publication basis = %q", out.PublicationBasis)
		}
		if out.Canonicality != "authoritative_or_corroborating" {
			t.Fatalf("canonicality = %q", out.Canonicality)
		}
		if !sort.StringsAreSorted(out.RawSignals) {
			t.Fatalf("raw signals are not sorted: %v", out.RawSignals)
		}
		if !containsString(out.RawSignals, "federal_register") {
			t.Fatalf("raw signals missing federal_register: %v", out.RawSignals)
		}
		if agencyName != "" && out.Sponsor == "" {
			t.Fatal("non-empty agency name did not hydrate sponsor")
		}
	})
}

func grantsXMLZip(t *testing.T, fields map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("GrantsDBExtract.xml")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprint(w, "<Grants><OpportunitySynopsisDetail_1_0>")
	for _, name := range []string{
		"OpportunityID",
		"OpportunityTitle",
		"OpportunityNumber",
		"AgencyName",
		"AgencyCode",
		"OpportunityCategory",
		"FundingInstrumentType",
		"CategoryOfFundingActivity",
		"Description",
		"PostDate",
		"CloseDate",
	} {
		fmt.Fprintf(w, "<%s>%s</%s>", name, xmlSafeText(fields[name]), name)
	}
	fmt.Fprint(w, "</OpportunitySynopsisDetail_1_0></Grants>")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func shortString(s string, limit int) string {
	if len(s) > limit {
		return s[:limit]
	}
	return s
}

func xmlSafeText(s string) string {
	var cleaned strings.Builder
	for _, r := range s {
		if r == utf8.RuneError {
			continue
		}
		if r == '\t' || r == '\n' || r == '\r' || r >= 0x20 {
			cleaned.WriteRune(r)
		}
	}
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(cleaned.String())); err != nil {
		return ""
	}
	return escaped.String()
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
