package grantfinder

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"regexp"
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

var (
	sourcePageTitleRE                  = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	sourcePageMetaDescriptionRE        = regexp.MustCompile(`(?is)<meta[^>]+(?:name|property)\s*=\s*["'](?:description|og:description)["'][^>]+content\s*=\s*["']([^"']*)["']`)
	sourcePageMetaDescriptionContentRE = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']*)["'][^>]+(?:name|property)\s*=\s*["'](?:description|og:description)["']`)
)

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
	data, code, contentType, err := getBytesForFeed(ctx, feed, timeout)
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

func getBytesForFeed(ctx context.Context, feed Feed, timeout time.Duration) ([]byte, int, string, error) {
	if supportsSourcePageFallback(feed) {
		return getBytesLimit(ctx, feed.URL, timeout, 6<<20)
	}
	return getBytes(ctx, feed.URL, timeout)
}

func ParseFeedItems(feed Feed, data []byte) ([]FeedItem, error) {
	var doc rssDoc
	if err := unmarshalFeedXML(data, &doc); err != nil {
		if supportsSourcePageFallback(feed) {
			return ParseSourcePageItem(feed, data), nil
		}
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
	if len(out) == 0 && supportsSourcePageFallback(feed) {
		return ParseSourcePageItem(feed, data), nil
	}
	return out, nil
}

func unmarshalFeedXML(data []byte, doc *rssDoc) error {
	if err := xml.Unmarshal(data, doc); err != nil {
		repaired, changed := escapeBareAmpersands(data)
		if !changed {
			return err
		}
		if repairedErr := xml.Unmarshal(repaired, doc); repairedErr == nil {
			return nil
		}
		return err
	}
	return nil
}

func escapeBareAmpersands(data []byte) ([]byte, bool) {
	out := make([]byte, 0, len(data))
	changed := false
	for i := 0; i < len(data); i++ {
		if data[i] == '&' && !startsKnownEntity(data[i:]) {
			out = append(out, '&', 'a', 'm', 'p', ';')
			changed = true
			continue
		}
		out = append(out, data[i])
	}
	return out, changed
}

func startsKnownEntity(data []byte) bool {
	for _, entity := range [][]byte{
		[]byte("&amp;"),
		[]byte("&lt;"),
		[]byte("&gt;"),
		[]byte("&quot;"),
		[]byte("&apos;"),
	} {
		if bytes.HasPrefix(data, entity) {
			return true
		}
	}
	if len(data) < 4 || data[0] != '&' || data[1] != '#' {
		return false
	}
	for i := 2; i < len(data) && i < 16; i++ {
		switch {
		case data[i] == ';':
			return i > 2
		case data[i] == 'x' || data[i] == 'X':
			if i != 2 {
				return false
			}
		case (data[i] >= '0' && data[i] <= '9') ||
			(data[i] >= 'a' && data[i] <= 'f') ||
			(data[i] >= 'A' && data[i] <= 'F'):
			continue
		default:
			return false
		}
	}
	return false
}

func supportsSourcePageFallback(feed Feed) bool {
	switch strings.ToLower(strings.TrimSpace(feed.Type)) {
	case "source_page", "topic_page", "html_page":
		return true
	default:
		return false
	}
}

func ParseSourcePageItem(feed Feed, data []byte) []FeedItem {
	title := strings.TrimSpace(feed.Name)
	if title == "" {
		title = sourcePageTitle(data)
	}
	if title == "" && strings.TrimSpace(feed.URL) == "" {
		return nil
	}
	summary := sourcePageSummary(data)
	if summary == "" || looksLikeSourcePageBoilerplate(summary) {
		summary = "Public source page for " + title + "."
	}
	return []FeedItem{{
		SourceID:  feed.ID,
		SourceURL: feed.URL,
		RawID:     feed.URL,
		Title:     title,
		URL:       feed.URL,
		Summary:   summary,
		Signals:   SortedSignals(feed.Signals),
	}}
}

func sourcePageTitle(data []byte) string {
	match := sourcePageTitleRE.FindSubmatch(data)
	if len(match) < 2 {
		return ""
	}
	return cleanText(html.UnescapeString(stripTags(string(match[1]))))
}

func sourcePageSummary(data []byte) string {
	for _, re := range []*regexp.Regexp{sourcePageMetaDescriptionRE, sourcePageMetaDescriptionContentRE} {
		match := re.FindSubmatch(data)
		if len(match) >= 2 {
			return truncateText(cleanText(html.UnescapeString(string(match[1]))), 800)
		}
	}
	return ""
}

func looksLikeSourcePageBoilerplate(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "skip to main content") ||
		strings.Contains(lower, "an official website of the united states government") ||
		strings.Contains(lower, "official websites use .gov")
}

func truncateText(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes]))
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
