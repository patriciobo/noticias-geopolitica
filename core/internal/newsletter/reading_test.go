package newsletter

import (
	"strings"
	"testing"
)

func TestReadingMinutes(t *testing.T) {
	body := strings.Repeat("palabra ", 400) + "[[1]](<https://x>) **negrita**"
	md := "## Resumen ejecutivo\n\n" + body + "\n## Noticias utilizadas\n\n" + strings.Repeat("- [nota](<https://y>) — Medio (País)\n", 500)
	if got := ReadingMinutes(md); got != 2 {
		t.Errorf("ReadingMinutes = %d, want 2 (~404 palabras, sin contar la lista de notas)", got)
	}
	if got := ReadingMinutes("hola"); got != 1 {
		t.Errorf("mínimo 1 minuto, got %d", got)
	}
}
