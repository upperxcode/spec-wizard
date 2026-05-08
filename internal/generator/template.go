package generator

import (
	"os"
	"text/template"
)

// Estrutura de dados para o PRD
type AnchorData struct {
	ProjectName string
	Objective   string
	Features    []string
	TechStack   map[string]string
}

const PrdTemplate = `# PRD - {{.ProjectName}}
## Objetivo
{{.Objective}}

## Funcionalidades Identificadas
{{range .Features}}- {{.}}
{{end}}
`

const SpecTemplate = `# SPEC Técnica - {{.ProjectName}}
## Arquitetura e Stack
- **Linguagem:** {{index .TechStack "language"}}
- **Framework:** {{index .TechStack "framework"}}
- **Estado:** {{index .TechStack "state_management"}}
`

func SaveMarkdown(path string, tmplStr string, data interface{}) error {
	tmpl, err := template.New("doc").Parse(tmplStr)
	if err != nil { return err }
	
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()
	
	return tmpl.Execute(f, data)
}