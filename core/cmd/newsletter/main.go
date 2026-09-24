// Command newsletter manda por correo el resumen del día (Resumen
// ejecutivo + Resumen por región) a los suscriptores activos. Corre una vez
// por día, después de cmd/ingest, como paso separado del pipeline —
// opcional: sin DATABASE_URL o BREVO_API_KEY configurados no hace nada.
package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"noticias/core/internal/config"
	"noticias/core/internal/newsletter"
	"noticias/core/internal/report"
	"noticias/core/internal/store"
	"noticias/core/internal/subscriber"
)

func main() {
	config.LoadDotEnv(".env")

	dsn := os.Getenv("DATABASE_URL")
	brevoKey := os.Getenv("BREVO_API_KEY")
	if dsn == "" || brevoKey == "" {
		log.Print("DATABASE_URL o BREVO_API_KEY no configurados, no se envía newsletter")
		return
	}

	outDir := config.EnvOrDefault("OUT_DIR", "./out")
	reportDate := config.EnvOrDefault("REPORT_DATE", time.Now().Format("2006-01-02"))
	apiBaseURL := config.EnvOrDefault("API_BASE_URL", "http://localhost:8080")
	siteURL := config.EnvOrDefault("SITE_URL", "http://localhost:3000")
	senderEmail := config.EnvOrDefault("BREVO_SENDER_EMAIL", "")
	senderName := config.EnvOrDefault("BREVO_SENDER_NAME", "Radar Global")
	subject := "Radar Global"
	// En CI, un SITE_URL/API_BASE_URL sin configurar cae al default de
	// localhost y el mail sale con imágenes y links rotos (pasó hasta el
	// 2026-09-24: la variable SITE_URL no existía en el repo). Mejor no
	// mandar nada que mandar eso a todos los suscriptores.
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		for name, v := range map[string]string{"SITE_URL": siteURL, "API_BASE_URL": apiBaseURL} {
			if strings.Contains(v, "localhost") {
				log.Fatalf("%s apunta a %s en GitHub Actions — configurá la variable del repo (Settings → Secrets and variables → Actions → Variables); no se envía el newsletter", name, v)
			}
		}
	}
	cafecitoURL := config.EnvOrDefault("DONATION_CAFECITO_URL", "https://cafecito.app/radarglobal")
	tecitoURL := config.EnvOrDefault("DONATION_TECITO_URL", "https://tecito.app/radarglobal")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := store.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("no se pudo conectar a la base: %v", err)
	}
	defer db.Close()
	repo := subscriber.NewRepository(db)

	reportPath := filepath.Join(outDir, reportDate+".md")
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		log.Printf("no existe %s todavía, se omite el envío: %v", reportPath, err)
		return
	}

	sections := report.Sections(string(raw))
	abstract, ok := sections["Resumen ejecutivo"]
	if !ok {
		log.Fatal("el reporte no tiene sección \"Resumen ejecutivo\"")
	}
	regionSummary, ok := sections["Resumen por región"]
	if !ok {
		log.Fatal("el reporte no tiene sección \"Resumen por región\"")
	}
	populated, empty := newsletter.SplitRegions(regionSummary)

	// Altas que nunca se confirmaron no se guardan para siempre (ver
	// subscriber.ConfirmTokenTTL). Un fallo acá no frena el envío.
	if purged, err := repo.PurgeStalePending(ctx); err != nil {
		log.Printf("no se pudieron purgar altas pendientes vencidas: %v", err)
	} else if purged > 0 {
		log.Printf("purgadas %d altas pendientes sin confirmar", purged)
	}

	active, err := repo.ListActive(ctx)
	if err != nil {
		log.Fatalf("no se pudo listar suscriptores activos: %v", err)
	}
	if len(active) == 0 {
		log.Print("no hay suscriptores activos, no se envía nada")
		return
	}

	base := newsletter.EmailData{
		SiteURL:          siteURL,
		EditionURL:       fmt.Sprintf("%s/reportes/%s", siteURL, reportDate),
		ReportErrorURL:   fmt.Sprintf("%s/issues/new?template=error-en-edicion.yml&fecha=%s", config.EnvOrDefault("REPO_URL", "https://github.com/patriciobo/noticias-geopolitica"), reportDate),
		DateLabel:        newsletter.FormatDateEs(reportDate),
		AbstractHTML:     template.HTML(newsletter.MarkdownFragmentToHTML(abstract)),
		PopulatedRegions: populated,
		EmptyRegions:     empty,
		CafecitoURL:      cafecitoURL,
		TecitoURL:        tecitoURL,
	}

	recipients := make([]newsletter.Recipient, 0, len(active))
	for _, s := range active {
		data := base
		data.UnsubscribeURL = apiBaseURL + "/subscribers/unsubscribe?token=" + s.UnsubscribeToken
		html, err := newsletter.RenderEmail(data)
		if err != nil {
			log.Fatalf("no se pudo renderizar el email para %s: %v", s.Email, err)
		}
		recipients = append(recipients, newsletter.Recipient{Email: s.Email, HTMLContent: html, UnsubscribeURL: data.UnsubscribeURL})
	}

	brevo := newsletter.NewBrevo(brevoKey, senderEmail, senderName)
	sent, failed, err := brevo.Send(ctx, subject, recipients)
	if err != nil {
		log.Printf("newsletter: hubo errores enviando (primer error: %v)", err)
	}
	log.Printf("newsletter: %d enviados, %d fallidos de %d suscriptores activos", sent, failed, len(active))
}
