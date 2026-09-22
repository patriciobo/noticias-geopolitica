package filter

import (
	"context"
	"errors"
	"testing"

	"noticias/core/internal/model"
)

type stubClassifier struct {
	cls model.Classification
	err error
}

func (s stubClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	return s.cls, s.err
}

func TestFallbackClassifierUsesPrimaryOnSuccess(t *testing.T) {
	f := &FallbackClassifier{
		Primary:   stubClassifier{cls: model.Classification{Reason: "primary"}},
		Secondary: stubClassifier{cls: model.Classification{Reason: "secondary"}},
	}
	got, err := f.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reason != "primary" {
		t.Errorf("expected primary result, got %q", got.Reason)
	}
}

func TestFallbackClassifierFallsBackOnPrimaryError(t *testing.T) {
	f := &FallbackClassifier{
		Primary:   stubClassifier{err: errors.New("cuota agotada")},
		Secondary: stubClassifier{cls: model.Classification{Reason: "secondary"}},
	}
	got, err := f.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reason != "secondary" {
		t.Errorf("expected secondary result, got %q", got.Reason)
	}
}

func TestFallbackClassifierPropagatesSecondaryError(t *testing.T) {
	wantErr := errors.New("secondary también falló")
	f := &FallbackClassifier{
		Primary:   stubClassifier{err: errors.New("primary falló")},
		Secondary: stubClassifier{err: wantErr},
	}
	_, err := f.Classify(context.Background(), model.Article{}, PrefilterResult{})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected secondary error to propagate, got %v", err)
	}
}

// countingClassifier cuenta cuántas veces se llamó — usado para verificar
// que, tras la primera falla, Primary no se vuelve a invocar.
type countingClassifier struct {
	calls *int
	cls   model.Classification
	err   error
}

func (c countingClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	*c.calls++
	return c.cls, c.err
}

func TestFallbackClassifierStopsCallingDeadPrimary(t *testing.T) {
	primaryCalls := 0
	f := &FallbackClassifier{
		Primary:   countingClassifier{calls: &primaryCalls, err: errors.New("cuota agotada")},
		Secondary: stubClassifier{cls: model.Classification{Reason: "secondary"}},
	}

	for i := 0; i < 5; i++ {
		got, err := f.Classify(context.Background(), model.Article{}, PrefilterResult{})
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if got.Reason != "secondary" {
			t.Errorf("call %d: expected secondary result, got %q", i, got.Reason)
		}
	}

	if primaryCalls != 1 {
		t.Errorf("expected primary to be called exactly once (dies after first failure), got %d calls", primaryCalls)
	}
}
