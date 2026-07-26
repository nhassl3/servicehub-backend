package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

const (
	ResetPassword     = "reset_password.html"
	EmailConfirmation = "email_confirmation.html"
	AnyMessage        = "any.html"
)

// Render rendering new html message on email client
func Render(name string, data interface{}) (string, error) {
	if !strings.HasSuffix(name, ".html") {
		name = name + ".html"
	}
	t, err := template.ParseFiles(
		"pkg/mailer/templates/base.html",
		"pkg/mailer/templates/"+name,
	)
	if err != nil {
		return "", fmt.Errorf("renderer.Render: failed to parse template: %w", err)
	}

	var body bytes.Buffer
	if err = t.Execute(&body, data); err != nil {
		return "", fmt.Errorf("renderer.Render: failed to execute template: %w", err)
	}

	return body.String(), nil
}
