package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"mobincus/incus"
)

func FormatContainerList(containers []incus.DockerContainer) string {
	var b strings.Builder
	b.WriteString("CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS   PORTS     NAMES\n")
	for _, c := range containers {
		id := c.ID
		if len(id) > 12 {
			id = id[:12]
		}
		status := c.Status
		names := strings.Join(c.Names, ", ")
		b.WriteString(fmt.Sprintf("%-14s %-9s %-9s %-8s %-8s %-9s %s\n", id, "", "", "", status, "", names))
	}
	return b.String()
}

func TemplateOutput(tmplStr string, data interface{}) (string, error) {
	funcMap := template.FuncMap{
		"json": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return fmt.Sprintf("<error: %v>", err)
			}
			return string(b)
		},
	}
	tmpl, err := template.New("").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parsing error: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
