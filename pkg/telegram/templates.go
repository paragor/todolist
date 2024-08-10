package telegram

import (
	"bytes"
	"github.com/paragor/todo/pkg/telegram/telegramtemplates"
	"github.com/paragor/todo/pkg/templatesutils"
	"html/template"
)

var templates *template.Template

func init() {
	templates = template.New("").Funcs(templatesutils.GetFunctions())
	templates = must(templates.ParseFS(telegramtemplates.Messages, "messages/*.html"))
}

func renderTemplate(template string, data any) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := templates.ExecuteTemplate(buffer, template, data); err != nil {
		return "", err
	}
	return buffer.String(), nil

}
func must[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}
