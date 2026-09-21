package ingest

import "testing"

const sampleRSS2 = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Medio de Prueba</title>
<item>
  <title>Primer titular</title>
  <link>https://example.com/1</link>
  <description>Bajada del primer titular</description>
  <pubDate>Mon, 02 Jan 2006 15:04:05 -0700</pubDate>
</item>
<item>
  <title>Segundo titular</title>
  <link>https://example.com/2</link>
  <description>Bajada del segundo titular</description>
  <pubDate>Tue, 03 Jan 2006 10:00:00 -0700</pubDate>
</item>
</channel></rss>`

const sampleRDF = `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/" xmlns:dc="http://purl.org/dc/elements/1.1/">
<channel rdf:about="https://example.com/"><title>Medio RDF de Prueba</title></channel>
<item>
  <title>Titular RDF</title>
  <link>https://example.com/rdf-1</link>
  <description>Bajada del titular RDF</description>
  <dc:date>2006-01-02T15:04:05+09:00</dc:date>
</item>
</rdf:RDF>`

// iso-8859-1 encoded body with a raw 0xF3 byte ('ó' in Latin-1) in the description.
var sampleRSS2Latin1 = append([]byte(`<?xml version="1.0" encoding="iso-8859-1"?>
<rss version="2.0"><channel>
<item>
  <title>Reforma exterior actualizada</title>
  <link>https://example.com/latin1</link>
  <description>Descripci`), append([]byte{0xf3}, []byte(`n breve</description>
</item>
</channel></rss>`)...)...)

const sampleAtom = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Medio Atom de Prueba</title>
<entry>
  <title>Titular atom</title>
  <link href="https://example.com/atom-1" rel="alternate"/>
  <summary>Resumen del titular atom</summary>
  <published>2006-01-02T15:04:05Z</published>
</entry>
</feed>`

func TestParseRSS2(t *testing.T) {
	articles := parseRSS2([]byte(sampleRSS2), "medio-prueba", 10)
	if len(articles) != 2 {
		t.Fatalf("esperaba 2 artículos, dio %d", len(articles))
	}
	if articles[0].Title != "Primer titular" {
		t.Errorf("título inesperado: %q", articles[0].Title)
	}
	if articles[0].Link != "https://example.com/1" {
		t.Errorf("link inesperado: %q", articles[0].Link)
	}
	if articles[0].PublishedAt.IsZero() {
		t.Errorf("esperaba PublishedAt parseado, dio zero value")
	}
}

func TestParseRSS2RespectsMaxItems(t *testing.T) {
	articles := parseRSS2([]byte(sampleRSS2), "medio-prueba", 1)
	if len(articles) != 1 {
		t.Fatalf("esperaba 1 artículo con maxItems=1, dio %d", len(articles))
	}
}

func TestParseAtom(t *testing.T) {
	articles := parseAtom([]byte(sampleAtom), "medio-atom", 10)
	if len(articles) != 1 {
		t.Fatalf("esperaba 1 artículo, dio %d", len(articles))
	}
	if articles[0].Link != "https://example.com/atom-1" {
		t.Errorf("link inesperado: %q", articles[0].Link)
	}
	if articles[0].Snippet != "Resumen del titular atom" {
		t.Errorf("snippet inesperado: %q", articles[0].Snippet)
	}
}

func TestParseRSS2RejectsAtomInput(t *testing.T) {
	articles := parseRSS2([]byte(sampleAtom), "medio-atom", 10)
	if len(articles) != 0 {
		t.Fatalf("esperaba 0 artículos al parsear Atom como RSS2, dio %d", len(articles))
	}
}

func TestParseRDF(t *testing.T) {
	articles := parseRDF([]byte(sampleRDF), "medio-rdf", 10)
	if len(articles) != 1 {
		t.Fatalf("esperaba 1 artículo, dio %d", len(articles))
	}
	if articles[0].Title != "Titular RDF" {
		t.Errorf("título inesperado: %q", articles[0].Title)
	}
	if articles[0].PublishedAt.IsZero() {
		t.Errorf("esperaba PublishedAt parseado desde dc:date, dio zero value")
	}
}

func TestParseRSS2DecodesLatin1(t *testing.T) {
	articles := parseRSS2(sampleRSS2Latin1, "medio-latin1", 10)
	if len(articles) != 1 {
		t.Fatalf("esperaba 1 artículo, dio %d", len(articles))
	}
	want := "Descripción breve"
	if articles[0].Snippet != want {
		t.Errorf("snippet mal decodificado: got %q want %q", articles[0].Snippet, want)
	}
}
