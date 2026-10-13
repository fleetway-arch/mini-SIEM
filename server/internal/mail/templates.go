package mail

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
	"time"
)

// layout is shared by every email. Values are inserted through html/template,
// which HTML-escapes them; colors match the app's dark-blue theme.
var htmlLayout = htmltemplate.Must(htmltemplate.New("layout").Parse(`<!doctype html>
<html lang="fr">
<body style="margin:0;padding:0;background:#0b1220;font-family:Inter,Segoe UI,Arial,sans-serif;color:#e2e8f0;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#0b1220;padding:32px 16px;">
    <tr><td align="center">
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#111a2e;border:1px solid #1e2a45;border-radius:10px;">
        <tr><td style="padding:24px 28px;border-bottom:1px solid #1e2a45;">
          <span style="display:inline-block;width:32px;height:32px;line-height:32px;border-radius:8px;background:#2563eb;color:#ffffff;text-align:center;font-weight:700;">S</span>
          <span style="margin-left:10px;font-size:16px;font-weight:700;color:#e2e8f0;vertical-align:middle;">Mini SIEM</span>
        </td></tr>
        <tr><td style="padding:28px;">
          <h1 style="margin:0 0 16px;font-size:20px;color:#e2e8f0;">{{.Title}}</h1>
          {{range .Paragraphs}}<p style="margin:0 0 14px;font-size:14px;line-height:1.6;color:#94a3b8;">{{.}}</p>{{end}}
          {{if .ButtonURL}}
          <p style="margin:24px 0;">
            <a href="{{.ButtonURL}}" style="display:inline-block;padding:12px 22px;background:#3b82f6;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:600;font-size:14px;">{{.ButtonText}}</a>
          </p>
          <p style="margin:0 0 14px;font-size:12px;line-height:1.6;color:#64748b;">Si le bouton ne fonctionne pas, copiez ce lien :<br><span style="color:#94a3b8;word-break:break-all;">{{.ButtonURL}}</span></p>
          {{end}}
          {{if .Footer}}<p style="margin:20px 0 0;font-size:12px;line-height:1.6;color:#64748b;">{{.Footer}}</p>{{end}}
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`))

var textLayout = texttemplate.Must(texttemplate.New("layout").Parse(`{{.Title}}

{{range .Paragraphs}}{{.}}

{{end}}{{if .ButtonURL}}{{.ButtonText}} :
{{.ButtonURL}}

{{end}}{{if .Footer}}{{.Footer}}
{{end}}--
Mini SIEM
`))

type content struct {
	Title      string
	Paragraphs []string
	ButtonText string
	ButtonURL  string
	Footer     string
}

func render(subject string, c content) (Message, error) {
	var html, text bytes.Buffer
	if err := htmlLayout.Execute(&html, c); err != nil {
		return Message{}, fmt.Errorf("render html email: %w", err)
	}
	if err := textLayout.Execute(&text, c); err != nil {
		return Message{}, fmt.Errorf("render text email: %w", err)
	}
	return Message{Subject: subject, Text: text.String(), HTML: html.String()}, nil
}

func PasswordResetEmail(to, link string) (Message, error) {
	message, err := render("Réinitialisation de votre mot de passe Mini SIEM", content{
		Title: "Réinitialiser votre mot de passe",
		Paragraphs: []string{
			"Nous avons reçu une demande de réinitialisation du mot de passe de votre compte Mini SIEM.",
			"Ce lien est valable 30 minutes et ne peut être utilisé qu'une seule fois.",
		},
		ButtonText: "Choisir un nouveau mot de passe",
		ButtonURL:  link,
		Footer:     "Si vous n'êtes pas à l'origine de cette demande, ignorez cet email : votre mot de passe actuel reste valable.",
	})
	message.To = to
	return message, err
}

func PasswordChangedEmail(to string, changedAt time.Time) (Message, error) {
	message, err := render("Votre mot de passe Mini SIEM a été modifié", content{
		Title: "Mot de passe modifié",
		Paragraphs: []string{
			"Le mot de passe de votre compte Mini SIEM a été modifié le " + changedAt.UTC().Format("02/01/2006 à 15:04") + " (UTC).",
			"Toutes vos sessions ouvertes ont été déconnectées.",
		},
		Footer: "Si vous n'êtes pas à l'origine de ce changement, contactez immédiatement l'administrateur de votre espace.",
	})
	message.To = to
	return message, err
}

type InvitationData struct {
	CompanyName string
	InviterName string
	Role        string
	Link        string
}

func InvitationEmail(to string, data InvitationData) (Message, error) {
	message, err := render("Invitation à rejoindre "+data.CompanyName+" sur Mini SIEM", content{
		Title: "Vous êtes invité sur Mini SIEM",
		Paragraphs: []string{
			fmt.Sprintf("%s vous invite à rejoindre l'espace %s avec le rôle %s.", data.InviterName, data.CompanyName, strings.ToLower(data.Role)),
			"Cliquez sur le bouton ci-dessous pour créer votre compte. Ce lien est personnel, valable 7 jours et utilisable une seule fois.",
		},
		ButtonText: "Accepter l'invitation",
		ButtonURL:  data.Link,
		Footer:     "Si vous n'attendiez pas cette invitation, vous pouvez ignorer cet email.",
	})
	message.To = to
	return message, err
}
