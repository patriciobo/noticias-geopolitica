package report

// SystemPrompts devuelve el prompt de sistema de la síntesis por nombre,
// para publicar su hash en el registro de auditoría de cada edición (ver
// model.Provenance).
func SystemPrompts() map[string]string {
	return map[string]string{
		"synthesize": synthesisSystemPrompt,
	}
}
