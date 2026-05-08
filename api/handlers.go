package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"spec-wizard/config"
	"spec-wizard/internal/adapters"
	"spec-wizard/internal/llm"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/orchestrator"
	"spec-wizard/internal/patterns"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/workspace"
	"strings"
	"time"
)

const ConfigPath = "/home/john/.spec-wizard"
const ProjectsFile = "projects.json"

type ProjectEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type APIHandler struct {
	Plugins     []*registry.ExpertPlugin
	PatternRepo *patterns.PatternRepository
	Engine      *config.Engine
	Workspace   *workspace.Workspace
}

// GetLLMClient cria um cliente configurado baseado nas configurações globais ou do projeto
func (h *APIHandler) GetLLMClient(projectCfg orchestrator.ProjectConfig) (*llm.LLMClient, error) {
	providerName := projectCfg.LLM.Provider
	baseURL := projectCfg.LLM.BaseURL
	apiKey := projectCfg.LLM.APIKey
	envKey := projectCfg.LLM.EnvAPIKey
	model := projectCfg.LLM.Model

	fmt.Printf("🔍 [API] Resolvendo cliente LLM (Proj: %s, Model: %s)\n", providerName, model)

	// Fallback para Engine se não houver config específica ou se estiver incompleta
	if (providerName == "" || apiKey == "") && h.Engine != nil {
		fmt.Println("ℹ️ [API] Configuração do projeto incompleta, buscando na Engine Global...")
		p, m, err := h.Engine.GetActiveModel()
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
		}
	}

	if providerName == "" {
		return nil, fmt.Errorf("nenhum provedor de IA configurado (global ou no projeto)")
	}

	provider := llm.NewProvider(providerName, baseURL, apiKey, envKey)
	return llm.NewLLMClient(provider, model, nil), nil
}

// GET /api/languages - Lista as linguagens com MCP disponível
func (h *APIHandler) GetLanguages(w http.ResponseWriter, r *http.Request) {
	languages := make([]string, 0)
	for _, p := range h.Plugins {
		if p.Language != "" {
			languages = append(languages, p.Language)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

// GET /api/patterns/{lang} - Retorna padrões imperativos para a linguagem (Híbrido: Global + Plugin)
func (h *APIHandler) GetPatterns(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	lang := parts[len(parts)-1]

	// Mapa para unificação por ID (evita duplicatas e permite sobrescrita)
	unifiedPatterns := make(map[string]patterns.Pattern)

	// 1. Carrega padrões base do repositório
	basePatterns := h.PatternRepo.Store[strings.ToLower(lang)]
	for _, p := range basePatterns {
		unifiedPatterns[p.ID] = p
	}

	// 2. Tenta buscar padrões dinâmicos do Expert
	expert, err := registry.FindExpertByLanguage(lang, h.Plugins)
	if err == nil {
		if err := registry.GlobalManager.EnsureExpertRunning(expert); err == nil {
			client := adapters.NewExpertClient()

			// A) Padrões via GetPatterns do Expert
			resp, err := client.CallExpert(expert, "GetPatterns", map[string]interface{}{"language": lang})
			if err == nil {
				if pluginPatternsRaw, ok := resp["patterns"]; ok {
					var pluginPatterns []patterns.Pattern
					b, _ := json.Marshal(pluginPatternsRaw)
					json.Unmarshal(b, &pluginPatterns)

					for _, p := range pluginPatterns {
						p.Scope = "language"
						unifiedPatterns[p.ID] = p // Sobrescreve global se ID colidir
					}
				}
			}

			// B) Opções estruturadas (arquiteturas, estados, estratégias)
			options, err := client.GetOptions(expert)
			if err == nil {
				mapOptions := func(key string, category string) {
					if list, ok := options[key].([]interface{}); ok {
						for _, itemRaw := range list {
							item := itemRaw.(map[string]interface{})
							id, _ := item["id"].(string)
							name, _ := item["name"].(string)
							desc, _ := item["description"].(string)

							unifiedPatterns[id] = patterns.Pattern{
								ID:          id,
								Name:        name,
								Category:    category,
								Scope:       "language",
								Description: desc,
							}
						}
					}
				}

				mapOptions("architectures", "Architecture")
				mapOptions("state_managements", "StateManagement")
				mapOptions("data_strategies", "DataStrategy")
			}
		}
	}

	// Converter mapa de volta para slice
	allPatterns := make([]patterns.Pattern, 0, len(unifiedPatterns))
	for _, p := range unifiedPatterns {
		allPatterns = append(allPatterns, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allPatterns)
}

// POST /api/initialize - Ancora o projeto com Caminho + Linguagem + Padrões
func (h *APIHandler) InitializeProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path                      string   `json:"path"`
		ProjectName               string   `json:"projectName"`
		Language                  string   `json:"language"`
		Patterns                  []string `json:"patterns"`
		Instructions              string   `json:"instructions"`
		Domain                    string   `json:"domain"`
		FunctionalRequirements    string   `json:"functionalRequirements"`
		NonFunctionalRequirements string   `json:"nonFunctionalRequirements"`
		DataStrategy              string   `json:"dataStrategy"`
		StateManagement           string   `json:"stateManagement"`
		ApiContract               string   `json:"apiContract"`
		Customization             string   `json:"customization"`
		Architecture              string   `json:"architecture"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	// 1. Valida o Expert
	expert, err := registry.FindExpertByLanguage(req.Language, h.Plugins)
	if err != nil {
		sendJSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	// 2. Orquestra a criação física da ancoragem
	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	config := orchestrator.ProjectConfig{
		Language:                  req.Language,
		Patterns:                  req.Patterns,
		Instructions:              req.Instructions,
		ProjectName:               req.ProjectName,
		Domain:                    req.Domain,
		FunctionalRequirements:    req.FunctionalRequirements,
		NonFunctionalRequirements: req.NonFunctionalRequirements,
		DataStrategy:              req.DataStrategy,
		StateManagement:           req.StateManagement,
		ApiContract:               req.ApiContract,
		Customization:             req.Customization,
		Architecture:              req.Architecture,
	}

	if err := core.InitializeWithLanguage(config, h.PatternRepo, expert); err != nil {
		sendJSONError(w, "Falha na inicialização: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Projeto ancorado com sucesso!"))
}

// POST /api/project/save-roadmap - Salva as edições manuais do roadmap
func (h *APIHandler) SaveRoadmap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string                      `json:"path"`
		Roadmap orchestrator.ProjectRoadmap `json:"roadmap"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	if err := core.SaveRoadmap(req.Roadmap); err != nil {
		sendJSONError(w, "Falha ao salvar roadmap: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Roadmap salvo com sucesso!"))
}

// POST /api/roadmap - Gera o plano de ação (sprints.json) baseado no design
func (h *APIHandler) GenerateRoadmap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path                   string   `json:"path"`
		Language               string   `json:"language"`
		Patterns               []string `json:"patterns"`
		Architecture           string   `json:"architecture"`
		DataStrategy           string   `json:"dataStrategy"`
		StateManagement        string   `json:"stateManagement"`
		ApiContract            string   `json:"apiContract"`
		Customization          string   `json:"customization"`
		AdditionalInstructions string   `json:"additionalInstructions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	// 1. Instancia o Orquestrador para o caminho fornecido
	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)

	// 2. Monta o objeto de configuração para a geração
	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en" // Fallback
	}

	config := orchestrator.ProjectConfig{
		Language:        req.Language,
		Patterns:        req.Patterns,
		Architecture:    req.Architecture,
		DataStrategy:    req.DataStrategy,
		StateManagement: req.StateManagement,
		ApiContract:     req.ApiContract,
		Customization:   req.Customization,
		Instructions:    req.AdditionalInstructions,
		UserLanguage:    userLang,
	}

	// 3. Chama a lógica de geração do Roadmap (que usa a Prompt Factory e LM Studio)
	roadmap, err := core.GenerateInitialRoadmap(config)
	if err != nil {
		sendJSONError(w, "Falha ao gerar roadmap: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Retorna o JSON do Roadmap para o cliente visualizar no Dashboard
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roadmap)
}

// POST /api/execute-task - Executa uma tarefa específica via Clean Window
func (h *APIHandler) ExecuteTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectPath string                  `json:"project_path"`
		SprintID    orchestrator.FlexibleID `json:"sprint_id"`
		TaskID      orchestrator.FlexibleID `json:"task_id"`
		Task        orchestrator.Task       `json:"task"`
		Sprint      orchestrator.Sprint     `json:"sprint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Instancia o Orquestrador para recuperar o estado (linguagem)
	core := orchestrator.NewOrchestrator(req.ProjectPath, h.Plugins, h.Engine)
	state, err := core.CheckInitialState()
	if err != nil || state.Language == "" {
		sendJSONError(w, "Projeto não inicializado ou linguagem não encontrada", http.StatusBadRequest)
		return
	}

	// 2. Instancia o Context Assembler
	assembler := orchestrator.NewContextAssembler(req.ProjectPath)

	// 3. Monta a janela limpa de contexto para a tarefa
	taskContext, err := assembler.AssembleTaskContext(req.Task, req.Sprint, h.Plugins)
	if err != nil {
		sendJSONError(w, "Falha ao montar contexto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Executa a tarefa via Feedback Loop (IA + Sensores + Auto-correção)
	llmClient, err := h.GetLLMClient(orchestrator.ProjectConfig{})
	if err != nil {
		sendJSONError(w, "Erro ao obter cliente LLM: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en"
	}

	loop := orchestrator.NewFeedbackLoop(req.ProjectPath, state.Language, userLang, llmClient)

	fmt.Printf("🔄 Iniciando execução da tarefa %v via Feedback Loop...\n", req.TaskID)
	
	// Timeout de 180s (3 minutos) para a execução completa (IA + Ferramentas + Verificação)
	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	attempt, err := loop.ExecuteTaskWithFeedback(ctx, taskContext, req.Task, req.Sprint)

	// 5. Mesmo que falhe após as tentativas, salvamos o log
	if attempt != nil {
		loop.SaveExecutionLog(req.ProjectPath, fmt.Sprintf("sprint-%v-task-%v", req.SprintID, req.TaskID), attempt)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error("❌ Timeout na execução da tarefa (180s excedidos)", err)
			sendJSONError(w, "A execução da tarefa demorou demais (timeout de 3min). Verifique se o modelo selecionado está disponível ou tente um modelo mais rápido.", http.StatusGatewayTimeout)
			return
		}
		sendJSONError(w, "Falha na execução (após auto-correção): "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 6. Se sucesso total, marcamos como concluída no roadmap
	core.UpdateTaskStatus(req.SprintID, req.TaskID, "completed")

	// 6. Retorna a resposta da IA (validada)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"task_id":           fmt.Sprintf("sprint-%v-task-%v", req.SprintID, req.TaskID),
		"ai_response":       attempt.AIResponse,
		"validation_passed": attempt.ValidationPassed,
		"attempts":          attempt.AttemptNumber,
		"diff":              attempt.Diff,
	})
}

// POST /api/project/generate-prompt
func (h *APIHandler) GenerateTaskPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectPath string              `json:"project_path"`
		Task        orchestrator.Task   `json:"task"`
		Sprint      orchestrator.Sprint `json:"sprint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	// 1. Instancia o Context Assembler
	assembler := orchestrator.NewContextAssembler(req.ProjectPath)

	// 2. Monta o contexto da tarefa
	taskContext, err := assembler.AssembleTaskContext(req.Task, req.Sprint, h.Plugins)
	if err != nil {
		sendJSONError(w, "Falha ao montar contexto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Constrói o prompt final
	prompt := assembler.BuildPromptForTask(taskContext)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"prompt": prompt,
	})
}

// GET /api/project/status?path={path}
func (h *APIHandler) GetProjectStatus(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSONError(w, "Caminho é obrigatório", http.StatusBadRequest)
		return
	}

	core := orchestrator.NewOrchestrator(path, h.Plugins, h.Engine)
	state, err := core.CheckInitialState()
	if err != nil {
		sendJSONError(w, "Erro ao verificar estado: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// POST /api/project/interpret
func (h *APIHandler) InterpretProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		Language string `json:"language"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	// 1. Acha o expert
	expert, err := registry.FindExpertByLanguage(req.Language, h.Plugins)
	if err != nil {
		sendJSONError(w, "Especialista não encontrado para "+req.Language, http.StatusBadRequest)
		return
	}

	// 2. Garante que o expert esteja rodando
	if err := registry.GlobalManager.EnsureExpertRunning(expert); err != nil {
		sendJSONError(w, "Falha ao iniciar expert: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Orquestra interpretação
	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en-US"
	}

	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	suggestion, err := core.InterpretExistingProject(req.Language, expert, userLang)
	if err != nil {
		sendJSONError(w, "Falha na interpretação: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Registra o projeto na lista global
	projectName := filepath.Base(req.Path)
	h.saveProjectEntry(projectName, req.Path)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestion)
}

// GET /api/llm/status
func (h *APIHandler) CheckLLMStatus(w http.ResponseWriter, r *http.Request) {
	status := "online"
	provider := "unknown"
	model := "none"
	label := "Nenhum"

	if h.Engine != nil {
		p, m, err := h.Engine.GetActiveModel()
		if err == nil {
			provider = p.Name
			model = m.Name
			label = m.Label
			if err := h.Engine.PingProvider(p.Name); err != nil {
				status = "offline"
			}
		} else {
			status = "offline"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   status,
		"provider": provider,
		"model":    model,
		"label":    label,
	})
}

// GET /api/llm/config - Retorna a configuração global de provedores
func (h *APIHandler) GetLLMConfig(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Engine.GetConfig())
}

// POST /api/llm/active - Define o modelo ativo
func (h *APIHandler) SetActiveModel(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	var req struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if err := h.Engine.SetActiveModel(req.Provider, req.Model); err != nil {
		sendJSONError(w, "Erro ao definir modelo ativo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Modelo ativo atualizado com sucesso"))
}

// POST /api/llm/provider/status - Ativa ou desativa um provedor
func (h *APIHandler) SetProviderStatus(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if err := h.Engine.SetProviderStatus(req.Name, req.Enabled); err != nil {
		sendJSONError(w, "Erro ao atualizar status do provedor: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status do provedor atualizado com sucesso"))
}

// POST /api/llm/reload - Força o recarregamento do config do disco
func (h *APIHandler) ReloadLLMConfig(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	if err := h.Engine.ReloadConfig(); err != nil {
		sendJSONError(w, "Erro ao recarregar configuração: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Engine.GetConfig())
}

// POST /api/project/infer-purpose - Realiza engenharia reversa do propósito via IA
func (h *APIHandler) InferPurpose(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string                 `json:"path"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	metadataJSON, _ := json.MarshalIndent(req.Metadata, "", "  ")

	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en-US"
	}

	langInstruction := "6. RESPOND EVERYTHING IN ENGLISH."
	if userLang != "en-US" {
		langInstruction = fmt.Sprintf("6. RESPOND EVERYTHING IN %s. (ALL CONTENT MUST BE IN %s)", strings.ToUpper(userLang), strings.ToUpper(userLang))
	}

	prompt := fmt.Sprintf(`You are a Senior Product Owner and Software Architect. Your task is to perform reverse purpose engineering on a project located at: %s.
 
Receive the following metadata extracted by the Micro-Expert:
%s
 
Based on dependencies, folder structure, and filenames, ignore technical implementation for a moment and strictly respond to the following points to compose the PRD (Product Requirement Document):
 
REAL PROBLEM (THE "WHY"):
Analyze the pain this software tries to heal. Why would someone spend time and money building this? Focus on business value and end-user needs.
 
DOMAIN DEFINITION:
What is the area of operation (e.g., Logistics, Fintech, E-commerce)? Who are the main actors (users) interacting with this system?
 
SCOPE AND BOUNDARIES:
What does this system aim to solve now and what is clearly not its responsibility (out of scope)?
 
RESPONSE RULES (AUTHORITATIVE PERSONA):
1. Use executive, direct, and professional language. You are the Product Owner and the authority on the project.
2. DO NOT use uncertain terms like "seems that", "possibly", "probably", or "might be". State what the system is based on evidence.
3. Do not respond as if in a chat. Respond DIRECTLY with the PRD report content.
4. If technical context is insufficient for a categorical statement, use technical facts to describe the system's capacity rather than assuming its purpose.
5. Respond in clean, structured Markdown format.
%s
 
Start your analysis:`, req.Path, string(metadataJSON), langInstruction)

	llmClient, err := h.GetLLMClient(orchestrator.ProjectConfig{UserLanguage: userLang})
	if err != nil {
		sendJSONError(w, "Erro ao obter cliente LLM: "+err.Error(), http.StatusInternalServerError)
		return
	}
	response, err := llmClient.Ask(prompt)

	if err != nil {
		sendJSONError(w, "Falha ao consultar IA: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"analysis": response})
}

// POST /api/project/save-spec - Salva a documentação .agents e o config raiz
func (h *APIHandler) SaveProjectSpec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string                     `json:"path"`
		Config  orchestrator.ProjectConfig `json:"config"`
		Roadmap interface{}                `json:"roadmap"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en-US"
	}
	req.Config.UserLanguage = userLang

	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)

	// Auto-tradução de campos críticos para Inglês (Window Cleaner Policy)
	llmClient, err := h.GetLLMClient(req.Config)
	if err == nil {
		req.Config.Domain, _ = llmClient.TranslateToEnglish(req.Config.Domain)
		req.Config.FunctionalRequirements, _ = llmClient.TranslateToEnglish(req.Config.FunctionalRequirements)
		req.Config.NonFunctionalRequirements, _ = llmClient.TranslateToEnglish(req.Config.NonFunctionalRequirements)
		req.Config.Instructions, _ = llmClient.TranslateToEnglish(req.Config.Instructions)
		req.Config.Customization, _ = llmClient.TranslateToEnglish(req.Config.Customization)
	}

	// Injeta o roadmap no objeto de config para centralização
	req.Config.Roadmap = req.Roadmap

	err = core.SaveAgentsDocs(req.Config, h.PatternRepo)

	if err != nil {
		sendJSONError(w, "Erro ao salvar documentação: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Registra/Atualiza o projeto na lista global
	projectName := req.Config.ProjectName
	if projectName == "" {
		projectName = filepath.Base(req.Path)
	}
	h.saveProjectEntry(projectName, req.Path)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Documentação .spec-wizard/ atualizada com sucesso!"))
}

// GET /api/projects - Lista todos os projetos salvos (Global)
func (h *APIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.loadProjects()
	if err != nil {
		json.NewEncoder(w).Encode([]ProjectEntry{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// POST /api/workspace/projects - Adiciona um projeto à lista
func (h *APIHandler) AddProject(w http.ResponseWriter, r *http.Request) {
	var entry workspace.ProjectEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if entry.Name == "" || entry.Path == "" {
		sendJSONError(w, "Nome e Path são obrigatórios", http.StatusBadRequest)
		return
	}

	if err := h.Workspace.AddProject(entry.Name, entry.Path); err != nil {
		sendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Projeto adicionado com sucesso!"))
}

// DELETE /api/workspace/projects - Remove um projeto da lista
func (h *APIHandler) RemoveProject(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSONError(w, "Path é obrigatório", http.StatusBadRequest)
		return
	}

	if err := h.Workspace.RemoveProject(path); err != nil {
		sendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Projeto removido com sucesso!"))
}

func (h *APIHandler) loadProjects() ([]ProjectEntry, error) {
	fullPath := filepath.Join(ConfigPath, ProjectsFile)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var projects []ProjectEntry
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (h *APIHandler) saveProjectEntry(name, path string) error {
	projects, _ := h.loadProjects()

	// Evitar duplicados
	for _, p := range projects {
		if p.Path == path {
			return nil
		}
	}

	projects = append(projects, ProjectEntry{Name: name, Path: path})

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		os.MkdirAll(ConfigPath, 0755)
	}

	data, _ := json.MarshalIndent(projects, "", "  ")
	return os.WriteFile(filepath.Join(ConfigPath, ProjectsFile), data, 0644)
}

// GET /api/project/harness-prompt?path={path}&question={question}
func (h *APIHandler) GetHarnessPrompt(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	question := r.URL.Query().Get("question")
	if path == "" {
		sendJSONError(w, "Caminho é obrigatório", http.StatusBadRequest)
		return
	}

	core := orchestrator.NewOrchestrator(path, h.Plugins, h.Engine)
	// Configura o idioma do usuário no orquestrador se necessário (ex: via ProjectConfig)
	prompt, err := core.GenerateHarnessPrompt(question)
	if err != nil {
		sendJSONError(w, "Erro ao gerar prompt: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(prompt))
}

// POST /api/project/generate-roadmap-from-code
func (h *APIHandler) GenerateRoadmapFromCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path   string                     `json:"path"`
		Config orchestrator.ProjectConfig `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	userLang := r.Header.Get("X-Language")
	req.Config.UserLanguage = userLang
	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	roadmap, err := core.UpdateRoadmapFromCode(req.Config)
	if err != nil {
		fmt.Printf("❌ [API] Erro ao gerar roadmap: %v\n", err)
		sendJSONError(w, "Falha ao gerar roadmap via código: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"roadmap":            roadmap,
		"roadmapLastUpdated": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// POST /api/project/roadmap-prompt
func (h *APIHandler) GetRoadmapPrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path   string                     `json:"path"`
		Config orchestrator.ProjectConfig `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	userLang := r.Header.Get("X-Language")
	req.Config.UserLanguage = userLang
	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	p := core.PreviewRoadmapPrompt(req.Config)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(p))
}

// GET/DELETE /api/knowledge?path={path}
func (h *APIHandler) GetKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		h.DeleteKnowledge(w, r)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSONError(w, "Caminho é obrigatório", http.StatusBadRequest)
		return
	}

	// Tenta carregar o config.json para pegar a base de conhecimento atualizada
	configPath := filepath.Join(path, ".spec-wizard", "config.json")
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg orchestrator.ProjectConfig
		json.Unmarshal(data, &cfg)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg.KnowledgeBase)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]orchestrator.KnowledgeSource{})
}

// POST /api/knowledge/upload
func (h *APIHandler) UploadKnowledge(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10MB limit

	projectPath := r.FormValue("path")
	file, handler, err := r.FormFile("file")
	if err != nil {
		sendJSONError(w, "Erro ao recuperar arquivo: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	km := orchestrator.NewKnowledgeManager(projectPath)
	source, err := km.IngestFile(handler.Filename, file)
	if err != nil {
		sendJSONError(w, "Erro ao processar arquivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Atualiza o config.json
	h.addKnowledgeToConfig(projectPath, *source)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source)
}

// POST /api/knowledge/link
func (h *APIHandler) AddKnowledgeLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	km := orchestrator.NewKnowledgeManager(req.Path)
	source, err := km.IngestLink(req.URL)
	if err != nil {
		sendJSONError(w, "Erro ao processar link: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.addKnowledgeToConfig(req.Path, *source)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(source)
}

func (h *APIHandler) addKnowledgeToConfig(projectPath string, source orchestrator.KnowledgeSource) {
	configPath := filepath.Join(projectPath, ".spec-wizard", "config.json")
	data, _ := os.ReadFile(configPath)
	var cfg orchestrator.ProjectConfig
	json.Unmarshal(data, &cfg)

	cfg.KnowledgeBase = append(cfg.KnowledgeBase, source)

	newData, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, newData, 0644)
}

// DELETE /api/knowledge?path={path}&id={id}
func (h *APIHandler) DeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	id := r.URL.Query().Get("id")

	if path == "" || id == "" {
		sendJSONError(w, "Path e ID são obrigatórios", http.StatusBadRequest)
		return
	}

	configPath := filepath.Join(path, ".spec-wizard", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		sendJSONError(w, "Config não encontrado", http.StatusNotFound)
		return
	}

	var cfg orchestrator.ProjectConfig
	json.Unmarshal(data, &cfg)

	var newKB []orchestrator.KnowledgeSource
	for _, s := range cfg.KnowledgeBase {
		if s.ID != id {
			newKB = append(newKB, s)
		} else {
			// Se for arquivo, remove o arquivo físico também
			if s.Type == "file" && s.Path != "" {
				os.Remove(s.Path)
				// Também remove o processado se existir
				processedPath := filepath.Join(path, ".spec-wizard", "knowledge", "processed", fmt.Sprintf("%s.md", s.ID))
				os.Remove(processedPath)
			}
		}
	}
	cfg.KnowledgeBase = newKB

	newData, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(configPath, newData, 0644)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Fonte removida com sucesso"))
}

// DELETE /api/project
func (h *APIHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSONError(w, "Caminho do projeto é obrigatório", http.StatusBadRequest)
		return
	}

	core := orchestrator.NewOrchestrator(path, h.Plugins, h.Engine)
	if err := core.DeleteProjectAnchor(); err != nil {
		sendJSONError(w, "Falha ao remover âncora: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Projeto removido (âncora deletada)"))
}

// POST /api/project/rename
func (h *APIHandler) RenameProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.Path == "" || req.NewName == "" {
		sendJSONError(w, "Caminho e Novo Nome são obrigatórios", http.StatusBadRequest)
		return
	}

	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	if err := core.RenameProject(req.NewName); err != nil {
		sendJSONError(w, "Falha ao renomear: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Projeto renomeado com sucesso"))
}

// POST /api/project/audit-task
func (h *APIHandler) AuditTaskStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectPath string                  `json:"project_path"`
		SprintID    orchestrator.FlexibleID `json:"sprint_id"`
		TaskID      orchestrator.FlexibleID `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	// 1. Instancia o Orquestrador
	absPath, _ := filepath.Abs(req.ProjectPath)
	core := orchestrator.NewOrchestrator(absPath, h.Plugins, h.Engine)

	// 2. Busca o estado do projeto para pegar a configuração de LLM
	state, err := core.CheckInitialState()
	if err != nil {
		sendJSONError(w, "Projeto não encontrado", http.StatusNotFound)
		return
	}

	// 3. Carrega o ProjectConfig do arquivo para garantir contexto completo
	configPath := filepath.Join(req.ProjectPath, ".spec-wizard", "config.json")
	data, _ := os.ReadFile(configPath)
	var config orchestrator.ProjectConfig
	json.Unmarshal(data, &config)

	// Injeta a linguagem se não estiver no config
	if config.Language == "" {
		config.Language = state.Language
	}

	// 4. Executa a auditoria com timeout para evitar travamentos infinitos
	logger.Info("🔍 Iniciando auditoria via Orquestrador", "sprint", req.SprintID, "task", req.TaskID)

	// Timeout de 180s (3 minutos) para a auditoria
	ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
	defer cancel()

	result, err := core.AuditTask(ctx, req.SprintID, req.TaskID, config)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error("❌ Timeout na auditoria (180s excedidos)", err)
			sendJSONError(w, "A auditoria demorou demais para responder (timeout de 3min). Verifique se o modelo selecionado está disponível ou tente um modelo mais rápido.", http.StatusGatewayTimeout)
			return
		}
		logger.Error("❌ Erro na auditoria", err)

		msg := fmt.Sprintf("Falha na auditoria: %v", err)
		// Se o erro indicar resposta vazia ou problema de parsing, sugerimos a troca de modelo
		if strings.Contains(err.Error(), "vazia") || strings.Contains(err.Error(), "JSON") {
			_, modelCfg, _ := h.Engine.GetActiveModel()
			msg = fmt.Sprintf("%s. Dica: O modelo atual (%s) pode não ter capacidade suficiente para esta análise. Tente mudar para um modelo mais robusto (ex: Gemini 2.0 ou GPT-4) nas configurações globais.", msg, modelCfg.Name)
		}

		sendJSONError(w, msg, http.StatusInternalServerError)
		return
	}
	logger.Info("✅ Auditoria finalizada com sucesso")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"audit":  result,
	})
}

// sendJSONError envia uma resposta de erro em formato JSON
// POST /api/project/config/exclude - Atualiza padrões de exclusão
func (h *APIHandler) UpdateExcludePatterns(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path            string   `json:"path"`
		ExcludePatterns []string `json:"excludePatterns"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	configPath := filepath.Join(req.Path, ".spec-wizard", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		sendJSONError(w, "Projeto não inicializado ou config.json ausente", http.StatusNotFound)
		return
	}

	var cfg orchestrator.ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		sendJSONError(w, "Erro ao ler configuração: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cfg.ExcludePatterns = req.ExcludePatterns

	updatedData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		sendJSONError(w, "Erro ao processar JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(configPath, updatedData, 0644); err != nil {
		sendJSONError(w, "Erro ao salvar arquivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Padrões de exclusão atualizados com sucesso!"))
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
