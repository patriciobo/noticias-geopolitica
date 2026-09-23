package newsletter

import (
	"bytes"
	"fmt"
	"html/template"
)

// emailTemplate usa html/template (no text/template) porque bodyHTML y el
// link de unsubscribe se interpolan en contexto HTML — el auto-escapado de
// html/template es la defensa por default contra XSS acá. bodyHTML ya viene
// convertido a HTML por MarkdownFragmentToHTML, así que se marca como
// template.HTML para que no se re-escape (el escapado ya pasó en esa etapa).
// CSS inline: los clientes de correo no soportan <style> externo de forma
// confiable.
const emailTemplateSrc = `<!doctype html>
<html lang="es">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"></head>
<body style="margin:0; padding:0; background:#f4f4f4; font-family: Georgia, 'Times New Roman', serif; color:#1a1a1a;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f4f4f4; padding: 24px 0;">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="background:#ffffff; border-radius:8px; overflow:hidden;">
<tr><td style="padding: 24px 32px 8px 32px; border-bottom: 1px solid #eaeaea;">
<span style="font-size:13px; letter-spacing:0.04em; text-transform:uppercase; color:#666;">Noticias Internacionales</span>
</td></tr>
<tr><td style="padding: 24px 32px; font-size:16px; line-height:1.55;">
{{.BodyHTML}}
</td></tr>
<tr><td style="padding: 16px 32px 24px 32px; border-top: 1px solid #eaeaea; font-size:12px; color:#999;">
<a href="{{.UnsubscribeURL}}" style="color:#999;">Darme de baja</a>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>
`

var emailTemplate = template.Must(template.New("newsletter").Parse(emailTemplateSrc))

type emailTemplateData struct {
	BodyHTML       template.HTML
	UnsubscribeURL string
}

// RenderEmail arma el HTML final del correo a partir del fragmento ya
// convertido (bodyHTML) y la URL de baja específica del destinatario.
func RenderEmail(bodyHTML, unsubscribeURL string) (string, error) {
	var buf bytes.Buffer
	data := emailTemplateData{BodyHTML: template.HTML(bodyHTML), UnsubscribeURL: unsubscribeURL}
	if err := emailTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("newsletter: no se pudo renderizar el template: %w", err)
	}
	return buf.String(), nil
}
