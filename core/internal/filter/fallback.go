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

// ClassifyBatch aplica la misma cadena de fallback que Classify, pero a
// nivel de lote: cada link se prueba con TODO lo que sigue pendiente, y solo
// lo que le falló a ese link pasa al siguiente. Un link se marca muerto para
// el resto de la corrida únicamente cuando le falló el batch COMPLETO que
// recibió — un fallo puntual de uno o dos items (ej. el modelo se salteó un
// índice y el top-up individual también falló) no lo banca: sería tirar por
// la borda un proveedor que en los hechos está funcionando bien, empujando
// tráfico de más al fallback (con cupo mucho más chico) sin necesidad.
func (c *ChainClassifier) ClassifyBatch(ctx context.Context, items []BatchItem) []BatchResult {
	results := make([]BatchResult, len(items))
	pending := items
	pendingIdx := make([]int, len(items))
	for i := range pendingIdx {
		pendingIdx[i] = i
	}

	var lastErr error
	for i, link := range c.links {
		if len(pending) == 0 {
			break
		}
		if c.dead[i].Load() {
			continue
		}

		linkResults := ClassifyBatch(ctx, link, pending)

		var stillPending []BatchItem
		var stillPendingIdx []int
		failCount := 0
		for j, r := range linkResults {
			origIdx := pendingIdx[j]
			if r.Err != nil {
				failCount++
				lastErr = r.Err
				stillPending = append(stillPending, pending[j])
				stillPendingIdx = append(stillPendingIdx, origIdx)
				continue
			}
			results[origIdx] = r
		}

		switch {
		case failCount == 0:
			// todo el batch salió bien en este link
		case failCount == len(pending):
			log.Printf("classify batch: proveedor %d/%d falló para los %d items del batch (%v), probando el siguiente", i+1, len(c.links), failCount, lastErr)
			c.dead[i].Store(true)
		default:
			log.Printf("classify batch: proveedor %d/%d falló para %d/%d items del batch, probando el siguiente para esos", i+1, len(c.links), failCount, len(pending))
		}

		pending = stillPending
		pendingIdx = stillPendingIdx
	}

	for _, origIdx := range pendingIdx {
		if lastErr == nil {
			results[origIdx] = BatchResult{Err: fmt.Errorf("los %d proveedores configurados ya habían fallado antes en esta corrida", len(c.links))}
			continue
		}
		results[origIdx] = BatchResult{Err: fmt.Errorf("los %d proveedores configurados fallaron, último error: %w", len(c.links), lastErr)}
	}

	return results
}
