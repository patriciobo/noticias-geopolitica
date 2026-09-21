package model

import "time"

// Article is one headline pulled from a source's feed.
type Article struct {
	SourceID    string    `json:"source_id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Snippet     string    `json:"snippet"`
	PublishedAt time.Time `json:"published_at"`
}

// Classification is the output of the filter pipeline for one article.
type Classification struct {
	IsInternational bool     `json:"is_international"`
	Countries       []string `json:"countries"`
	Companies       []string `json:"companies"`
	RelationType    string   `json:"relation_type"` // treaty | trade | sanction | supply_chain | regulatory | none
	Confidence      float64  `json:"confidence"`
	Reason          string   `json:"reason"`
}

// ClassifiedArticle bundles an article with its classification.
type ClassifiedArticle struct {
	Article        Article        `json:"article"`
	Source         Source         `json:"source"`
	Classification Classification `json:"classification"`
}

// SourceSummary is the minimal per-outlet info the blog needs to show
// "medios consultados" — deliberately not the full Source (no RSS URL,
// stance, etc.), this is public-facing metadata.
type SourceSummary struct {
	Name    string `json:"name"`
	Country string `json:"country"`
}
