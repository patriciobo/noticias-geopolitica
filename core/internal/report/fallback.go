package report

import (
	"context"
	"log"

	"noticias/core/internal/model"
)

// FallbackSynthesizer prueba Primary y, si falla, cae a Secondary — mismo
// criterio que filter.FallbackClassifier, para el paso final de síntesis.
type FallbackSynthesizer struct {
	Primary   Synthesizer
	Secondary Synthesizer
}

func (f *FallbackSynthesizer) Synthesize(ctx context.Context, items []model.ClassifiedArticle) (string, error) {
	markdown, err := f.Primary.Synthesize(ctx, items)
	if err == nil {
		return markdown, nil
	}
	log.Printf("synthesize: proveedor principal falló (%v), probando fallback", err)
	return f.Secondary.Synthesize(ctx, items)
}
