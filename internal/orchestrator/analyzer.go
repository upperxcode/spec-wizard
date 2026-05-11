package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/adapters"
	"spec-wizard/internal/generator"
	"spec-wizard/internal/llm"
	"spec-wizard/internal/prompt"
	"spec-wizard/internal/registry"
)

// GetLLMClient cria um cliente configurado baseado no estado do projeto ou nas configurações globais
func (o *Orchestrator) GetLLMClient(config ProjectConfig) (*llm.LLMClient, error) {
	providerName := config.LLM.Provider
	baseURL := config.LLM.BaseURL
	apiKey := config.LLM.APIKey
	envKey := config.LLM.EnvAPIKey
	model := config.LLM.Model

	fmt.Printf("🔍 [Orchestrator] Resolvendo cliente LLM (Proj: %s, Model: %s)\n", providerName, model)

	// Fallback para Engine se não houver config completa no projeto
	if (providerName == "" || apiKey == "") && o.Engine != nil {
		fmt.Println("ℹ️ [Orchestrator] Configuração do projeto incompleta, buscando na Engine Global...")
		p, m, err := o.Engine.GetActiveModel()
		if err == nil {
			if providerName == "" {
				providerName = p.Name
			}
			if baseURL == "" {
				baseURL = p.APIURL
			}
			if apiKey == "" {
				apiKey = p.APIKey
			}
			envKey = p.EnvAPIKey
			if model == "" {
				model = m.Name
			}
			config.LLM.Sequential = p.Sequential
			config.LLM.UseCLI = p.UseCLI
		}
	}

	if providerName == "" {
		return nil, fmt.Errorf("nenhum provedor de IA configurado (global ou no projeto)")
	}

	fmt.Printf("✅ [Orchestrator] Cliente resolvido: %s (%s, UseCLI: %v)\n", providerName, model, config.LLM.UseCLI)

	provider, err := llm.NewProvider(providerName, baseURL, apiKey, envKey, config.LLM.Sequential, config.LLM.UseCLI, o.Plugins)
	if err != nil {
		return nil, err
	}
	return llm.NewLLMClient(provider, model, o.Tools), nil
}

// RequestAnchorFiles solicita ao Expert a geração dos documentos iniciais
func (o *Orchestrator) RequestAnchorFiles(language string, plugin *registry.ExpertPlugin) {
	client := adapters.NewExpertClient()
	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")

	o.InitializeProject(language, plugin)

	fmt.Println("📡 Solicitando geração de PRD e SPEC ao Expert...")
	resp, err := client.CallExpert(plugin, "GenerateSpec", map[string]string{
		"project_type": language,
	})
	if err != nil {
		fmt.Printf("❌ Erro ao obter especificações: %v\n", err)
		return
	}

	specData := resp["spec_content"].(string)
	prdData := resp["prd_content"].(string)

	os.WriteFile(filepath.Join(wizardPath, "SPEC.md"), []byte(specData), 0644)
	os.WriteFile(filepath.Join(wizardPath, "PRD.md"), []byte(prdData), 0644)
}

// OrchestrateIntelligence coordena a análise técnica via IA
func (o *Orchestrator) OrchestrateIntelligence(plugin *registry.ExpertPlugin, rawFileContent string) {
	client := adapters.NewExpertClient()
	factory := prompt.NewPromptFactory()

	technicalFacts, err := client.CallExpert(plugin, "AnalyzeCodebase", rawFileContent)
	if err != nil {
		fmt.Printf("❌ Erro na análise técnica: %v\n", err)
		return
	}

	finalPrompt := factory.GenerateAnchoringPrompt(plugin.Language, technicalFacts, rawFileContent)

	// TODO: Obter config real, por enquanto criamos uma vazia para disparar o fallback
	llmLocal, err := o.GetLLMClient(ProjectConfig{})
	if err != nil {
		fmt.Printf("❌ Erro ao obter cliente LLM: %v\n", err)
		return
	}
	aiResponse, err := llmLocal.Ask(finalPrompt)

	if err != nil {
		fmt.Printf("❌ Erro ao processar IA: %v\n", err)
		return
	}

	data := generator.AnchorData{
		ProjectName: technicalFacts["project_name"].(string),
		Objective:   aiResponse,
		TechStack:   make(map[string]string),
	}

	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")
	generator.SaveMarkdown(filepath.Join(wizardPath, "PRD.md"), generator.PrdTemplate, data)
}

// InterpretExistingProject analisa o diretório atual e sugere uma configuração via IA
func (o *Orchestrator) InterpretExistingProject(lang string, plugin *registry.ExpertPlugin, userLang string) (interface{}, error) {
	client := adapters.NewExpertClient()
	factory := prompt.NewPromptFactory()

	expertPayload := map[string]interface{}{
		"project_path": o.ProjectPath,
		"language":     lang,
	}

	fmt.Printf("📡 [Analyzer] Solicitando análise técnica ao Expert %s...\n", lang)
	technicalFacts, err := client.CallExpert(plugin, "AnalyzeCodebase", expertPayload)
	if err != nil {
		fmt.Printf("❌ [Analyzer] Erro no Expert Call: %v\n", err)
		return nil, fmt.Errorf("erro no plugin expert: %v", err)
	}

	var inferred map[string]interface{}
	if val, ok := technicalFacts["inferred_properties"]; ok {
		inferred = val.(map[string]interface{})
	} else {
		inferred = make(map[string]interface{})
	}

	pluginPatterns, _ := client.CallExpert(plugin, "GetPatterns", expertPayload)
	technicalFacts["plugin_patterns"] = pluginPatterns["patterns"]

	interpretPrompt := factory.GenerateInterpretationPrompt(lang, technicalFacts, userLang)

	llmLocal, err := o.GetLLMClient(ProjectConfig{UserLanguage: userLang})
	if err != nil {
		fmt.Printf("❌ [Analyzer] Erro ao obter cliente LLM: %v\n", err)
		return nil, err
	}

	fmt.Printf("📡 [Analyzer] Solicitando interpretação à IA (Modelo: %s)...\n", llmLocal.Model)
	aiResponse, err := llmLocal.Ask(interpretPrompt)
	if err != nil {
		fmt.Printf("❌ [Analyzer] Erro na resposta da IA: %v\n", err)
		return nil, fmt.Errorf("erro na comunicação com a IA: %v", err)
	}
	if aiResponse == "" {
		return nil, fmt.Errorf("a IA retornou uma resposta vazia")
	}

	cleaned := CleanJSONResponse(aiResponse)

	var aiSuggestion map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &aiSuggestion); err != nil {
		return nil, fmt.Errorf("IA retornou JSON inválido: %v | Raw: %s", err, cleaned)
	}

	finalResult := make(map[string]interface{})
	expertFields := []string{
		"projectName", "domain", "functionalRequirements",
		"nonFunctionalRequirements", "architecture", "patterns",
		"dataStrategy", "stateManagement", "customization", "apiContract",
	}

	for _, field := range expertFields {
		// Tenta camelCase e snake_case (para compatibilidade com Experts Go/Flutter)
		snakeField := toSnakeCase(field)
		if val, ok := inferred[field]; ok && !isEmpty(val) {
			finalResult[field] = val
		} else if val, ok := inferred[snakeField]; ok && !isEmpty(val) {
			finalResult[field] = val
		} else if val, ok := aiSuggestion[field]; ok && !isEmpty(val) {
			finalResult[field] = val
		}
	}

	// Se ainda não temos projectName, tentamos inferir do diretório ou metadata
	if isEmpty(finalResult["projectName"]) {
		if meta, ok := technicalFacts["metadata"].(map[string]interface{}); ok {
			if name, ok := meta["project_name"].(string); ok && !isEmpty(name) {
				finalResult["projectName"] = name
			}
		}
	}

	// Fallback final: Nome do diretório
	if isEmpty(finalResult["projectName"]) {
		finalResult["projectName"] = filepath.Base(o.ProjectPath)
	}

	for k, v := range aiSuggestion {
		if _, ok := finalResult[k]; !ok {
			finalResult[k] = v
		}
	}

	finalResult["metadata"] = technicalFacts
	return finalResult, nil
}
