package main

import (
	"testing"

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
	a := buildAudit("gemini", []string{"m1", "openrouter:m2"}, []string{"s1"}, entries, 3)

	want := model.AuditCounts{Fetched: 5, PrefilterRejected: 1, ClassifierRejected: 1, ClassifierErrors: 1, Accepted: 2, LinksRemoved: 3}
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
