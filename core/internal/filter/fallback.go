package filter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"

	"noticias/core/internal/model"
)

// ErrQuotaExhausted marca un fallo que no se recupera en lo que queda de la
// corrida (cuota diaria/mensual agotada) — el ChainClassifier banca el link
// en el acto en vez de esperar maxConsecutiveFailures.
var ErrQuotaExhausted = errors.New("cuota del proveedor agotada")

// maxConsecutiveFailures es cuántos fallos seguidos (sin ningún éxito en el
// medio) hacen falta para dar un link por muerto el resto de la corrida.
// Con 1 (el comportamiento anterior) un solo timeout puntual de Gemini lo
// sacaba de juego y todo el volumen caía a los free de OpenRouter, que con
// su cupo chico terminaban fallando también — el 2026-09-24 eso dejó 48
// titulares sin clasificar y a Europa y Asia Oriental vacías.
const maxConsecutiveFailures = 3

// ChainClassifier prueba una lista de proveedores en orden hasta que uno
// responda. Pensado para no perder artículos cuando un solo proveedor de
// fallback (ej. un modelo free de OpenRouter) falla puntualmente — con más
// de un candidato, un "Provider returned error" en uno no tira el artículo,
// prueba el siguiente.
//
// Un link se marca "muerto" para el resto de esta corrida (y no se vuelve a
// intentar) cuando acumula maxConsecutiveFailures fallos seguidos, o en el
// acto si el error es ErrQuotaExhausted — evita pagar el viaje de red a
// algo que ya sabemos caído, sin sacar de juego a un proveedor sano por un
// timeout puntual. Mientras no esté muerto, lo que le falla a un link pasa
// igual al siguiente, así que ningún fallo transitorio pierde artículos.
type ChainClassifier struct {
	links    []Classifier
	dead     []atomic.Bool
	failures []atomic.Int32
}

func NewChainClassifier(links ...Classifier) *ChainClassifier {
	return &ChainClassifier{
		links:    links,
		dead:     make([]atomic.Bool, len(links)),
		failures: make([]atomic.Int32, len(links)),
	}
}

// recordFailure cuenta un fallo del link i y lo banca si corresponde.
// Devuelve true si el link quedó muerto con este fallo.
func (c *ChainClassifier) recordFailure(i int, err error) bool {
	if errors.Is(err, ErrQuotaExhausted) || c.failures[i].Add(1) >= maxConsecutiveFailures {
		return !c.dead[i].Swap(true)
	}
	return false
}

func (c *ChainClassifier) recordSuccess(i int) {
	c.failures[i].Store(0)
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
			c.recordSuccess(i)
			return cls, nil
		}
		log.Printf("classify %q: proveedor %d/%d falló (%v), probando el siguiente", a.Title, i+1, len(c.links), err)
		if c.recordFailure(i, err) {
			log.Printf("classify: proveedor %d/%d descartado por el resto de la corrida", i+1, len(c.links))
		}
		lastErr = err
	}
	if !tried {
		return model.Classification{}, fmt.Errorf("los %d proveedores configurados ya habían fallado antes en esta corrida", len(c.links))
	}
	return model.Classification{}, fmt.Errorf("los %d proveedores configurados fallaron, último error: %w", len(c.links), lastErr)
}

// ClassifyBatch aplica la misma cadena de fallback que Classify, pero a
// nivel de lote: cada link se prueba con TODO lo que sigue pendiente, y solo
// lo que le falló a ese link pasa al siguiente. Solo cuenta como fallo del
// link un batch COMPLETO fallido — un fallo puntual de uno o dos items (ej.
// el modelo se salteó un índice y el top-up individual también falló) no:
// sería empujar hacia la muerte a un proveedor que en los hechos está
// funcionando bien, mandando tráfico de más al fallback (con cupo mucho más
// chico) sin necesidad.
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
			c.recordSuccess(i)
		case failCount == len(pending):
			log.Printf("classify batch: proveedor %d/%d falló para los %d items del batch (%v), probando el siguiente", i+1, len(c.links), failCount, lastErr)
			if c.recordFailure(i, lastErr) {
				log.Printf("classify batch: proveedor %d/%d descartado por el resto de la corrida", i+1, len(c.links))
			}
		default:
			c.recordSuccess(i)
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
