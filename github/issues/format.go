package issues

import (
	"bytes"
	"text/template"
)

func execTemplate(templateName, templateString string, ctx interface{}) string {
	// Prepare a buffer.
	var buffer bytes.Buffer

	// Parse the template string.
	t := template.Must(template.New(templateName).Parse(templateString))

	// Execute the template.
	// If we get an error here, something is seriously wrong.
	if err := t.Execute(&buffer, ctx); err != nil {
		panic(err)
	}

	// Return the buffer content.
	return buffer.String()
}
