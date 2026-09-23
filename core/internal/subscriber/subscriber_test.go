package subscriber

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"  Persona@Ejemplo.com  ", "persona@ejemplo.com", false},
		{"sin-arroba", "", true},
		{"", "", true},
		{"dos@arrobas@ejemplo.com", "", true},
		{"Nombre Apellido <persona@ejemplo.com>", "", true}, // no aceptamos display name, solo la dirección pelada
	}
	for _, c := range cases {
		got, err := NormalizeEmail(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeEmail(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeEmail(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
