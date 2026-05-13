package grantfinder

import (
	"context"
	"fmt"
	"time"
)

type SourceSmokeResult struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Category    string `json:"category"`
	URL         string `json:"url"`
	StatusCode  int    `json:"status_code,omitempty"`
	Status      int    `json:"status"`
	ContentType string `json:"content_type,omitempty"`
	OK          bool   `json:"ok"`
	Error       string `json:"error,omitempty"`
}

func SmokeSources(ctx context.Context, limit int, timeout time.Duration) ([]SourceSmokeResult, error) {
	sources, err := Sources()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(sources) {
		limit = len(sources)
	}
	var out []SourceSmokeResult
	for _, source := range sources[:limit] {
		res := SourceSmokeResult{ID: source.ID, Label: source.Label, Category: source.Category, URL: source.URL}
		_, code, contentType, err := getBytes(ctx, source.URL, timeout)
		res.StatusCode = code
		res.Status = code
		res.ContentType = contentType
		if err != nil {
			res.Error = err.Error()
		} else if code >= 200 && code <= 399 {
			res.OK = true
		} else {
			res.Error = fmt.Sprintf("HTTP %d", code)
		}
		out = append(out, res)
	}
	return out, nil
}

type SmokeSummaryResult struct {
	Checked int `json:"checked"`
	OK      int `json:"ok"`
	Failed  int `json:"failed"`
}

func SmokeSummary(results any) any {
	switch rs := results.(type) {
	case []SourceSmokeResult:
		s := SmokeSummaryResult{Checked: len(rs)}
		for _, r := range rs {
			if r.OK {
				s.OK++
			} else {
				s.Failed++
			}
		}
		return s
	case []FeedSmokeResult:
		s := SmokeSummaryResult{Checked: len(rs)}
		for _, r := range rs {
			if r.OK {
				s.OK++
			} else {
				s.Failed++
			}
		}
		return s
	default:
		return results
	}
}
