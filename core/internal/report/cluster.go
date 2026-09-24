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
	// voices son las fuentes independientes (ver voiceKey): cuatro medios
	// que publican el mismo cable de Reuters son una sola voz.
	voices map[string]bool
	// voiceCountries son los países de las voces que no son cables.
	voiceCountries map[string]bool
	// wires cuenta cuántos medios del cluster publicaron cada agencia.
	wires map[string]int
}

func (c *storyCluster) sourceCount() int      { return len(c.sourceIDs) }
func (c *storyCluster) countryCount() int     { return len(c.countries) }
func (c *storyCluster) independentCount() int { return len(c.voices) }

// weight favorece las historias con más fuentes independientes y duplica
// el crédito por cada país distinto entre ellas. Cuenta voces, no medios:
// la cantidad de medios que repiten un cable mide difusión, no
// corroboración.
func (c *storyCluster) weight() float64 {
	return float64(c.independentCount()) + float64(len(c.voiceCountries))*2
}

// wireSummary describe los cables del cluster: "Reuters en 3 medios".
func (c *storyCluster) wireSummary() string {
	var names []string
	for w := range c.wires {
		names = append(names, w)
	}
	sort.Strings(names)
	var parts []string
	for _, w := range names {
		parts = append(parts, fmt.Sprintf("%s en %d medios", w, c.wires[w]))
	}
	return strings.Join(parts, ", ")
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
			cl = &storyCluster{sourceIDs: map[string]bool{}, countries: map[string]bool{}, voices: map[string]bool{}, voiceCountries: map[string]bool{}, wires: map[string]int{}}
			byKey[key] = cl
			order = append(order, key)
		}
		cl.items = append(cl.items, it)
		cl.sourceIDs[it.Article.SourceID] = true
		cl.countries[it.Source.Country] = true
		cl.voices[voiceKey(it)] = true
		if w := detectWire(it); w != "" {
			cl.wires[w]++
		} else {
			cl.voiceCountries[it.Source.Country] = true
		}
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
