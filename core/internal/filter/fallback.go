package filter

import (
	"context"
	"log"
	"sync/atomic"

	"noticias/core/internal/model"
)

// FallbackClassifier prueba Primary y, si falla (cuota agotada, caída del
// proveedor — Primary ya reintentó lo que tenía que reintentar puertas
// adentro antes de rendirse), cae a Secondary en vez de perder el artículo.
// Pensado para no perder la corrida del día entero por un solo proveedor
// sin cupo.
//
// Una vez que Primary falla la primera vez, se marca "muerto" para el
// resto de esta corrida y las llamadas siguientes van directo a Secondary
// sin volver a intentarlo. Sin esto, una cuota diaria agotada (no se
// recupera sola en el resto del día) hace que CADA artículo pague el
// viaje de red completo a un proveedor que ya sabemos que va a fallar,
// antes de recién ahí caer al fallback — con cientos de artículos por
// corrida, esa duplicación de latencia es la diferencia entre minutos y
// horas.
type FallbackClassifier struct {
	Primary   Classifier
	Secondary Classifier

	primaryDead atomic.Bool
}

func (f *FallbackClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	if !f.primaryDead.Load() {
		cls, err := f.Primary.Classify(ctx, a, pre)
		if err == nil {
			return cls, nil
		}
		log.Printf("classify %q: proveedor principal falló (%v), paso a fallback para el resto de la corrida", a.Title, err)
		f.primaryDead.Store(true)
	}
	return f.Secondary.Classify(ctx, a, pre)
}
