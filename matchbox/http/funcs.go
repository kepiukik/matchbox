package http

import (
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

func funcMap() template.FuncMap {
	f := sprig.TxtFuncMap()
	delete(f, "env")
	delete(f, "expandenv")

	f["toYaml"] = toYAML

	return f
}

func toYAML(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}
