package newsletter

import "testing"

func TestSplitRegions(t *testing.T) {
	body := `### América del Norte

- **Estados Unidos**: algo pasó.

### Europa

Sin novedades relevantes hoy.

### Asia Oriental

Sin novedades relevantes hoy.
`
	populated, empty := SplitRegions(body)

	if len(populated) != 1 || populated[0].Label != "América del Norte" {
		t.Errorf("populated = %+v", populated)
	}
	if len(populated) == 1 && populated[0].HTML == "" {
		t.Error("expected non-empty HTML for populated region")
	}

	if len(empty) != 2 || empty[0].Label != "Europa" || empty[1].Label != "Asia Oriental" {
		t.Errorf("empty = %+v", empty)
	}
}

func TestSplitRegionsAllPopulated(t *testing.T) {
	body := "### América Latina\n\n- **Argentina**: algo.\n"
	populated, empty := SplitRegions(body)
	if len(populated) != 1 || len(empty) != 0 {
		t.Errorf("populated=%+v empty=%+v", populated, empty)
	}
}
