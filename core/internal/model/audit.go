package model

import "time"

// Audit es el registro público de cómo se armó una edición: qué titulares
// entraron, qué pasó con cada uno y con qué configuración corrió el
// pipeline. Se publica junto al reporte (reports/FECHA.audit.json) para que
// cualquiera pueda verificar qué quedó afuera y por qué — no solo lo que
// terminó en el informe.
type Audit struct {
	Provenance
	Headlines []AuditEntry `json:"headlines"`
}

// Provenance describe la corrida que generó el reporte.
type Provenance struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	Commit           string            `json:"commit,omitempty"`  // commit del código que corrió (GITHUB_SHA)
	RunURL           string            `json:"run_url,omitempty"` // log público de la corrida en GitHub Actions
	Provider         string            `json:"provider"`
	ClassifyModels   []string          `json:"classify_models"`   // en orden: principal y fallbacks
	SynthesizeModels []string          `json:"synthesize_models"` // en orden: principal y fallbacks
	PromptSHA256     map[string]string `json:"prompt_sha256"`     // hash de cada prompt de sistema, verificable contra el código del commit
	Counts           AuditCounts       `json:"counts"`
}

type AuditCounts struct {
	Fetched            int `json:"fetched"`             // titulares descargados de los feeds
	PrefilterRejected  int `json:"prefilter_rejected"`  // descartados por el prefiltro de palabras clave
	ClassifierRejected int `json:"classifier_rejected"` // descartados por el clasificador (no internacionales)
	ClassifierErrors   int `json:"classifier_errors"`   // el clasificador falló y el titular quedó afuera
	Accepted           int `json:"accepted"`            // entraron a la síntesis
	LinksRemoved       int `json:"links_removed"`       // enlaces que escribió el LLM y se sacaron por no ser de una nota procesada
}

// Etapas posibles de un AuditEntry.
const (
	StagePrefilterRejected  = "prefilter_rejected"
	StageClassifierRejected = "classifier_rejected"
	StageClassifierError    = "classifier_error"
	StageAccepted           = "accepted"
)

// AuditEntry es el destino de un titular descargado.
type AuditEntry struct {
	Source       string  `json:"source"`
	Country      string  `json:"country"`
	Title        string  `json:"title"`
	Link         string  `json:"link,omitempty"`
	Stage        string  `json:"stage"`
	Reason       string  `json:"reason,omitempty"`        // explicación del clasificador, o el error
	RelationType string  `json:"relation_type,omitempty"` // solo si pasó por el clasificador
	Confidence   float64 `json:"confidence,omitempty"`
}
