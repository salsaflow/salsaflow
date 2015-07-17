package issues

import (
	"io"
	"text/template"
)

func execTemplate(w io.Writer, templateName, templateString string, ctx interface{}) {
	// Parse the template string.
	t := template.Must(template.New(templateName).Parse(templateString))

	// Execute the template.
	// If we get an error here, something is seriously wrong.
	if err := t.Execute(w, ctx); err != nil {
		panic(err)
	}
}
