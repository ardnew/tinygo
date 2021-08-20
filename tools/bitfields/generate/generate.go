package generate

import (
	"log"
	"os"
	"text/template"
	"time"
)

const bitfieldsTemplate = `// Code generated {{.Now}} with {{.Generator}}; DO NOT EDIT.

{{- define "BitField" -}}
type BitField{{.}} interface {
	Get() uint{{.}}
	Set(v uint{{.}})
}
{{end}}

{{template "BitField" 8}}
{{template "BitField" 16}}
{{template "BitField" 32}}
{{template "BitField" 64}}
`

func Generate() {
	t := template.Must(template.New("bitfields").Parse(bitfieldsTemplate))
	err := t.Execute(os.Stdout, struct {
		Now       time.Time
		Generator string
	}{Now: time.Now(), Generator: "tinygo.org/tinygo/tools/bitfields"})
	if err != nil {
		log.Println("executing template:", err)
	}
}
