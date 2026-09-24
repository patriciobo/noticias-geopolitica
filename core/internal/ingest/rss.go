package ingest

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"noticias/core/internal/model"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// rss2Feed maps the common subset of RSS 2.0 used by news outlets.
type rss2Feed struct {
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

// atomFeed maps the common subset of Atom used by news outlets.
type atomFeed struct {
	Entries []struct {
		Title   string `xml:"title"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
		Links   []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
	} `xml:"entry"`
}

// rdfFeed maps RSS 1.0 (RDF), still used by some Japanese/older outlets.
// Unlike RSS 2.0, <item> elements are siblings of <channel>, not nested in it.
type rdfFeed struct {
	Items []struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		Date        string `xml:"http://purl.org/dc/elements/1.1/ date"`
	} `xml:"item"`
}

// decodeXML wraps xml.Unmarshal with charset support for feeds that declare
// a non-UTF-8 encoding (iso-8859-1/windows-1252 are common in older CMSes).
// No external dependency: Latin-1 maps 1:1 onto the first 256 Unicode code
// points, so transcoding is a direct byte->rune re-encode.
func decodeXML(body []byte, v any) error {
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charset) {
		case "iso-8859-1", "latin1", "windows-1252", "us-ascii", "utf-8":
			return &latin1Reader{r: bufio.NewReader(input)}, nil
		default:
			return nil, fmt.Errorf("unsupported charset: %s", charset)
		}
	}
	return dec.Decode(v)
}

// latin1Reader re-encodes a Latin-1/ASCII/UTF-8-superset byte stream as
// UTF-8. Safe for plain UTF-8 and US-ASCII input too since both are byte-
// compatible with Latin-1 in the 0-127 range and this only touches the
// CharsetReader path (never applied to declared utf-8 without a directive).
type latin1Reader struct {
	r *bufio.Reader
}

func (lr *latin1Reader) Read(p []byte) (int, error) {
	var n int
	for n < len(p)-utf8.UTFMax {
		b, err := lr.r.ReadByte()
		if err != nil {
			if n == 0 {
				return 0, err
			}
			return n, nil
		}
		n += utf8.EncodeRune(p[n:], rune(b))
	}
	return n, nil
}

var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC3339,
	"2006-01-02T15:04:05Z07:00",
	// Fecha sola, sin hora (Xinhua): sin esto quedaba en cero y una nota de
	// 2017 pasaba el filtro de antigüedad como "sin fecha".
	"2006-01-02",
	"2006-01-02 15:04:05",
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// FetchHeadlines pulls up to maxItems headlines from a source's RSS/Atom feed.
func FetchHeadlines(src model.Source, maxItems int) ([]model.Article, error) {
	if src.RSS == "" || src.RSS == "NO_RSS" {
		return nil, fmt.Errorf("source %s has no RSS feed configured", src.ID)
	}

	req, err := http.NewRequest(http.MethodGet, src.RSS, nil)
	if err != nil {
		return nil, err
	}
	// Varios medios bloquean por defecto cualquier User-Agent no-navegador
	// en sus feeds RSS públicos, aunque el contenido sea el mismo. No hay
	// bypass de autenticación/paywall involucrado: es contenido publicado
	// públicamente como RSS.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("source %s: unexpected status %d", src.ID, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB cap
	if err != nil {
		return nil, err
	}

	if articles := parseRSS2(body, src.ID, maxItems); len(articles) > 0 {
		return articles, nil
	}
	if articles := parseAtom(body, src.ID, maxItems); len(articles) > 0 {
		return articles, nil
	}
	if articles := parseRDF(body, src.ID, maxItems); len(articles) > 0 {
		return articles, nil
	}
	return nil, fmt.Errorf("source %s: feed did not parse as RSS2, Atom or RDF", src.ID)
}

func parseRSS2(body []byte, sourceID string, maxItems int) []model.Article {
	var feed rss2Feed
	if err := decodeXML(body, &feed); err != nil {
		return nil
	}
	items := feed.Channel.Items
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	articles := make([]model.Article, 0, len(items))
	for _, it := range items {
		articles = append(articles, model.Article{
			SourceID:    sourceID,
			Title:       cleanText(it.Title, maxTitleRunes),
			Link:        cleanLink(it.Link),
			Snippet:     cleanText(it.Description, maxSnippetRunes),
			PublishedAt: parseDate(it.PubDate),
		})
	}
	return articles
}

func parseAtom(body []byte, sourceID string, maxItems int) []model.Article {
	var feed atomFeed
	if err := decodeXML(body, &feed); err != nil {
		return nil
	}
	entries := feed.Entries
	if len(entries) > maxItems {
		entries = entries[:maxItems]
	}
	articles := make([]model.Article, 0, len(entries))
	for _, e := range entries {
		link := ""
		for _, l := range e.Links {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}
		snippet := e.Summary
		if snippet == "" {
			snippet = e.Content
		}
		published := e.Published
		if published == "" {
			published = e.Updated
		}
		articles = append(articles, model.Article{
			SourceID:    sourceID,
			Title:       cleanText(e.Title, maxTitleRunes),
			Link:        cleanLink(link),
			Snippet:     cleanText(snippet, maxSnippetRunes),
			PublishedAt: parseDate(published),
		})
	}
	return articles
}

func parseRDF(body []byte, sourceID string, maxItems int) []model.Article {
	var feed rdfFeed
	if err := decodeXML(body, &feed); err != nil {
		return nil
	}
	items := feed.Items
	if len(items) > maxItems {
		items = items[:maxItems]
	}
	articles := make([]model.Article, 0, len(items))
	for _, it := range items {
		articles = append(articles, model.Article{
			SourceID:    sourceID,
			Title:       cleanText(it.Title, maxTitleRunes),
			Link:        cleanLink(it.Link),
			Snippet:     cleanText(it.Description, maxSnippetRunes),
			PublishedAt: parseDate(it.Date),
		})
	}
	return articles
}
