package filter

import (
	"context"

	"noticias/core/internal/model"
)

// Classifier decides, precisely, whether an article that survived the
// prefilter is genuinely international/multinational in scope.
type Classifier interface {
	Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error)
}
