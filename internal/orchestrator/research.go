package orchestrator

import (
	"fmt"
	"os"
	"spec-wizard/internal/adapters"
	"spec-wizard/internal/generator"
	"spec-wizard/internal/prompt"
	"spec-wizard/internal/registry"

	"path/filepath"
)

func (o *Orchestrator) RunResearch(plugin *registry.ExpertPlugin) error {
	fmt.Println("🔍 Iniciando Fase de Research: Analisando código existente...")
	client := adapters.NewExpertClient()

	// 1. Ler o ficheiro principal para extrair o contexto técnico
	// No caso do Flutter, lemos o pubspec.yaml
	rawContent, err := os.ReadFile(filepath.Join(o.ProjectPath, "pubspec.yaml"))
	if err != nil {
		return err
	}

	// 2. Chamar o Expert para identificar a stack e funcionalidades
	resp, err := client.CallExpert(plugin, "AnalyzeCodebase", string(rawContent))
	if err != nil {
		return err
	}

	// 3. Prompt Factory: Criar o prompt para o PRD
	// "Baseado nestas libs detectadas: %v, descreva o propósito de negócio deste projeto"
	factory := prompt.NewPromptFactory()
	researchPrompt := factory.GenerateResearchPrompt(resp)

	// 4. Inteligência Centralizada: Chamar o LLM configurado
	llmLocal, err := o.GetLLMClient(ProjectConfig{})
	if err != nil {
		return err
	}
	prdContent, err := llmLocal.Ask(researchPrompt)

	if err != nil {
		return err
	}

	// 5. Salvar o PRD.md via Template
	data := generator.AnchorData{
		ProjectName: resp["project_name"].(string),
		Objective:   prdContent,
	}
	return generator.SaveMarkdown(filepath.Join(o.ProjectPath, ".spec-wizard/PRD.md"), generator.PrdTemplate, data)
}
