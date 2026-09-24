package report

import (
	"context"
	"fmt"
	"log"
	"strings"
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

func (c *ChainSynthesizer) Synthesize(ctx context.Context, in Input) (string, error) {
	var lastErr error
	for i, link := range c.Links {
		markdown, err := link.Synthesize(ctx, in)
		if err == nil {
			err = validateReport(markdown)
		}
		if err == nil {
			return markdown, nil
		}
		log.Printf("synthesize: proveedor %d/%d falló (%v), probando el siguiente", i+1, len(c.Links), err)
		lastErr = err
	}
	return "", fmt.Errorf("los %d proveedores configurados fallaron, último error: %w", len(c.Links), lastErr)
}

// requiredSections son los encabezados que el prompt de síntesis exige
// siempre. Si falta alguno, el modelo no hizo el trabajo pedido o la
// respuesta vino cortada — pasa con los routers de modelos free (a veces
// caen en un modelo de moderación que contesta "User Safety: safe") y pasó
// con un informe truncado a mitad de "Resumen por región" el 2026-09-24.
// Mejor probar el siguiente proveedor que publicar eso.
var requiredSections = []string{
	"## Resumen ejecutivo",
	"## Resumen por región",
	"## Clima internacional",
	"## Empresas potencialmente afectadas",
}

func validateReport(markdown string) error {
	for _, h := range requiredSections {
		if !strings.Contains(markdown, h) {
			return fmt.Errorf("la respuesta no tiene la sección %q (output: %s)", h, truncate(markdown, 200))
		}
	}
	return nil
}
