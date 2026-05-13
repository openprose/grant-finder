package grantfinder

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"strings"
	"time"
)

type FeedItem struct {
	SourceID  string   `json:"source_id"`
	SourceURL string   `json:"source_url"`
	RawID     string   `json:"raw_id"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	Summary   string   `json:"summary,omitempty"`
	Published string   `json:"published,omitempty"`
	Signals   []string `json:"signals,omitempty"`
}

type FeedSmokeResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	StatusCode  int    `json:"status_code,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	OK          bool   `json:"ok"`
	ItemCount   int    `json:"item_count"`
	Status      int    `json:"status"`
	Items       int    `json:"items"`
	Error       string `json:"error,omitempty"`
}

type rssDoc struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
	Entries []atomEntry `xml:"entry"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	ID      string     `xml:"id"`
	Links   []atomLink `xml:"link"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func SmokeFeeds(ctx context.Context, limit int, timeout time.Duration) ([]FeedSmokeResult, error) {
	feeds, err := Feeds()
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(feeds) {
		limit = len(feeds)
	}
	var results []FeedSmokeResult
	for _, feed := range feeds[:limit] {
		res, _ := smokeOneFeed(ctx, feed, timeout)
		results = append(results, res)
	}
	return results, nil
}

func smokeOneFeed(ctx context.Context, feed Feed, timeout time.Duration) (FeedSmokeResult, []FeedItem) {
	res := FeedSmokeResult{ID: feed.ID, Name: feed.Name, URL: feed.URL}
	data, code, contentType, err := getBytes(ctx, feed.URL, timeout)
	res.StatusCode = code
	res.Status = code
	res.ContentType = contentType
	if err != nil {
		res.Error = err.Error()
		return res, nil
	}
	if code < 200 || code > 299 {
		res.Error = fmt.Sprintf("HTTP %d", code)
		return res, nil
	}
	items, err := ParseFeedItems(feed, data)
	if err != nil {
		res.Error = err.Error()
		return res, nil
	}
	res.OK = len(items) > 0
	res.ItemCount = len(items)
	res.Items = len(items)
	return res, items
}

func ParseFeedItems(feed Feed, data []byte) ([]FeedItem, error) {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var out []FeedItem
	for _, item := range doc.Channel.Items {
		title := cleanText(html.UnescapeString(item.Title))
		link := strings.TrimSpace(item.Link)
		rawID := strings.TrimSpace(item.GUID)
		if rawID == "" {
			rawID = link
		}
		if rawID == "" {
			rawID = title
		}
		if title == "" && link == "" {
			continue
		}
		out = append(out, FeedItem{
			SourceID:  feed.ID,
			SourceURL: feed.URL,
			RawID:     rawID,
			Title:     title,
			URL:       link,
			Summary:   cleanText(html.UnescapeString(stripTags(item.Description))),
			Published: cleanText(item.PubDate),
			Signals:   SortedSignals(feed.Signals),
		})
	}
	for _, entry := range doc.Entries {
		link := ""
		for _, candidate := range entry.Links {
			if candidate.Rel == "" || candidate.Rel == "alternate" {
				link = candidate.Href
				break
			}
		}
		summary := entry.Summary
		if summary == "" {
			summary = entry.Content
		}
		title := cleanText(html.UnescapeString(entry.Title))
		if title == "" && link == "" {
			continue
		}
		rawID := strings.TrimSpace(entry.ID)
		if rawID == "" {
			rawID = link
		}
		out = append(out, FeedItem{
			SourceID:  feed.ID,
			SourceURL: feed.URL,
			RawID:     rawID,
			Title:     title,
			URL:       link,
			Summary:   cleanText(html.UnescapeString(stripTags(summary))),
			Published: cleanText(entry.Updated),
			Signals:   SortedSignals(feed.Signals),
		})
	}
	return out, nil
}

func OpportunityFromFeedItem(feed Feed, item FeedItem) Opportunity {
	text := strings.Join([]string{item.Title, item.Summary, item.URL}, " ")
	opportunityNumber := ""
	if strings.Contains(strings.ToUpper(text), "DE-FOA-") {
		opportunityNumber = GuessOpportunityNumber(text)
	}
	recordType := "opportunity"
	for _, signal := range feed.Signals {
		switch {
		case strings.Contains(signal, "grant"):
			recordType = "grant"
		case strings.Contains(signal, "fellowship"):
			recordType = "fellowship"
		case strings.Contains(signal, "challenge") || strings.Contains(signal, "prize"):
			recordType = "challenge"
		}
	}
	return Opportunity{
		RecordType:        recordType,
		Title:             item.Title,
		Sponsor:           feed.Name,
		URL:               item.URL,
		Published:         item.Published,
		OpportunityNumber: opportunityNumber,
		DocumentNumber:    GuessFRDocumentNumber(item.URL),
		Canonicality:      feed.Canonicality,
		PublicationBasis:  feed.Type,
		Summary:           item.Summary,
		RawSignals:        SortedSignals(append(item.Signals, feed.Type, feed.Canonicality)),
		SourceRefs:        []Ref{{SourceID: item.SourceID, SourceURL: item.SourceURL, RawID: item.RawID}},
	}
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
