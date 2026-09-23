package newsletter

import (
	"bytes"
	"fmt"
	"html/template"
)

// ConfirmSubject es el asunto del mail de doble opt-in.
const ConfirmSubject = "Confirmá tu suscripción a Radar Global"

type confirmData struct {
	SiteURL    string
	ConfirmURL string
}

// El mail de confirmación es deliberadamente austero: sin imágenes (muchos
// clientes las bloquean en remitentes nuevos) y con el link también en
// texto plano, por si el botón no se renderiza. Aclara qué hacer si no fue
// la persona quien se anotó — la respuesta correcta es no hacer nada.
const confirmTemplateSrc = `<!doctype html>
<html lang="es">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0; padding:24px 12px; background:#eeeeee; font-family: Georgia, 'Times New Roman', serif; color:#171717;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center">
<table role="presentation" width="560" cellpadding="0" cellspacing="0" style="background:#ffffff; max-width:560px;">
<tr><td style="background:#171717; color:#ffffff; padding:18px 28px; font-size:15px; font-weight:bold; letter-spacing:0.14em; text-transform:uppercase;">Radar Global</td></tr>
<tr><td style="background:#c81e1e; height:3px; font-size:1px; line-height:1px;">&nbsp;</td></tr>
<tr><td style="padding:28px; font-size:16px; line-height:1.55;">
<p style="margin:0 0 16px;">Alguien (probablemente vos) pidió recibir el resumen diario de <a href="{{.SiteURL}}" style="color:#c81e1e;">Radar Global</a> en esta dirección.</p>
<p style="margin:0 0 24px;">Para empezar a recibirlo, confirmá tu suscripción:</p>
<p style="margin:0 0 24px;"><a href="{{.ConfirmURL}}" style="display:inline-block; background:#c81e1e; color:#ffffff; text-decoration:none; font-weight:bold; padding:12px 22px;">Confirmar suscripción</a></p>
<p style="margin:0 0 8px; font-size:13px; color:#555555;">Si el botón no funciona, copiá este enlace en el navegador:</p>
<p style="margin:0 0 24px; font-size:13px; word-break:break-all;"><a href="{{.ConfirmURL}}" style="color:#555555;">{{.ConfirmURL}}</a></p>
<p style="margin:0; font-size:13px; color:#555555;">Si no fuiste vos, ignorá este mensaje: sin confirmar no vas a recibir nada y tu dirección se borra sola en 7 días.</p>
</td></tr>
</table>
</td></tr></table>
</body></html>`

var confirmTemplate = template.Must(template.New("confirm").Parse(confirmTemplateSrc))

// RenderConfirmEmail arma el HTML del mail de confirmación de alta.
func RenderConfirmEmail(siteURL, confirmURL string) (string, error) {
	var buf bytes.Buffer
	if err := confirmTemplate.Execute(&buf, confirmData{SiteURL: siteURL, ConfirmURL: confirmURL}); err != nil {
		return "", fmt.Errorf("newsletter: no se pudo renderizar el mail de confirmación: %w", err)
	}
	return buf.String(), nil
}
