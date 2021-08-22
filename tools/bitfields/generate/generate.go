package generate

import (
	"io"
	"text/template"
	"time"

	"github.com/tinygo-org/tinygo/tools/bitfields/parse"
)

var FileHeader = template.Must(template.New("FILE-HEADER").Parse(
	`// Code generated {{.Datetime}} with "{{.Config.Generator}}"; DO NOT EDIT.

import (
	"runtime/volatile"
{{range .Config.Parser.Package}}{{printf "\t//%q // Uncomment if embedding peripherals\n" .Json.ImportPath}}{{end}})`))

var PeriphType = template.Must(template.New("PERIPH-TYPE").
	Parse(`
{{range .Periph}}
type {{.Ident}} struct {
	//*{{.Package.Json.Name}}.{{.Ident}} // Remember to uncomment providing package's import statement
	{{- range .Register}}{{if .IsBlank}}{{- printf "\n\t_ [%d]byte" .Size}}{{else}}{{- printf "\n\t%s %sType" .Ident .Prefix}}{{end}}{{end}}
}{{end}}
type (
{{- range .Periph}}{{- range .Register}}{{if (not .IsBlank)}}{{- printf "\n\t%sType volatile.Register%d" .Prefix .Bits}}{{end}}{{end}}{{end}}
)
{{range $periph := .Periph}}{{range $reg := $periph.Register}}{{if (not $reg.IsBlank)}}{{range $ident, $field := $reg.Field}}
func (r *{{$reg.Prefix}}Type) {{$ident}}(){{if eq 1 $field.Len}}{{- printf " bool {\n" -}}{{else -}}{{- printf " uint%d {\n" $field.Register.Bits -}}{{end -}}
	{{"\t"}}return (r.Get() >> {{$field.Pos}}) & {{$field.Mask}}{{if eq 1 $field.Len}} == 1{{end}}
}
func (r *{{$reg.Prefix}}Type) Set{{$ident}}{{if eq 1 $field.Len}}{{- printf "(b bool) {\n" -}}{{else -}}{{- printf "(v uint%d) {\n" $field.Register.Bits -}}{{end -}}
	{{"\t"}}{{if eq 1 $field.Len}}v := uint{{$field.Register.Bits}}(0)
	if b {
		v = 1
	}
	{{end}}r.ReplaceBits(v, {{$field.Mask}}, {{$field.Pos}})
}{{end}}
{{end}}{{end}}{{end}}
`))

//{{range .Register}}{{- printf "\t%s %sType\n" .Ident .Prefix -}}{{end}}}

type Generator struct {
	Config   Config
	Datetime time.Time
}

type Config struct {
	Generator string
	Parser    *parse.Parser
}

func New(c Config) *Generator {
	return &Generator{
		Config:   c,
		Datetime: time.Now(),
	}
}

func (g *Generator) Run(w io.Writer) error {

	if err := FileHeader.Execute(w, g); err != nil {
		return err
	}

	for _, p := range g.Config.Parser.Package {
		if err := PeriphType.Execute(w, p); err != nil {
			return err
		}

	}

	return nil
}
