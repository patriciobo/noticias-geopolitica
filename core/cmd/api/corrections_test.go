package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"noticias/core/internal/model"
)

func TestCorrectionsHandlerListsOnlyRegenerated(t *testing.T) {
	dir := t.TempDir()
	write := func(date string, p model.Provenance) {
		raw, _ := json.Marshal(p)
		os.WriteFile(filepath.Join(dir, date+".audit.json"), raw, 0o644)
	}
	now := time.Now().UTC()
	write("2026-09-25", model.Provenance{Revisions: []model.Revision{{Version: 1, GeneratedAt: now}}})
	write("2026-09-26", model.Provenance{Revisions: []model.Revision{{Version: 1, GeneratedAt: now}, {Version: 2, GeneratedAt: now, Reason: "error"}}})

	rec := httptest.NewRecorder()
	correctionsHandler(dir)(rec, httptest.NewRequest("GET", "/corrections", nil))

	var got []correction
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Date != "2026-09-26" || got[0].Revisions[1].Reason != "error" {
		t.Errorf("got %+v", got)
	}
}
