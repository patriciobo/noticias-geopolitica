// Command checkfeeds fetches every configured source's RSS/Atom feed and
// reports which ones are alive, without touching the Claude API — useful to
// smoke-test config/sources.yaml and to re-audit the NO_RSS entries
// periodically, with zero API cost.
package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"noticias/core/internal/ingest"
)

func main() {
	sourcesPath := "../config/sources.yaml"
	if v := os.Getenv("SOURCES_PATH"); v != "" {
		sourcesPath = v
	}

	sources, err := ingest.LoadSources(sourcesPath)
	if err != nil {
		log.Fatalf("cargando sources: %v", err)
	}

	var ok, skipped, failed int
	var failures []string

	for _, src := range sources {
		if src.RSS == "" || src.RSS == "NO_RSS" {
			skipped++
			continue
		}
		arts, err := ingest.FetchHeadlines(src, 1)
		if err != nil || len(arts) == 0 {
			failed++
			msg := "sin items"
			if err != nil {
				msg = err.Error()
			}
			failures = append(failures, fmt.Sprintf("%-24s %s", src.ID, msg))
			continue
		}
		ok++
		fmt.Printf("%-24s OK  %s\n", src.ID, arts[0].Title)
	}

	sort.Strings(failures)
	fmt.Println()
	for _, f := range failures {
		fmt.Println(f)
	}

	fmt.Printf("\ntotal=%d ok=%d fallaron=%d sin_rss=%d\n", len(sources), ok, failed, skipped)
}
