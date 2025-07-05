package rssbot

import (
	"testing"
)

func TestParseFeedData(t *testing.T) {
	tests := []struct {
		name     string
		feedData string
		wantErr  bool
		wantInfo *FeedInfo
	}{
		{
			name: "RSS feed",
			feedData: `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed</title>
    <link>https://example.com</link>
    <description>A test RSS feed</description>
    <item>
      <title>Test Item</title>
      <link>https://example.com/item1</link>
      <guid>item1</guid>
    </item>
  </channel>
</rss>`,
			wantErr: false,
			wantInfo: &FeedInfo{
				Title:       "Test RSS Feed",
				Link:        "https://example.com",
				Description: "A test RSS feed",
			},
		},
		{
			name: "Atom feed",
			feedData: `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Atom Feed</title>
  <link href="https://example.com" rel="alternate"/>
  <entry>
    <title>Test Entry</title>
    <link href="https://example.com/entry1"/>
    <id>entry1</id>
  </entry>
</feed>`,
			wantErr: false,
			wantInfo: &FeedInfo{
				Title: "Test Atom Feed",
				Link:  "https://example.com",
			},
		},
		{
			name: "Atom feed with empty title",
			feedData: `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title></title>
  <link href="https://example.com" rel="alternate"/>
  <entry>
    <title>Test Entry</title>
    <link href="https://example.com/entry1"/>
    <id>entry1</id>
  </entry>
</feed>`,
			wantErr: false,
			wantInfo: &FeedInfo{
				Title: "Untitled Feed",
				Link:  "https://example.com",
			},
		},
		{
			name:     "Invalid XML",
			feedData: `not xml`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseFeedData([]byte(tt.feedData))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFeedData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if info.Title != tt.wantInfo.Title {
					t.Errorf("Title = %v, want %v", info.Title, tt.wantInfo.Title)
				}
				if info.Link != tt.wantInfo.Link {
					t.Errorf("Link = %v, want %v", info.Link, tt.wantInfo.Link)
				}
				if info.Description != tt.wantInfo.Description {
					t.Errorf("Description = %v, want %v", info.Description, tt.wantInfo.Description)
				}
			}
		})
	}
}

func TestFindFeedURLsInHTML(t *testing.T) {
	html := `
<!DOCTYPE html>
<html>
<head>
  <link rel="alternate" type="application/rss+xml" href="/feed.xml" title="RSS Feed">
  <link rel="alternate" type="application/atom+xml" href="https://example.com/atom.xml" title="Atom Feed">
  <link rel="stylesheet" href="/style.css">
</head>
<body>Test</body>
</html>`

	urls := findFeedURLsInHTML([]byte(html), "https://example.com/page")
	if len(urls) != 2 {
		t.Errorf("Expected 2 feed URLs, got %d", len(urls))
	}

	expectedURLs := map[string]bool{
		"https://example.com/feed.xml": true,
		"https://example.com/atom.xml": true,
	}

	for _, url := range urls {
		if !expectedURLs[url] {
			t.Errorf("Unexpected feed URL: %s", url)
		}
	}
}

func TestExtractRSSItems(t *testing.T) {
	feed := &RSSFeed{
		Channel: struct {
			Title       string    `xml:"title"`
			Link        string    `xml:"link"`
			Description string    `xml:"description"`
			Items       []RSSItem `xml:"item"`
		}{
			Items: []RSSItem{
				{
					Title:  "Item 1",
					Link:   "https://example.com/1",
					GUID:   "guid1",
					Author: "Author 1",
				},
				{
					Title:   "Item 2",
					Link:    "https://example.com/2",
					GUID:    "guid2",
					Creator: "Creator 2",
				},
			},
		},
	}

	bot := &Bot{}
	items := bot.extractRSSItems(feed)

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	if items[0].Author != "Author 1" {
		t.Errorf("Expected author 'Author 1', got '%s'", items[0].Author)
	}

	if items[1].Author != "Creator 2" {
		t.Errorf("Expected author 'Creator 2', got '%s'", items[1].Author)
	}
}

func TestExtractAtomItems(t *testing.T) {
	feed := &AtomFeed{
		Entries: []AtomEntry{
			{
				Title: "Entry 1",
				Link: []AtomLink{
					{Href: "https://example.com/1", Rel: "alternate"},
				},
				ID:      "id1",
				Summary: "Summary 1",
				Author: struct {
					Name string `xml:"name"`
				}{Name: "Author 1"},
			},
			{
				Title: "Entry 2",
				Link: []AtomLink{
					{Href: "https://example.com/2"},
				},
				ID:      "id2",
				Content: "Content 2",
			},
		},
	}

	bot := &Bot{}
	items := bot.extractAtomItems(feed)

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	if items[0].Link != "https://example.com/1" {
		t.Errorf("Expected link 'https://example.com/1', got '%s'", items[0].Link)
	}

	if items[0].Description != "Summary 1" {
		t.Errorf("Expected description 'Summary 1', got '%s'", items[0].Description)
	}

	if items[1].Description != "Content 2" {
		t.Errorf("Expected description 'Content 2', got '%s'", items[1].Description)
	}
}