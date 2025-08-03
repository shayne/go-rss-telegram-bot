package rssbot

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Items       []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Link    []AtomLink  `xml:"link"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
	Author      string `xml:"author"`
	Creator     string `xml:"dc:creator"`
}

type AtomEntry struct {
	Title   string     `xml:"title"`
	Link    []AtomLink `xml:"link"`
	Summary string     `xml:"summary"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
	ID      string     `xml:"id"`
	Author  struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

func (b *Bot) findAndParseFeed(ctx context.Context, urlStr string) (string, *FeedInfo, error) {
	urlsToTry := generateParentURLs(urlStr)

	for _, tryURL := range urlsToTry {
		feedURL, feedInfo, err := b.tryFindFeedAtURL(ctx, tryURL)
		if err == nil {
			return feedURL, feedInfo, nil
		}
	}

	return "", nil, fmt.Errorf("no valid RSS/Atom feed found")
}

func generateParentURLs(urlStr string) []string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return []string{urlStr}
	}

	var urls []string
	urls = append(urls, urlStr)

	path := parsedURL.Path
	for path != "" && path != "/" {
		lastSlash := strings.LastIndex(path, "/")
		if lastSlash <= 0 {
			break
		}
		path = path[:lastSlash]

		parentURL := *parsedURL
		parentURL.Path = path
		parentURL.RawQuery = ""
		parentURL.Fragment = ""
		urls = append(urls, parentURL.String())
	}

	if parsedURL.Path != "/" {
		rootURL := *parsedURL
		rootURL.Path = "/"
		rootURL.RawQuery = ""
		rootURL.Fragment = ""
		urls = append(urls, rootURL.String())
	}

	return urls
}
func (b *Bot) tryFindFeedAtURL(ctx context.Context, urlStr string) (string, *FeedInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("User-Agent", "RSS-Telegram-Bot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "xml") || strings.Contains(contentType, "rss") || strings.Contains(contentType, "atom") {
		feedInfo, err := parseFeedData(body)
		if err == nil {
			return urlStr, feedInfo, nil
		}
	}

	if strings.Contains(contentType, "html") {
		feedURLs := findFeedURLsInHTML(body, urlStr)
		for _, feedURL := range feedURLs {
			feedReq, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
			if err != nil {
				continue
			}
			feedReq.Header.Set("User-Agent", "RSS-Telegram-Bot/1.0")

			feedResp, err := client.Do(feedReq)
			if err != nil {
				continue
			}

			feedBody, err := io.ReadAll(feedResp.Body)
			feedResp.Body.Close()
			if err != nil {
				continue
			}

			if feedInfo, err := parseFeedData(feedBody); err == nil {
				return feedURL, feedInfo, nil
			}
		}
	}

	return "", nil, fmt.Errorf("no valid RSS/Atom feed found at %s", urlStr)
}

func parseFeedData(data []byte) (*FeedInfo, error) {
	var rssFeed RSSFeed
	if err := xml.Unmarshal(data, &rssFeed); err == nil {
		title := rssFeed.Channel.Title
		if title == "" {
			title = "Untitled Feed"
		}
		return &FeedInfo{
			Title:       title,
			Description: rssFeed.Channel.Description,
			Link:        rssFeed.Channel.Link,
		}, nil
	}

	var atomFeed AtomFeed
	if err := xml.Unmarshal(data, &atomFeed); err == nil {
		link := ""
		for _, l := range atomFeed.Link {
			if l.Rel == "alternate" || l.Rel == "" {
				link = l.Href
				break
			}
		}
		title := atomFeed.Title
		if title == "" {
			title = "Untitled Feed"
		}
		return &FeedInfo{
			Title: title,
			Link:  link,
		}, nil
	}

	return nil, fmt.Errorf("unable to parse feed")
}

func findFeedURLsInHTML(htmlData []byte, baseURL string) []string {
	doc, err := html.Parse(strings.NewReader(string(htmlData)))
	if err != nil {
		return nil
	}

	base, _ := url.Parse(baseURL)
	var feedURLs []string

	var findFeeds func(*html.Node)
	findFeeds = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href, feedType string
			for _, attr := range n.Attr {
				switch attr.Key {
				case "rel":
					rel = attr.Val
				case "href":
					href = attr.Val
				case "type":
					feedType = attr.Val
				}
			}

			if rel == "alternate" && (feedType == "application/rss+xml" || feedType == "application/atom+xml") && href != "" {
				feedURL, err := url.Parse(href)
				if err == nil {
					if !feedURL.IsAbs() {
						feedURL = base.ResolveReference(feedURL)
					}
					feedURLs = append(feedURLs, feedURL.String())
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findFeeds(c)
		}
	}

	findFeeds(doc)
	return feedURLs
}

func (b *Bot) fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, *AtomFeed, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "RSS-Telegram-Bot/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var rssFeed RSSFeed
	if err := xml.Unmarshal(body, &rssFeed); err == nil && (rssFeed.Channel.Title != "" || len(rssFeed.Channel.Items) > 0) {
		return &rssFeed, nil, nil
	}

	var atomFeed AtomFeed
	if err := xml.Unmarshal(body, &atomFeed); err == nil && len(atomFeed.Entries) > 0 {
		return nil, &atomFeed, nil
	}

	return nil, nil, fmt.Errorf("unable to parse feed")
}
