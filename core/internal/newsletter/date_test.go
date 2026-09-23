package newsletter

import "testing"

func TestFormatDateEs(t *testing.T) {
	got := FormatDateEs("2026-09-23")
	want := "miércoles, 23 de septiembre de 2026"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatDateEsInvalid(t *testing.T) {
	if got := FormatDateEs("no-es-fecha"); got != "no-es-fecha" {
		t.Errorf("got %q", got)
	}
}
