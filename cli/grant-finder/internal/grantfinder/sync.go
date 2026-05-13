package grantfinder

import (
	"context"
	"time"
)

type SyncOptions struct {
	DBPath        string `json:"db"`
	Limit         int    `json:"limit"`
	Timeout       int    `json:"timeout_seconds"`
	Keyword       string `json:"keyword,omitempty"`
	IncludeFeeds  bool   `json:"include_feeds"`
	IncludeGrants bool   `json:"include_grants"`
	IncludeXML    bool   `json:"include_xml"`
}

type SyncCounts struct {
	RawItems   int `json:"raw_items"`
	Normalized int `json:"normalized"`
	New        int `json:"new"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
	FTSRows    int `json:"fts_rows"`
	FeedErrors int `json:"feed_errors"`
}

type SyncReport struct {
	RunID         int64 `json:"run_id"`
	FeedsChecked  int   `json:"feeds_checked"`
	GrantsChecked int   `json:"grants_checked"`
	XMLChecked    int   `json:"xml_checked"`
	Items         int   `json:"items"`
	New           int   `json:"new"`
	Updated       int   `json:"updated"`
	Unchanged     int   `json:"unchanged"`
	Errors        int   `json:"errors"`
}

func Sync(ctx context.Context, opts SyncOptions) (SyncReport, error) {
	store, err := OpenStore(ctx, opts.DBPath)
	if err != nil {
		return SyncReport{}, err
	}
	defer store.Close()
	runID, err := store.StartRun(ctx, opts)
	if err != nil {
		return SyncReport{}, err
	}
	feeds, err := Feeds()
	if err != nil {
		return SyncReport{}, err
	}
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	if !opts.IncludeFeeds && !opts.IncludeGrants && !opts.IncludeXML {
		opts.IncludeFeeds = true
		opts.IncludeGrants = true
	}
	report := SyncReport{RunID: runID}
	if opts.IncludeFeeds {
		feedLimit := opts.Limit
		if feedLimit <= 0 || feedLimit > len(feeds) {
			feedLimit = len(feeds)
		}
		for _, feed := range feeds[:feedLimit] {
			result, items := smokeOneFeed(ctx, feed, timeout)
			report.FeedsChecked++
			if !result.OK && result.Error != "" {
				report.Errors++
				continue
			}
			for _, item := range items {
				opportunity := OpportunityFromFeedItem(feed, item)
				if opportunity.DocumentNumber != "" {
					if hydrated, err := HydrateFederalRegister(ctx, opportunity.DocumentNumber); err == nil {
						opportunity = OpportunityFromFederalRegister(opportunity, hydrated)
					} else {
						report.Errors++
					}
				}
				status, err := store.UpsertOpportunity(ctx, runID, item.SourceID, item.RawID, item.SourceURL, opportunity, item)
				if err != nil {
					report.Errors++
					continue
				}
				addSyncStatus(&report, status)
			}
		}
	}

	if opts.IncludeGrants {
		keyword := opts.Keyword
		if keyword == "" {
			keyword = "SBIR"
		}
		rows := opts.Limit
		if rows <= 0 {
			rows = 25
		}
		records, err := GrantsSearch(ctx, keyword, rows, "")
		report.GrantsChecked = len(records)
		if err != nil {
			report.Errors++
		} else {
			for _, record := range records {
				rawID := firstNonEmpty(record.OpportunityID, record.OpportunityNumber, record.FundingOpportunityNumber, record.Title)
				status, err := store.UpsertOpportunity(ctx, runID, record.SourceID, rawID, record.URL, OpportunityFromGrantsRecord(record), record)
				if err != nil {
					report.Errors++
					continue
				}
				addSyncStatus(&report, status)
			}
		}
	}

	if opts.IncludeXML {
		keyword := opts.Keyword
		if keyword == "" {
			keyword = "SBIR"
		}
		rows := opts.Limit
		if rows <= 0 {
			rows = 25
		}
		records, err := GrantsXMLRecords(ctx, []string{keyword}, rows, rows*1000)
		report.XMLChecked = len(records)
		if err != nil {
			report.Errors++
		} else {
			for _, record := range records {
				rawID := firstNonEmpty(record.OpportunityID, record.OpportunityNumber, record.FundingOpportunityNumber, record.Title)
				status, err := store.UpsertOpportunity(ctx, runID, record.SourceID, rawID, record.URL, OpportunityFromGrantsRecord(record), record)
				if err != nil {
					report.Errors++
					continue
				}
				addSyncStatus(&report, status)
			}
		}
	}
	_ = store.FinishRun(ctx, runID, report)
	return report, nil
}

func RunSync(ctx context.Context, opts SyncOptions) (SyncCounts, error) {
	if !opts.IncludeFeeds && !opts.IncludeGrants && !opts.IncludeXML {
		opts.IncludeFeeds = true
	}
	report, err := Sync(ctx, opts)
	if err != nil {
		return SyncCounts{}, err
	}
	store, err := OpenStore(ctx, opts.DBPath)
	if err != nil {
		return SyncCounts{}, err
	}
	defer store.Close()
	stats, _ := store.Stats(ctx)
	return SyncCounts{
		RawItems:   report.Items,
		Normalized: report.Items,
		New:        report.New,
		Updated:    report.Updated,
		Unchanged:  report.Unchanged,
		FTSRows:    int(stats.FTSRows),
		FeedErrors: report.Errors,
	}, nil
}

func addSyncStatus(report *SyncReport, status string) {
	report.Items++
	switch status {
	case "new":
		report.New++
	case "updated":
		report.Updated++
	default:
		report.Unchanged++
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
