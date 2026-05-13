package grantfinder

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/sources.json data/feeds.json data/grant-finder-feeds.opml
var embedded embed.FS

type Source struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
	URL      string `json:"url"`
	Surface  string `json:"surface"`
}

type Feed struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	Type         string   `json:"type"`
	Access       string   `json:"access"`
	Canonicality string   `json:"canonicality"`
	Signals      []string `json:"signals"`
	Poll         string   `json:"poll"`
	NextSmoke    string   `json:"next_smoke"`
}

func Sources() ([]Source, error) {
	data, err := embedded.ReadFile("data/sources.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded sources: %w", err)
	}
	var sources []Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, fmt.Errorf("parsing sources: %w", err)
	}
	return sources, nil
}

func LoadSources() ([]Source, error) {
	return Sources()
}

func Feeds() ([]Feed, error) {
	data, err := embedded.ReadFile("data/feeds.json")
	if err != nil {
		return nil, fmt.Errorf("reading embedded feeds: %w", err)
	}
	var feeds []Feed
	if err := json.Unmarshal(data, &feeds); err != nil {
		return nil, fmt.Errorf("parsing feeds: %w", err)
	}
	return feeds, nil
}

func LoadFeeds() ([]Feed, error) {
	return Feeds()
}

func OPML() []byte {
	data, err := embedded.ReadFile("data/grant-finder-feeds.opml")
	if err != nil {
		return nil
	}
	return append([]byte(nil), data...)
}

func FilterFeeds(feeds []Feed, query string) []Feed {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return feeds
	}
	var out []Feed
	for _, feed := range feeds {
		haystack := strings.ToLower(strings.Join([]string{
			feed.ID, feed.Name, feed.Type, feed.Access, feed.Canonicality,
			strings.Join(feed.Signals, " "), feed.URL,
		}, " "))
		if strings.Contains(haystack, query) {
			out = append(out, feed)
		}
	}
	return out
}

func FilterSources(sources []Source, query string) []Source {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return sources
	}
	var out []Source
	for _, source := range sources {
		haystack := strings.ToLower(strings.Join([]string{
			source.ID, source.Label, source.Category, source.Surface, source.URL,
		}, " "))
		if strings.Contains(haystack, query) {
			out = append(out, source)
		}
	}
	return out
}
