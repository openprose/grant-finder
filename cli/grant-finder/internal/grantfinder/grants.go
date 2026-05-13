package grantfinder

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	grantsSearchURL = "https://api.grants.gov/v1/api/search2"
	grantsDetailURL = "https://api.grants.gov/v1/api/fetchOpportunity"
	grantsXMLPage   = "https://grants.gov/xml-extract"
)

var grantsExtractRE = regexp.MustCompile(`https://[^"<>]+GrantsDBExtract\d+v2\.zip`)

var grantsXMLSearchFields = []func(grantsXMLDetail) string{
	func(d grantsXMLDetail) string { return d.OpportunityTitle },
	func(d grantsXMLDetail) string { return d.OpportunityNumber },
	func(d grantsXMLDetail) string { return d.AgencyName },
	func(d grantsXMLDetail) string { return d.AgencyCode },
	func(d grantsXMLDetail) string { return d.FundingInstrumentType },
	func(d grantsXMLDetail) string { return d.CategoryOfFundingActivity },
	func(d grantsXMLDetail) string { return d.Description },
}

type GrantsRecord struct {
	SourceID                 string   `json:"source_id"`
	PublicationBasis         string   `json:"publication_basis"`
	Canonicality             string   `json:"canonicality"`
	RecordType               string   `json:"record_type"`
	OpportunityID            string   `json:"opportunity_id,omitempty"`
	FundingOpportunityNumber string   `json:"funding_opportunity_number,omitempty"`
	OpportunityNumber        string   `json:"opportunity_number,omitempty"`
	Title                    string   `json:"title,omitempty"`
	AgencyCode               string   `json:"agency_code,omitempty"`
	Agency                   string   `json:"agency,omitempty"`
	Sponsor                  string   `json:"sponsor,omitempty"`
	Status                   string   `json:"status,omitempty"`
	PostDate                 string   `json:"post_date,omitempty"`
	CloseDate                string   `json:"close_date,omitempty"`
	DeadlineText             string   `json:"deadline_text,omitempty"`
	URL                      string   `json:"url,omitempty"`
	RawSignals               []string `json:"raw_signals"`
}

type grantsXMLDetail struct {
	OpportunityID             string `xml:"OpportunityID"`
	OpportunityTitle          string `xml:"OpportunityTitle"`
	OpportunityNumber         string `xml:"OpportunityNumber"`
	AgencyName                string `xml:"AgencyName"`
	AgencyCode                string `xml:"AgencyCode"`
	OpportunityCategory       string `xml:"OpportunityCategory"`
	FundingInstrumentType     string `xml:"FundingInstrumentType"`
	CategoryOfFundingActivity string `xml:"CategoryOfFundingActivity"`
	CFDANumbers               string `xml:"CFDANumbers"`
	Description               string `xml:"Description"`
	PostDate                  string `xml:"PostDate"`
	CloseDate                 string `xml:"CloseDate"`
}

// GrantsSearch queries the Grants.gov search2 API. By default it requests
// only currently-actionable opportunities (forecasted and posted) so the
// CLI's research surface stays focused on grants an agent could actually
// apply for. The maintainer-only debug surface (`debug grants search
// --status ...`) can pass an explicit status string to override this default
// when historical research is needed.
func GrantsSearch(ctx context.Context, keyword string, rows int, oppNum string) ([]GrantsRecord, error) {
	return GrantsSearchWithStatus(ctx, keyword, rows, oppNum, "forecasted|posted")
}

// GrantsSearchWithStatus is the explicit-status form. Pass an empty string to
// accept the Grants.gov default (all statuses).
func GrantsSearchWithStatus(ctx context.Context, keyword string, rows int, oppNum, oppStatuses string) ([]GrantsRecord, error) {
	if rows <= 0 {
		rows = 10
	}
	payload := map[string]any{
		"rows": rows,
	}
	if oppStatuses != "" {
		payload["oppStatuses"] = oppStatuses
	}
	if oppNum != "" {
		payload["oppNum"] = oppNum
	} else {
		payload["keyword"] = keyword
	}
	data, err := postJSON(ctx, grantsSearchURL, payload, 30*time.Second)
	if err != nil {
		return nil, err
	}
	d, _ := data["data"].(map[string]any)
	hits, _ := d["oppHits"].([]any)
	var out []GrantsRecord
	for _, raw := range hits {
		hit, _ := raw.(map[string]any)
		out = append(out, normalizeGrantHit(hit))
	}
	return out, nil
}

func GrantsFetch(ctx context.Context, opportunityID string) (map[string]any, error) {
	if opportunityID == "" {
		return nil, fmt.Errorf("opportunity id is required")
	}
	data, err := postJSON(ctx, grantsDetailURL, map[string]any{"opportunityId": opportunityID}, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if d, ok := data["data"].(map[string]any); ok {
		return d, nil
	}
	return data, nil
}

func LatestGrantsXMLExtract(ctx context.Context) (map[string]any, error) {
	body, status, contentType, err := getBytes(ctx, grantsXMLPage, 30*time.Second)
	if err != nil {
		return nil, err
	}
	latestZip := ""
	if match := grantsExtractRE.Find(body); len(match) > 0 {
		latestZip = string(match)
	}
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	return map[string]any{
		"page":         grantsXMLPage,
		"status":       status,
		"content_type": contentType,
		"latest_zip":   latestZip,
		"json":         parsed,
		"note":         "This verifies the canonical XML extract surface. Full XML row ingestion remains the next parser port from pipeline_poc.py.",
	}, nil
}

func GrantsXMLRecords(ctx context.Context, keywords []string, limit int, maxScan int) ([]GrantsRecord, error) {
	info, err := LatestGrantsXMLExtract(ctx)
	if err != nil {
		return nil, err
	}
	zipURL := stringish(info["latest_zip"])
	if zipURL == "" {
		return nil, fmt.Errorf("latest Grants.gov XML ZIP not found")
	}
	body, status, _, err := getBytesLimit(ctx, zipURL, 90*time.Second, 128<<20)
	if err != nil {
		return nil, err
	}
	if status < 200 || status > 299 {
		return nil, fmt.Errorf("fetching Grants.gov XML ZIP: HTTP %d", status)
	}
	return ParseGrantsXMLZip(body, keywords, limit, maxScan)
}

func ParseGrantsXMLZip(body []byte, keywords []string, limit int, maxScan int) ([]GrantsRecord, error) {
	if limit <= 0 {
		limit = 25
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	var xmlFile *zip.File
	for _, file := range zr.File {
		if strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
			xmlFile = file
			break
		}
	}
	if xmlFile == nil {
		return nil, fmt.Errorf("no XML file found in Grants.gov ZIP")
	}
	rc, err := xmlFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	records, err := parseGrantsXML(rc, keywords, limit, maxScan)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		return grantsDateKey(records[i]) > grantsDateKey(records[j])
	})
	if len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

func parseGrantsXML(r io.Reader, keywords []string, limit int, maxScan int) ([]GrantsRecord, error) {
	dec := xml.NewDecoder(r)
	var records []GrantsRecord
	scanned := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "OpportunitySynopsisDetail_1_0" {
			continue
		}
		var detail grantsXMLDetail
		if err := dec.DecodeElement(&detail, &start); err != nil {
			return nil, err
		}
		scanned++
		if !grantsXMLMatches(detail, keywords) {
			if maxScan > 0 && scanned >= maxScan {
				break
			}
			continue
		}
		records = append(records, grantsXMLDetailToRecord(detail))
		if maxScan > 0 && scanned >= maxScan {
			break
		}
	}
	return records, nil
}

func grantsXMLMatches(detail grantsXMLDetail, keywords []string) bool {
	var lowered []string
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" {
			lowered = append(lowered, keyword)
		}
	}
	if len(lowered) == 0 {
		return true
	}
	var haystack strings.Builder
	for _, field := range grantsXMLSearchFields {
		haystack.WriteString(" ")
		haystack.WriteString(strings.ToLower(field(detail)))
	}
	text := haystack.String()
	for _, keyword := range lowered {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func grantsXMLDetailToRecord(detail grantsXMLDetail) GrantsRecord {
	return GrantsRecord{
		SourceID:                 "grants-gov-xml-extract",
		PublicationBasis:         "official_bulk_xml",
		Canonicality:             "authoritative",
		RecordType:               "grant",
		OpportunityID:            strings.TrimSpace(detail.OpportunityID),
		FundingOpportunityNumber: strings.TrimSpace(detail.OpportunityNumber),
		OpportunityNumber:        strings.TrimSpace(detail.OpportunityNumber),
		Title:                    strings.TrimSpace(detail.OpportunityTitle),
		AgencyCode:               strings.TrimSpace(detail.AgencyCode),
		Agency:                   strings.TrimSpace(detail.AgencyName),
		Sponsor:                  strings.TrimSpace(detail.AgencyName),
		Status:                   strings.TrimSpace(detail.OpportunityCategory),
		PostDate:                 parseGrantsDate(detail.PostDate),
		CloseDate:                parseGrantsDate(detail.CloseDate),
		DeadlineText:             parseGrantsDate(detail.CloseDate),
		URL:                      grantsOpportunityURL(detail.OpportunityID),
		RawSignals:               []string{"grants.gov", "xml_extract", strings.TrimSpace(detail.OpportunityCategory), strings.TrimSpace(detail.AgencyCode)},
	}
}

func grantsOpportunityURL(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return "https://www.grants.gov/search-results-detail/" + id
}

func grantsDateKey(record GrantsRecord) string {
	if record.PostDate != "" {
		return record.PostDate
	}
	return record.DeadlineText
}

func parseGrantsDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, layout := range []string{"01022006", "01/02/2006", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}
	return value
}

func normalizeGrantHit(hit map[string]any) GrantsRecord {
	id := stringish(hit["id"])
	return GrantsRecord{
		SourceID:                 "grants-gov-api",
		PublicationBasis:         "official_api",
		Canonicality:             "authoritative",
		RecordType:               "grant",
		OpportunityID:            id,
		FundingOpportunityNumber: stringish(hit["number"]),
		OpportunityNumber:        stringish(hit["number"]),
		Title:                    stringish(hit["title"]),
		AgencyCode:               stringish(hit["agencyCode"]),
		Agency:                   stringish(hit["agency"]),
		Sponsor:                  stringish(hit["agency"]),
		Status:                   stringish(hit["oppStatus"]),
		PostDate:                 stringish(hit["openDate"]),
		CloseDate:                stringish(hit["closeDate"]),
		DeadlineText:             stringish(hit["closeDate"]),
		URL:                      "https://www.grants.gov/search-results-detail/" + id,
		RawSignals:               []string{"grants.gov", "official_api"},
	}
}

func OpportunityFromGrantsRecord(record GrantsRecord) Opportunity {
	return Opportunity{
		RecordType:        fallback(record.RecordType, "grant"),
		Title:             record.Title,
		Sponsor:           firstNonEmpty(record.Sponsor, record.Agency, record.AgencyCode),
		URL:               record.URL,
		DeadlineText:      firstNonEmpty(record.DeadlineText, record.CloseDate),
		Published:         record.PostDate,
		OpportunityNumber: firstNonEmpty(record.OpportunityNumber, record.FundingOpportunityNumber),
		Canonicality:      fallback(record.Canonicality, "authoritative"),
		PublicationBasis:  fallback(record.PublicationBasis, "official_api"),
		Summary:           strings.Join(nonEmptyStrings(record.Title, record.Agency, record.Status), " "),
		RawSignals:        SortedSignals(append(record.RawSignals, "grants.gov", record.Status, record.AgencyCode)),
		SourceRefs: []Ref{{
			SourceID:  record.SourceID,
			SourceURL: grantsSearchURL,
			RawID:     firstNonEmpty(record.OpportunityID, record.OpportunityNumber, record.FundingOpportunityNumber),
		}},
	}
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func stringish(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}
