package report

import (
	"context"
	"fmt"
	"log"

	"noticias/core/internal/model"
)

// ChainSynthesizer prueba una lista de proveedores en orden hasta que uno
// responda — mismo criterio que filter.ChainClassifier, pero sin estado de
// "muerto" persistente: síntesis se llama una sola vez por corrida, no hace
// falta recordar fallos entre llamadas.
type ChainSynthesizer struct {
	Links []Synthesizer
}

func NewChainSynthesizer(links ...Synthesizer) *ChainSynthesizer {
	return &ChainSynthesizer{Links: links}
}

func (c *ChainSynthesizer) Synthesize(ctx context.Context, items []model.ClassifiedArticle) (string, error) {
	var lastErr error
	for i, link := range c.Links {
		markdown, err := link.Synthesize(ctx, items)
		if err == nil {
			return markdown, nil
		}
		log.Printf("synthesize: proveedor %d/%d falló (%v), probando el siguiente", i+1, len(c.Links), err)
		lastErr = err
	}
	return "", fmt.Errorf("los %d proveedores configurados fallaron, último error: %w", len(c.Links), lastErr)
}
