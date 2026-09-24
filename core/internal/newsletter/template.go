package newsletter

import (
	"bytes"
	"fmt"
	"html/template"
)

// EmailData son los datos ya resueltos que necesita el template — el
// llamador (cmd/newsletter) arma esto a partir del reporte del día, el
// template solo se encarga del layout.
type EmailData struct {
	SiteURL          string // home del blog, base de los links de header/CTA
	EditionURL       string // link a la edición completa de este día
	DateLabel        string // "miércoles, 23 de septiembre de 2026" (CSS lo pasa a mayúscula)
	AbstractHTML     template.HTML
	PopulatedRegions []RegionBlock
	EmptyRegions     []RegionBlock
	UnsubscribeURL   string
	CafecitoURL      string // donaciones Argentina; vacío = no se muestra el botón
	TecitoURL        string // donaciones exterior; vacío = no se muestra el botón
}

// emailTemplateSrc usa layout de tablas (no flexbox/grid) por compatibilidad
// con clientes de correo viejos (Outlook en particular no soporta CSS
// moderno). Las regiones sin novedades van en una tabla de celdas chicas en
// vez de recibir el mismo espacio que una región con contenido real — pasa
// seguido que una o más regiones no tengan nada que informar ese día (ver
// SplitRegions), y day el mismo tratamiento ahí se ve roto/vacío.
const emailTemplateSrc = `<!doctype html>
<html lang="es">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0; padding:0; background:#eeeeee; font-family: Georgia, 'Times New Roman', serif; color:#171717;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#eeeeee;">
<tr><td align="center" style="padding: 24px 12px;">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="background:#ffffff; max-width:600px;">

<tr><td align="center" style="background:#171717; padding: 22px 28px 14px 28px;">
<table role="presentation" cellpadding="0" cellspacing="0"><tr>
<td><a href="{{.SiteURL}}"><img src="{{.SiteURL}}/email/logo.png" alt="Noticias Internacionales" width="104" height="104" style="display:block; border:0; outline:none; text-decoration:none; width:104px; height:104px; margin:0 auto;"></a></td>
<td style="padding-left:14px;"><a href="{{.SiteURL}}"><img src="{{.SiteURL}}/email/nombre-sf.png" alt="Noticias Internacionales" width="139" height="104" style="display:block; border:0; outline:none; text-decoration:none; width:139px; height:104px; margin:0 auto;"></a></td>
</tr></table>
<div style="text-align:center; font-family: Georgia, serif; font-size:15px; font-weight:bold; letter-spacing:0.14em; text-transform:uppercase; color:#ffffff; padding-top:12px;">Radar Global</div>
<div style="text-align:center; padding-top:12px; font-size:12px; color:#cccccc;">
<a href="{{.SiteURL}}/#ediciones-anteriores" style="color:#cccccc; text-decoration:none; margin-left:14px;">Ediciones</a>
<a href="{{.SiteURL}}/#resumen-por-region" style="color:#cccccc; text-decoration:none; margin-left:14px;">Regiones</a>
<a href="{{.SiteURL}}/#fuentes-consultadas" style="color:#cccccc; text-decoration:none; margin-left:14px;">Fuentes</a>
</div>
</td></tr>
<tr><td style="background:#c81e1e; height:3px; font-size:1px; line-height:1px;">&nbsp;</td></tr>

<tr><td style="padding: 28px 28px 8px 28px;">

<span style="display:inline-block; font-size:11px; font-weight:bold; letter-spacing:0.05em; text-transform:uppercase; color:#c81e1e; border:1px solid #c81e1e; padding:3px 9px;">Editorial</span>

<h1 style="margin:14px 0 20px; font-size:26px; line-height:1.25; font-weight:800;">Un informe por día: comercio, industria y empresas multinacionales, con enlaces a las notas originales de cada medio.</h1>

<p style="margin-top:8px; font-family: 'Courier New', monospace; font-size:12px; font-weight:bold; letter-spacing:0.05em; text-transform:uppercase; color:#666666;">{{.DateLabel}}</p>

<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin: 6px 0 22px;">
<tr><td style="border-left:4px solid #c81e1e; background:#fdf1f1; padding: 14px 18px; font-size:15px; line-height:1.55;">
{{.AbstractHTML}}
</td></tr>
</table>

<h2 style="margin:0 0 14px; padding-bottom:8px; font-size:17px; border-bottom:2px solid #171717;">&#9632; Resumen por región</h2>

{{range .PopulatedRegions}}
<h3 style="margin:16px 0 6px; font-size:15px;">{{.Label}}</h3>
<div style="font-size:14px; line-height:1.55;">{{.HTML}}</div>
{{end}}

{{if .EmptyRegions}}
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-top:18px; border-top:1px solid #dddddd; border-bottom:1px solid #dddddd;">
<tr>
{{range .EmptyRegions}}
<td valign="top" width="{{ len $.EmptyRegions | divPercent }}%" style="padding:14px 12px 14px 0; font-size:13px;">
<strong>{{.Label}}</strong><br>
<span style="color:#888888; font-style:italic;">Sin novedades relevantes hoy.</span>
</td>
{{end}}
</tr>
</table>
{{end}}

<p style="margin:24px 0 16px; font-size:13px; line-height:1.5; color:#666666;">La edición completa incluye además el clima de comercio, industria y materias primas, las empresas potencialmente afectadas por región, las fuentes consultadas y las ediciones anteriores.</p>

<table role="presentation" cellpadding="0" cellspacing="0"><tr><td style="background:#c81e1e; padding: 10px 18px; margin-bottom: 24px;">
<a href="{{.EditionURL}}" style="color:#ffffff; font-weight:bold; font-size:14px; text-decoration:none;">Ver edición completa &rarr;</a>
</td></tr></table>

{{if or .CafecitoURL .TecitoURL}}
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-top:26px; border:1px solid #dddddd; background:#fdf1f1;">
<tr><td style="padding:16px 18px;">
<p style="margin:0 0 12px; font-size:13px; line-height:1.55; color:#171717;">¿Te sirvió el resumen de hoy? Radar Global es un esfuerzo independiente: invitanos un café y ayudanos a seguir informando, día a día.</p>
<table role="presentation" cellpadding="0" cellspacing="0">
{{if .CafecitoURL}}
<tr><td style="background:#c81e1e; border-radius:8px;">
<a href="{{.CafecitoURL}}" style="display:block; padding:8px 14px; color:#ffffff; font-weight:bold; font-size:13px; text-decoration:none;"><img src="{{.SiteURL}}/email/donar-cafecito.png" width="22" height="22" alt="" style="vertical-align:middle; border:0; border-radius:5px; margin-right:8px;">Invitanos un cafecito &middot; desde $500 ARS</a>
</td></tr>
{{end}}
{{if .TecitoURL}}
<tr><td style="height:10px; font-size:1px; line-height:1px;">&nbsp;</td></tr>
<tr><td style="background:#c81e1e; border-radius:8px;">
<a href="{{.TecitoURL}}" style="display:block; padding:8px 14px; color:#ffffff; font-weight:bold; font-size:13px; text-decoration:none;"><img src="{{.SiteURL}}/email/donar-tecito.png" width="22" height="22" alt="" style="vertical-align:middle; border:0; border-radius:5px; margin-right:8px;">Invitanos un tecito &middot; desde USD 1</a>
</td></tr>
{{end}}
</table>
</td></tr>
</table>
{{end}}

</td></tr>

<tr><td style="background:#f4f4f4; padding: 20px 28px; font-size:12px; line-height:1.6; color:#888888;">
Radar Global es un resumen automatizado de titulares: un modelo de lenguaje selecciona y resume lo que publicaron los medios consultados. No verifica los hechos; atribuye cada dato al medio que lo publicó. <a href="{{.SiteURL}}/metodologia" style="color:#c81e1e;">Cómo se hace</a><br><br>
Recibiste este correo porque estás suscrito al boletín de Radar Global.<br>
<a href="{{.UnsubscribeURL}}" style="color:#c81e1e;">Darme de baja</a>
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>
`

var emailTemplate = template.Must(template.New("newsletter").Funcs(template.FuncMap{
	"divPercent": func(n int) int {
		if n == 0 {
			return 100
		}
		return 100 / n
	},
}).Parse(emailTemplateSrc))

// RenderEmail arma el HTML final del correo. data.AbstractHTML y
// data.PopulatedRegions[].HTML ya vienen convertidos por
// MarkdownFragmentToHTML — el template solo compone el layout.
func RenderEmail(data EmailData) (string, error) {
	var buf bytes.Buffer
	if err := emailTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("newsletter: no se pudo renderizar el template: %w", err)
	}
	return buf.String(), nil
}
