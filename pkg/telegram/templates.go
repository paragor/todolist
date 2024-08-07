package telegram

import (
	"bytes"
	"github.com/paragor/todo/pkg/telegram/telegramtemplates"
	"html/template"
	"strings"
	"time"
)

var templates *template.Template

func init() {
	functions := map[string]interface{}{
		"strings_contains": func(slice []string, item string) bool {
			for _, v := range slice {
				if v == item {
					return true
				}
			}
			return false
		},
		"join": strings.Join,
		"time_is_over": func(date *time.Time) bool {
			if date == nil {
				return false
			}
			return date.Before(time.Now())
		},
	}

	templates = template.New("").Funcs(functions)
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
