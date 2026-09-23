package main

import (
	"testing"

	"noticias/core/internal/model"
)

func TestToPublicSourcesHidesFeedURL(t *testing.T) {
	got := toPublicSources([]model.Source{
		{Name: "A", Country: "Chile", Region: "latin_america", Stance: "oposicion", RSS: "https://a.example/rss"},
		{Name: "B", RSS: "NO_RSS"},
		{Name: "C", RSS: ""},
	})
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if !got[0].HasFeed || got[1].HasFeed || got[2].HasFeed {
		t.Errorf("HasFeed incorrecto: %+v", got)
	}
	if got[0].Country != "Chile" || got[0].Stance != "oposicion" {
		t.Errorf("metadatos no copiados: %+v", got[0])
	}
}
