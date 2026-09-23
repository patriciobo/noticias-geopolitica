package filter

// SystemPrompts devuelve los prompts de sistema del clasificador por
// nombre, para publicar su hash en el registro de auditoría de cada
// edición (ver model.Provenance).
func SystemPrompts() map[string]string {
	return map[string]string{
		"classify":       classifySystemPrompt,
		"classify_batch": batchClassifySystemPrompt,
	}
}
