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
	// Used es el modelo del link que produjo el último informe válido.
	Used string
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
			c.Used = ModelName(link)
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

// ModelName devuelve el modelo de un sintetizador, o "" si no lo expone.
// Para una cadena, el del último link que respondió bien.
func ModelName(s Synthesizer) string {
	switch v := s.(type) {
	case *ChainSynthesizer:
		return v.Used
	case interface{ ModelName() string }:
		return v.ModelName()
	}
	return ""
}

// Completer lo cumplen los sintetizadores que aceptan un pedido de chat
// genérico (hoy, los compatibles con OpenAI: Gemini, OpenRouter).
type Completer interface {
	Complete(ctx context.Context, system, user string, temperature float64) (string, error)
}

// Complete prueba en orden los links de la cadena que saben completar. En
// la práctica el chequeo de fidelidad lo hace el mismo modelo principal que
// redactó: es un control de consistencia con las notas, no una revisión
// independiente (esa es la revisión humana por muestreo).
func (c *ChainSynthesizer) Complete(ctx context.Context, system, user string, temperature float64) (string, error) {
	var lastErr error = fmt.Errorf("ningún proveedor de la cadena acepta pedidos genéricos")
	for _, link := range c.Links {
		comp, ok := link.(Completer)
		if !ok {
			continue
		}
		out, err := comp.Complete(ctx, system, user, temperature)
		if err == nil {
			return out, nil
		}
		log.Printf("complete: proveedor %s falló (%v), probando el siguiente", ModelName(link), err)
		lastErr = err
	}
	return "", lastErr
}
