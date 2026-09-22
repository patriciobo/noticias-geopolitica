package filter

import (
	"context"
	"log"

	"noticias/core/internal/model"
)

// FallbackClassifier prueba Primary y, si falla (cuota agotada, caída del
// proveedor, lo que sea — Primary ya reintentó lo que tenía que reintentar
// puertas adentro), cae a Secondary en vez de perder el artículo. Pensado
// para no perder la corrida del día entero por un solo proveedor sin cupo.
type FallbackClassifier struct {
	Primary   Classifier
	Secondary Classifier
}

func (f *FallbackClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	cls, err := f.Primary.Classify(ctx, a, pre)
	if err == nil {
		return cls, nil
	}
	log.Printf("classify %q: proveedor principal falló (%v), probando fallback", a.Title, err)
	return f.Secondary.Classify(ctx, a, pre)
}
