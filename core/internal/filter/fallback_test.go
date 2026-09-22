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
