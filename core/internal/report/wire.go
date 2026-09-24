package report

import (
	"regexp"

	"noticias/core/internal/model"
)

// wireAgencies reconoce, en el título o el copete, el crédito de una
// agencia de noticias. Varios medios que publican el mismo cable no son
// fuentes independientes: cuentan como una sola voz, la de la agencia.
var wireAgencies = []struct {
	name string
	re   *regexp.Regexp
}{
	{"Reuters", regexp.MustCompile(`\bReuters\b`)},
	{"AFP", regexp.MustCompile(`\bAFP\b|Agence France-Presse`)},
	{"AP", regexp.MustCompile(`\(AP\)|\bAP\s[—–-]\s|Associated Press`)},
	{"EFE", regexp.MustCompile(`\bEFE\b`)},
	{"Europa Press", regexp.MustCompile(`Europa Press`)},
	{"ANSA", regexp.MustCompile(`\bANSA\b`)},
	{"dpa", regexp.MustCompile(`\bdpa\b|\bDPA\b`)},
	{"Bloomberg", regexp.MustCompile(`\bBloomberg\b`)},
	{"Xinhua", regexp.MustCompile(`\bXinhua\b`)},
	{"TASS", regexp.MustCompile(`\bTASS\b`)},
	{"RIA Novosti", regexp.MustCompile(`RIA Novosti`)},
	{"Anadolu", regexp.MustCompile(`\bAnadolu\b`)},
}

// detectWire devuelve la agencia acreditada en la nota, o "" si no hay
// ninguna reconocible. Una nota de la propia agencia (Anadolu publicando lo
// suyo) no cuenta como cable ajeno.
func detectWire(it model.ClassifiedArticle) string {
	text := it.Article.Title + " " + it.Article.Snippet
	for _, w := range wireAgencies {
		if w.re.MatchString(text) && !w.re.MatchString(it.Source.Name) {
			return w.name
		}
	}
	return ""
}

// voiceKey identifica la "voz" detrás de una nota para contar fuentes
// independientes: la agencia si es un cable, el Estado si es un medio
// estatal (RT y Sputnik repiten la misma línea oficial), o el medio.
func voiceKey(it model.ClassifiedArticle) string {
	if w := detectWire(it); w != "" {
		return "cable:" + w
	}
	if it.Source.Ownership == "estatal" {
		return "estatal:" + it.Source.Country
	}
	return "medio:" + it.Article.SourceID
}
