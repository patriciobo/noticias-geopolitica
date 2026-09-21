package report

import (
	"fmt"
	"sort"
	"strings"

	"noticias/core/internal/model"
)

// storyCluster groups classified articles that plausibly cover the same
// underlying event, so the synthesis prompt can tell the model which
// stories are cross-validated by multiple independent outlets.
type storyCluster struct {
	items     []model.ClassifiedArticle
	sourceIDs map[string]bool
	countries map[string]bool
}

func (c *storyCluster) sourceCount() int  { return len(c.sourceIDs) }
func (c *storyCluster) countryCount() int { return len(c.countries) }

// weight favors stories confirmed by more outlets, and doubles the credit
// for each distinct country among those outlets — coverage from portals in
// different countries is a stronger relevance signal than repeat coverage
// within the same country.
func (c *storyCluster) weight() float64 {
	return float64(c.sourceCount()) + float64(c.countryCount())*2
}

// storySignature approximates "same story" from the entities the
// classifier already extracted. A shared company plus relation type is a
// fairly specific signal (two headlines both about, say, Volkswagen +
// supply_chain are very likely the same event). Without a shared company it
// falls back to shared countries + relation type — noisier, but still a
// reasonable proxy. Returns "" when there isn't enough signal to cluster on
// at all, so unrelated single-mention articles aren't force-merged.
func storySignature(c model.Classification) string {
	if companies := normalizeSet(c.Companies); len(companies) > 0 {
		return "company:" + strings.Join(companies, "+") + "|" + c.RelationType
	}
	if countries := normalizeSet(c.Countries); len(countries) > 0 {
		return "countries:" + strings.Join(countries, "+") + "|" + c.RelationType
	}
	return ""
}

func normalizeSet(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		n := strings.ToLower(strings.TrimSpace(it))
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// clusterStories groups items by storySignature and sorts clusters by
// weight, highest coverage first.
func clusterStories(items []model.ClassifiedArticle) []*storyCluster {
	byKey := map[string]*storyCluster{}
	var order []string

	for i, it := range items {
		key := storySignature(it.Classification)
		if key == "" {
			key = fmt.Sprintf("singleton:%d", i)
		}
		cl, ok := byKey[key]
		if !ok {
			cl = &storyCluster{sourceIDs: map[string]bool{}, countries: map[string]bool{}}
			byKey[key] = cl
			order = append(order, key)
		}
		cl.items = append(cl.items, it)
		cl.sourceIDs[it.Article.SourceID] = true
		cl.countries[it.Source.Country] = true
	}

	clusters := make([]*storyCluster, 0, len(order))
	for _, key := range order {
		clusters = append(clusters, byKey[key])
	}
	sort.SliceStable(clusters, func(i, j int) bool {
		return clusters[i].weight() > clusters[j].weight()
	})
	return clusters
}
