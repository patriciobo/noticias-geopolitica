package newsletter

import (
	"fmt"
	"time"
)

var weekdaysEs = [...]string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
var monthsEs = [...]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}

// FormatDateEs da el mismo formato que ya usa el blog para la fecha de una
// edición (ver formatDate en web/src/components/ReportView.tsx, que usa
// toLocaleDateString("es-AR")) — acá no hay Intl, así que se arma a mano.
// reportDate viene en formato YYYY-MM-DD, el mismo que usa cmd/ingest para
// nombrar el archivo. La mayúscula del email es solo CSS (text-transform),
// igual que en la web — esto devuelve el texto en minúscula natural.
func FormatDateEs(reportDate string) string {
	t, err := time.Parse("2006-01-02", reportDate)
	if err != nil {
		return reportDate
	}
	return fmt.Sprintf("%s, %d de %s de %d", weekdaysEs[t.Weekday()], t.Day(), monthsEs[t.Month()-1], t.Year())
}
