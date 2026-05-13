package grantfinder

import (
	"context"
	"regexp"
	"strings"
	"time"
)

var federalOpportunityNumberRE = regexp.MustCompile(`(?i)(?:funding opportunity ID|funding opportunity number|opportunity ID)\s+([A-Z0-9][A-Z0-9_.-]+)`)

func HydrateFederalRegister(ctx context.Context, doc string) (map[string]any, error) {
	doc = strings.TrimSpace(doc)
	if strings.HasPrefix(doc, "http://") || strings.HasPrefix(doc, "https://") {
		return FetchJSON(ctx, doc, 20*time.Second)
	}
	return FetchJSON(ctx, "https://www.federalregister.gov/api/v1/documents/"+doc+".json", 20*time.Second)
}

func OpportunityFromFederalRegister(base Opportunity, data map[string]any) Opportunity {
	out := base
	title := stringish(data["title"])
	if title != "" {
		out.Title = title
	}
	if htmlURL := stringish(data["html_url"]); htmlURL != "" {
		out.URL = htmlURL
	}
	if dates := stringish(data["dates"]); dates != "" {
		out.DeadlineText = dates
	}
	if published := stringish(data["publication_date"]); published != "" {
		out.Published = published
	}
	docNumber := stringish(data["document_number"])
	if docNumber != "" {
		out.DocumentNumber = docNumber
	}
	abstract := stringish(data["abstract"])
	if abstract != "" {
		out.Summary = abstract
		if out.OpportunityNumber == "" {
			if match := federalOpportunityNumberRE.FindStringSubmatch(abstract); len(match) == 2 {
				out.OpportunityNumber = strings.TrimRight(match[1], ".;,")
			}
		}
	}
	if agencies, ok := data["agencies"].([]any); ok {
		var names []string
		for _, rawAgency := range agencies {
			agency, _ := rawAgency.(map[string]any)
			if name := stringish(agency["name"]); name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			out.Sponsor = strings.Join(names, "; ")
		}
	}
	titleLower := strings.ToLower(out.Title)
	switch {
	case strings.Contains(titleLower, "notice of funding opportunity"), strings.Contains(titleLower, "funding opportunity"):
		out.RecordType = "grant"
	case strings.Contains(titleLower, "prize"), strings.Contains(titleLower, "challenge"):
		out.RecordType = "challenge"
	}
	out.Canonicality = "authoritative_or_corroborating"
	out.PublicationBasis = "federal_register"
	out.RawSignals = SortedSignals(append(out.RawSignals, "federal_register", out.RecordType))
	return out
}
