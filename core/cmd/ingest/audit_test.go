package main

import (
	"testing"
	"time"

	"noticias/core/internal/model"
)

func TestBuildAudit(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "dueño/repo")
	t.Setenv("GITHUB_RUN_ID", "42")
	t.Setenv("GITHUB_SHA", "abc123")

	entries := []model.AuditEntry{
		{Source: "B", Title: "z", Stage: model.StageAccepted},
		{Source: "A", Title: "y", Stage: model.StagePrefilterRejected},
		{Source: "B", Title: "a", Stage: model.StageClassifierRejected},
		{Source: "C", Title: "x", Stage: model.StageClassifierError},
		{Source: "A", Title: "b", Stage: model.StageAccepted},
	}
	statuses := []model.SourceStatus{
		{Name: "A", Status: model.SourceOK, Headlines: 2},
		{Name: "B", Status: model.SourceError, Detail: "403"},
	}
	a := buildAudit("gemini", []string{"m1", "openrouter:m2"}, []string{"s1"}, entries, statuses, 3)
	if a.Counts.SourcesConfigured != 2 || a.Counts.SourcesResponded != 1 || len(a.SourceProblems) != 1 || a.SourceProblems[0].Name != "B" {
		t.Errorf("estado de medios mal resumido: counts=%+v problems=%+v", a.Counts, a.SourceProblems)
	}

	want := model.AuditCounts{SourcesConfigured: 2, SourcesResponded: 1, Fetched: 5, PrefilterRejected: 1, ClassifierRejected: 1, ClassifierErrors: 1, Accepted: 2, LinksRemoved: 3}
	if a.Counts != want {
		t.Errorf("Counts = %+v, want %+v", a.Counts, want)
	}
	if a.RunURL != "https://github.com/dueño/repo/actions/runs/42" || a.Commit != "abc123" {
		t.Errorf("provenance de Actions: run=%q commit=%q", a.RunURL, a.Commit)
	}
	for _, name := range []string{"classify", "classify_batch", "synthesize"} {
		if len(a.PromptSHA256[name]) != 64 {
			t.Errorf("falta hash del prompt %q: %v", name, a.PromptSHA256)
		}
	}
	if a.Headlines[0].Source != "A" || a.Headlines[0].Title != "b" || a.Headlines[4].Source != "C" {
		t.Errorf("titulares no quedaron ordenados por fuente y título: %+v", a.Headlines)
	}
}

func TestDropStale(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	mk := func(title string, pub time.Time) struct {
		Source  model.Source
		Article model.Article
	} {
		return struct {
			Source  model.Source
			Article model.Article
		}{Source: model.Source{Name: "X"}, Article: model.Article{Title: title, PublishedAt: pub}}
	}
	items := []struct {
		Source  model.Source
		Article model.Article
	}{
		mk("hoy", now.Add(-2*time.Hour)),
		mk("2020", time.Date(2020, 2, 27, 0, 0, 0, 0, time.UTC)),
		mk("sin fecha", time.Time{}),
	}

	got := dropStale(items, now, 72*time.Hour)

	if len(got) != 2 || got[0].Article.Title != "hoy" || got[1].Article.Title != "sin fecha" {
		t.Errorf("dropStale = %+v", got)
	}
}
