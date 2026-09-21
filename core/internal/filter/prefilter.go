package filter

import (
	"strings"

	"noticias/core/internal/model"
)

// PrefilterResult is the cheap, non-LLM pass over one article.
type PrefilterResult struct {
	MatchedCountries []string
	MatchedCompanies []string
	MatchedKeywords  []string
	Passed           bool
}

// Prefilter flags an article as a plausible international/multinational
// candidate using substring matching against the gazetteer. It is
// intentionally permissive (favors recall) — the LLM stage does the
// precise call. An article passes when it mentions a tracked company,
// a trade/treaty keyword, or two or more countries (one country alone
// is usually just domestic news naming its own country).
func Prefilter(a model.Article, g *Gazetteer) PrefilterResult {
	text := strings.ToLower(a.Title + " " + a.Snippet)

	var res PrefilterResult
	for _, c := range g.Countries {
		if strings.Contains(text, strings.ToLower(c)) {
			res.MatchedCountries = append(res.MatchedCountries, c)
		}
	}
	for _, c := range g.Companies {
		if strings.Contains(text, strings.ToLower(c)) {
			res.MatchedCompanies = append(res.MatchedCompanies, c)
		}
	}
	for _, k := range g.Keywords {
		if strings.Contains(text, strings.ToLower(k)) {
			res.MatchedKeywords = append(res.MatchedKeywords, k)
		}
	}

	res.Passed = len(res.MatchedCompanies) > 0 ||
		len(res.MatchedKeywords) > 0 ||
		len(res.MatchedCountries) >= 2

	return res
}
