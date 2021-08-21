package generate

import (
	"io"
	"text/template"
	"time"

	"github.com/tinygo-org/tinygo/tools/bitfields/parse"
)

var FileHeader = template.Must(template.New("FILE-HEADER").Parse(
	`// Code generated {{.Datetime}} with "{{.Config.Generator}}"; DO NOT EDIT.`))

var PeriphType = template.Must(template.New("PERIPH-TYPE").
	Parse(`
{{range $periph := .Periph}}{{range $reg := $periph.Register}}{{range $ident, $field := $reg.Field}}
func (p *{{$periph.Ident}}) {{if and (eq 0 (len $field.Enum)) (eq $ident $reg.Ident)}}{{$ident}}{{else}}{{$reg.Ident}}_{{$ident}}{{end}}(){{if eq 1 $field.Len}}{{- printf " bool {\n" -}}{{else -}}{{- printf " uint%d {\n" $field.Register.Bits -}}{{end -}}
	{{"\t"}}return (p.{{$field.Register.Ident}}.Get() >> {{$field.Pos}}) & {{$field.Mask}}{{if eq 1 $field.Len}} == 1{{end}}
}

func (p *{{$periph.Ident}}) SET_{{if and (eq 0 (len $field.Enum)) (eq $ident $reg.Ident)}}{{$ident}}{{else}}{{$reg.Ident}}_{{$ident}}{{end}}{{if eq 1 $field.Len}}{{- printf "(b bool) {\n" -}}{{else -}}{{- printf "(v uint%d) {\n" $field.Register.Bits -}}{{end -}}
	{{"\t"}}{{if eq 1 $field.Len}}v := uint{{$field.Register.Bits}}(0)
	if b {
		v = 1
	}
	{{end}}p.{{$field.Register.Ident}}.ReplaceBits(v, {{$field.Mask}}, {{$field.Pos}})
}
{{end}}{{end}}{{end}}
`))

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
