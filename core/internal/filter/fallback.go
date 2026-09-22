package filter

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"noticias/core/internal/model"
)

// ChainClassifier prueba una lista de proveedores en orden hasta que uno
// responda. Pensado para no perder artículos cuando un solo proveedor de
// fallback (ej. un modelo free de OpenRouter) falla puntualmente — con más
// de un candidato, un "Provider returned error" en uno no tira el artículo,
// prueba el siguiente.
//
// Cada link que falla se marca "muerto" para el resto de esta corrida y no
// se vuelve a intentar — evita pagar el viaje de red a algo que ya sabemos
// caído (ej. una cuota diaria agotada, que no se recupera sola en lo que
// queda del día) en cada uno de los siguientes artículos. El costo es que
// un fallo transitorio de un link también lo banca para el resto de la
// corrida — aceptable con varios candidatos en la lista: si uno se banca
// por una falla puntual, quedan los demás.
type ChainClassifier struct {
	links []Classifier
	dead  []atomic.Bool
}

func NewChainClassifier(links ...Classifier) *ChainClassifier {
	return &ChainClassifier{links: links, dead: make([]atomic.Bool, len(links))}
}

func (c *ChainClassifier) Classify(ctx context.Context, a model.Article, pre PrefilterResult) (model.Classification, error) {
	var lastErr error
	tried := false
	for i, link := range c.links {
		if c.dead[i].Load() {
			continue
		}
		tried = true
		cls, err := link.Classify(ctx, a, pre)
		if err == nil {
			return cls, nil
		}
		log.Printf("classify %q: proveedor %d/%d falló (%v), probando el siguiente", a.Title, i+1, len(c.links), err)
		c.dead[i].Store(true)
		lastErr = err
	}
	if !tried {
		return model.Classification{}, fmt.Errorf("los %d proveedores configurados ya habían fallado antes en esta corrida", len(c.links))
	}
	return model.Classification{}, fmt.Errorf("los %d proveedores configurados fallaron, último error: %w", len(c.links), lastErr)
}
