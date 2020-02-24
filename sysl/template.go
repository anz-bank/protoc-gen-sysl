package sysltemplate

import (
	"bufio"
	"bytes"
	"html/template"
)

const Type = `
!type aisgbaskdvndskfjgn {{.Name}}:
	{{range .Fields}} {{.Name}} <: {{.Type}} {{end}}

	
`

func Execute(tmpl string, obj interface{}) (string, error) {
	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	exampleTmpl := template.New(tmpl)
	if err := exampleTmpl.Execute(foo, obj); err != nil {
		return "", err
	}
	return b.String(), nil
}
