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

	"strings"
	"time"
)

type APIHandler struct {
	Plugins     []*registry.ExpertPlugin
	PatternRepo *patterns.PatternRepository
	Engine      *config.Engine
}

func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Wizard-Id", "spec-wizard")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "alive",
		"service": "spec-wizard",
		"version": "1.0.0",
	})
}

// GetLLMClient cria um cliente configurado baseado nas configurações globais ou do projeto
func (h *APIHandler) GetLLMClient(projectCfg orchestrator.ProjectConfig) (*llm.LLMClient, error) {
	providerName := projectCfg.LLM.Provider
	baseURL := projectCfg.LLM.BaseURL
	apiKey := projectCfg.LLM.APIKey
	envKey := projectCfg.LLM.EnvAPIKey
	model := projectCfg.LLM.Model
	sequential := projectCfg.LLM.Sequential

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
			if !sequential {
				sequential = p.Sequential
			}
			if !projectCfg.LLM.UseCLI {
				projectCfg.LLM.UseCLI = p.UseCLI
			}
		}
	}

	if providerName == "" {
		return nil, fmt.Errorf("nenhum provedor de IA configurado (global ou no projeto)")
	}

	provider, err := llm.NewProvider(providerName, baseURL, apiKey, envKey, sequential, projectCfg.LLM.UseCLI, h.Plugins)
	if err != nil {
		return nil, err
	}
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
	stackTemplates := make([]interface{}, 0)
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
			if err != nil {
				fmt.Printf("❌ [GetPatterns] Erro ao buscar opções do Expert %s: %v\n", expert.ID, err)
			} else {
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

				// C) Templates de Stack
				if st, ok := options["stack_templates"]; ok {
					if list, ok := st.([]interface{}); ok {
						stackTemplates = list
						fmt.Printf("✅ [GetPatterns] Expert %s forneceu %d templates para %s\n", expert.ID, len(stackTemplates), lang)
					} else {
						fmt.Printf("⚠️ [GetPatterns] Expert %s retornou stack_templates em formato inválido\n", expert.ID)
					}
				} else {
					fmt.Printf("ℹ️ [GetPatterns] Expert %s não possui a chave 'stack_templates'\n", expert.ID)
				}
			}
		} else {
			fmt.Printf("❌ [GetPatterns] Falha ao garantir que o Expert %s está rodando: %v\n", expert.ID, err)
		}
	} else {
		fmt.Printf("ℹ️ [GetPatterns] Nenhum expert encontrado para a linguagem: %s\n", lang)
	}

	// Converter mapa de volta para slice
	allPatterns := make([]patterns.Pattern, 0, len(unifiedPatterns))
	for _, p := range unifiedPatterns {
		allPatterns = append(allPatterns, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"patterns":        allPatterns,
		"stack_templates": stackTemplates,
	})
}

// POST /api/initialize - Ancora o projeto com Caminho + Linguagem + Padrões
func (h *APIHandler) InitializeProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path                      string                    `json:"path"`
		ProjectName               string                    `json:"projectName"`
		Language                  string                    `json:"language"`
		Patterns                  []string                  `json:"patterns"`
		Instructions              string                    `json:"instructions"`
		Domain                    string                    `json:"domain"`
		FunctionalRequirements    string                    `json:"functionalRequirements"`
		NonFunctionalRequirements string                    `json:"nonFunctionalRequirements"`
		DataStrategy              string                    `json:"dataStrategy"`
		StateManagement           string                    `json:"stateManagement"`
		ApiContract               string                    `json:"apiContract"`
		Customization             string                    `json:"customization"`
		Architecture              string                    `json:"architecture"`
		Stack                     *orchestrator.StackPlugin `json:"stack"`
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
		Stack:                     req.Stack,
	}

	if err := core.InitializeWithLanguage(config, h.PatternRepo, expert); err != nil {
		cleanMsg := cleanLLMError(err)
		sendJSONError(w, "Falha na inicialização: "+cleanMsg, http.StatusInternalServerError)
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
		cleanMsg := cleanLLMError(err)
		sendJSONError(w, "Falha ao salvar roadmap: "+cleanMsg, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Roadmap salvo com sucesso!"))
}

// POST /api/roadmap - Gera o plano de ação (sprints.json) baseado no design
func (h *APIHandler) GenerateRoadmap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path                   string                    `json:"path"`
		Language               string                    `json:"language"`
		Patterns               []string                  `json:"patterns"`
		Architecture           string                    `json:"architecture"`
		DataStrategy           string                    `json:"dataStrategy"`
		StateManagement        string                    `json:"stateManagement"`
		ApiContract            string                    `json:"apiContract"`
		Customization          string                    `json:"customization"`
		AdditionalInstructions string                    `json:"additionalInstructions"`
		Stack                  *orchestrator.StackPlugin `json:"stack"`
		ProjectName            string                    `json:"projectName"`
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
		Stack:           req.Stack,
		ProjectName:     req.ProjectName,
		UserLanguage:    userLang,
	}

	// 3. Chama a lógica de geração do Roadmap (que usa a Prompt Factory e LM Studio)
	roadmap, err := core.GenerateInitialRoadmap(config)
	if err != nil {
		cleanMsg := cleanLLMError(err)
		sendJSONError(w, "Falha ao gerar roadmap: "+cleanMsg, http.StatusInternalServerError)
		return
	}

	// 3. Retorna o JSON do Roadmap para o cliente visualizar no Dashboard
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roadmap)
}

// POST /api/execute-task - Executa uma tarefa específica via Clean Window
func (h *APIHandler) ExecuteTask(w http.ResponseWriter, r *http.Request) {
	// Panic recovery para evitar 500 silencioso em caso de falha catastrófica no orquestrador
	defer func() {
		if r := recover(); r != nil {
			logger.Error("🔥 [API] Panic em ExecuteTask", fmt.Errorf("%v", r))
			sendJSONError(w, "Erro interno catastrófico no orquestrador. Verifique os logs do servidor.", http.StatusInternalServerError)
		}
	}()

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

	// 3. (Removido) O contexto agora é montado internamente pelo Harness TDD

	// 4. Resolve a configuração de linguagem do usuário
	userLang := r.Header.Get("X-Language")
	if userLang == "" {
		userLang = "en"
	}

	// 5. Carrega a configuração do projeto (LLM, Language, etc.)
	pConfig := orchestrator.ProjectConfig{
		Language:     state.Language,
		UserLanguage: userLang,
	}

	projectCfgPath := filepath.Join(req.ProjectPath, ".spec-wizard", "config.json")
	if data, err := os.ReadFile(projectCfgPath); err == nil {
		var loadedCfg orchestrator.ProjectConfig
		if err := json.Unmarshal(data, &loadedCfg); err == nil {
			pConfig.LLM = loadedCfg.LLM
			pConfig.KnowledgeBase = loadedCfg.KnowledgeBase
			pConfig.Stack = loadedCfg.Stack
			pConfig.Roadmap = loadedCfg.Roadmap
			pConfig.Patterns = loadedCfg.Patterns
			pConfig.Architecture = loadedCfg.Architecture
			pConfig.ProjectName = loadedCfg.ProjectName
			pConfig.Domain = loadedCfg.Domain
			pConfig.FunctionalRequirements = loadedCfg.FunctionalRequirements
			pConfig.NonFunctionalRequirements = loadedCfg.NonFunctionalRequirements
		}
	}

	// Injeta os dados da requisição como fallback para o Harness
	pConfig.CurrentSprint = &req.Sprint
	pConfig.CurrentTask = &req.Task

	// AUTO-SAVE: Garante que os dados estejam sincronizados antes da execução
	core.SaveAgentsDocs(pConfig, nil)

	// 6. Executa a tarefa via Harness TDD do Orquestrador (Loop de Auto-correção)
	fmt.Printf("🔄 Iniciando execução da tarefa %v via Harness TDD...\n", req.TaskID)

	// Usamos context.Background() em vez de r.Context() para evitar que o fechamento da conexão
	// pelo browser (timeout do lado do cliente) interrompa o processo de IA em modelos lentos.
	// Aumentamos o timeout global para 30 minutos como uma rede de segurança (safety net).
	// O controle real de latência agora é feito por tentativa (300s cada) dentro do Orquestrador.
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Second)
	defer cancel()

	aiResponse, success, err := core.ExecuteTaskWithTDD(ctx, req.SprintID, req.TaskID, pConfig)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error("❌ Timeout na execução da tarefa (600s excedidos)", err)
			sendJSONError(w, "A execução da tarefa demorou demais (timeout de 10min). Verifique se o modelo selecionado está disponível ou tente um modelo mais rápido.", http.StatusGatewayTimeout)
			return
		}
		// Mesmo com erro de validação (success=false), retornamos o que a IA gerou se houver
		if aiResponse != "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":            "partial_fail",
				"message":           err.Error(),
				"ai_response":       aiResponse,
				"validation_passed": false,
			})
			return
		}
		sendJSONError(w, "Falha na execução: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 7. Retorna a resposta da IA (validada e com testes passando)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "success",
		"task_id":           req.TaskID,
		"ai_response":       aiResponse,
		"validation_passed": success,
		"message":           "Tarefa concluída e validada com sucesso! ✅",
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

// GET /api/project/headless/next-step?path={path}
func (h *APIHandler) GetNextHeadlessStep(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("path")
	if projectPath == "" {
		projectPath = "."
	}

	// 1. Lê o sprints.json
	roadmapPath := filepath.Join(projectPath, ".spec-wizard", "sprints.json")
	absPath, _ := filepath.Abs(roadmapPath)
	fmt.Printf("📂 [Headless] Lendo roadmap de: %s\n", absPath)
	data, err := os.ReadFile(roadmapPath)
	if err != nil {
		sendJSONError(w, "Roadmap não encontrado em "+roadmapPath, http.StatusNotFound)
		return
	}

	var roadmap orchestrator.ProjectRoadmap
	if err := json.Unmarshal(data, &roadmap); err != nil {
		sendJSONError(w, "Erro ao ler roadmap: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Encontra a tarefa (específica ou a próxima pendente)
	targetTaskID := r.URL.Query().Get("task_id")
	var nextTask *orchestrator.Task
	var nextSprint *orchestrator.Sprint

	for i := range roadmap.Sprints {
		for j := range roadmap.Sprints[i].Tasks {
			task := &roadmap.Sprints[i].Tasks[j]
			
			// Se pediu ID específico, busca ele
			if targetTaskID != "" {
				if string(task.ID) == targetTaskID {
					nextTask = task
					nextSprint = &roadmap.Sprints[i]
					goto found
				}
				continue
			}

			// Se não, busca a próxima pendente
			status := task.Status
			if status == "pending" || status == "⏳" || status == "❌" || status == "" {
				nextTask = task
				nextSprint = &roadmap.Sprints[i]
				goto found
			}
		}
	}

found:
	if nextTask == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "completed",
			"message": "Todas as tarefas foram concluídas!",
		})
		return
	}

	// 3. Monta o prompt para a tarefa
	assembler := orchestrator.NewContextAssembler(absPath)
	taskContext, err := assembler.AssembleTaskContext(*nextTask, *nextSprint, h.Plugins)
	if err != nil {
		sendJSONError(w, "Falha ao montar contexto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	prompt := assembler.BuildPromptForTask(taskContext)

	// 4. Retorna o pacote "Headless"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "pending",
		"sprint_id":   nextSprint.ID,
		"task_id":     nextTask.ID,
		"title":       nextTask.Title,
		"description": nextTask.Description,
		"prompt":      prompt,
		"instruction": "Você deve executar esta tarefa agora. Após terminar, chame o endpoint de auditoria /api/project/audit-task para validar.",
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
	h.Engine.AddProject(projectName, req.Path)

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

			if p.UseCLI {
				// Para CLI, o status depende do expert correspondente ao modelo
				if !registry.GlobalManager.IsHealthy(m.Name) {
					status = "offline"
				}
			} else {
				if err := h.Engine.PingProvider(p.Name); err != nil {
					status = "offline"
				}
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

// GET /api/plugins/status - Retorna o status de saúde de todos os experts registrados
func (h *APIHandler) GetPluginsStatus(w http.ResponseWriter, r *http.Request) {
	status := make(map[string]interface{})
	for _, p := range h.Plugins {
		isHealthy := registry.GlobalManager.IsHealthy(p.ID)
		isRunning := registry.GlobalManager.IsRunning(p.ID)

		status[p.ID] = map[string]interface{}{
			"id":        p.ID,
			"name":      p.ID, // Usamos ID como nome por enquanto
			"running":   isRunning,
			"healthy":   isHealthy,
			"endpoint":  p.Endpoint,
			"language":  p.Language,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GET /api/llm/config - Retorna a configuração global de provedores com status em tempo real
func (h *APIHandler) GetLLMConfig(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	cfg := h.Engine.GetConfig()

	type ProviderWithStatus struct {
		config.ProviderConfig
		Status string `json:"status"`
	}

	providers := make([]ProviderWithStatus, len(cfg.Providers))
	for i, p := range cfg.Providers {
		status := "online"
		if p.UseCLI {
			// Busca se existe um expert com id igual ao nome do primeiro modelo
			expertID := ""
			if len(p.Models) > 0 {
				expertID = p.Models[0].Name
			}

			if expertID != "" {
				if !registry.GlobalManager.IsHealthy(expertID) {
					status = "offline"
				}
			}
		} else {
			if err := h.Engine.PingProvider(p.Name); err != nil {
				status = "offline"
			}
		}

		providers[i] = ProviderWithStatus{
			ProviderConfig: p,
			Status:         status,
		}
	}

	response := map[string]interface{}{
		"active":    cfg.Active,
		"providers": providers,
		"projects":  cfg.Projects,
		"port":      cfg.Port,
		"theme":     cfg.Theme,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// POST /api/llm/provider - Adiciona ou atualiza um provedor
func (h *APIHandler) AddOrUpdateProvider(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	var req config.ProviderConfig

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		sendJSONError(w, "Nome do provedor é obrigatório", http.StatusBadRequest)
		return
	}

	if err := h.Engine.AddOrUpdateProvider(req); err != nil {
		sendJSONError(w, "Erro ao adicionar/atualizar provedor: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Provedor atualizado com sucesso"})
}

// POST /api/llm/provider/model - Adiciona um modelo a um provedor
func (h *APIHandler) AddModel(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	var req struct {
		ProviderName string             `json:"provider"`
		Model        config.ModelConfig `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if req.ProviderName == "" || req.Model.Name == "" {
		sendJSONError(w, "Nome do provedor e nome do modelo são obrigatórios", http.StatusBadRequest)
		return
	}

	if err := h.Engine.AddModel(req.ProviderName, req.Model); err != nil {
		sendJSONError(w, "Erro ao adicionar modelo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Modelo adicionado com sucesso"})
}

// DELETE /api/llm/provider - Remove um provedor
func (h *APIHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		sendJSONError(w, "Nome do provedor é obrigatório", http.StatusBadRequest)
		return
	}

	if err := h.Engine.DeleteProvider(name); err != nil {
		sendJSONError(w, "Erro ao excluir provedor: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Provedor excluído com sucesso"})
}

// DELETE /api/llm/provider/model - Remove um modelo de um provedor
func (h *APIHandler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	if h.Engine == nil {
		sendJSONError(w, "Config Engine não inicializada", http.StatusInternalServerError)
		return
	}

	provider := r.URL.Query().Get("provider")
	model := r.URL.Query().Get("model")

	if provider == "" || model == "" {
		sendJSONError(w, "Nomes do provedor e do modelo são obrigatórios", http.StatusBadRequest)
		return
	}

	if err := h.Engine.DeleteModel(provider, model); err != nil {
		sendJSONError(w, "Erro ao excluir modelo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Modelo excluído com sucesso"})
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
	if req.Config.Stack != nil {
		fmt.Printf("📦 [API] Stack recebida: %s (%s) com %d libs\n", req.Config.Stack.Name, req.Config.Stack.ID, len(req.Config.Stack.Libraries))
	} else {
		fmt.Printf("⚠️ [API] Stack não recebida no payload!\n")
	}

	core := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)

	// Injeta o roadmap no objeto de config para centralização
	req.Config.Roadmap = req.Roadmap

	err := core.SaveAgentsDocs(req.Config, h.PatternRepo)

	if err != nil {
		sendJSONError(w, "Erro ao salvar documentação: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Registra/Atualiza o projeto na lista global
	projectName := req.Config.ProjectName
	if projectName == "" {
		projectName = filepath.Base(req.Path)
	}
	h.Engine.AddProject(projectName, req.Path)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Documentação .spec-wizard/ atualizada com sucesso!"))
}

// GET /api/projects - Lista todos os projetos salvos (Global)
func (h *APIHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects := h.Engine.GetProjects()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

// POST /api/projects - Adiciona um projeto à lista
func (h *APIHandler) AddProject(w http.ResponseWriter, r *http.Request) {
	var entry config.ProjectEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	if entry.Name == "" || entry.Path == "" {
		sendJSONError(w, "Nome e Path são obrigatórios", http.StatusBadRequest)
		return
	}

	if err := h.Engine.AddProject(entry.Name, entry.Path); err != nil {
		sendJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Projeto adicionado com sucesso!"))
}

// DELETE /api/projects - Remove um projeto da lista
func (h *APIHandler) RemoveProject(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSONError(w, "Path é obrigatório", http.StatusBadRequest)
		return
	}

	if err := h.Engine.RemoveProject(path); err != nil {
		sendJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Projeto removido com sucesso!"))
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

	// Tenta carregar a config básica para o prompt de depuração
	var config orchestrator.ProjectConfig
	wizardPath := filepath.Join(path, ".spec-wizard")
	if data, err := os.ReadFile(filepath.Join(wizardPath, "config.json")); err == nil {
		json.Unmarshal(data, &config)
	}

	// Tenta carregar o roadmap para achar a primeira tarefa (depuração)
	var task *orchestrator.Task
	var sprint *orchestrator.Sprint
	var roadmap orchestrator.ProjectRoadmap
	if data, err := os.ReadFile(filepath.Join(wizardPath, "sprints.json")); err == nil {
		if err := json.Unmarshal(data, &roadmap); err == nil {
			for i := range roadmap.Sprints {
				for j := range roadmap.Sprints[i].Tasks {
					t := roadmap.Sprints[i].Tasks[j]
					if t.Status == "pending" || t.Status == "in_progress" {
						task = &roadmap.Sprints[i].Tasks[j]
						sprint = &roadmap.Sprints[i]
						break
					}
				}
				if task != nil {
					break
				}
			}
		}
	}

	prompt, err := core.GenerateHarnessPrompt(task, sprint, config, question)
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
// POST /api/project/check-tests
func (h *APIHandler) CheckTaskTests(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			errObj := fmt.Errorf("%v", r)
			logger.Error("🔥 [CheckTaskTests] PANIC RECOVERED", errObj)
			sendJSONError(w, fmt.Sprintf("Erro interno (panic): %v", r), http.StatusInternalServerError)
		}
	}()

	var req struct {
		ProjectPath string                  `json:"project_path"`
		SprintID    orchestrator.FlexibleID `json:"sprint_id"`
		TaskID      orchestrator.FlexibleID `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("❌ [CheckTaskTests] Falha ao decodificar payload", err)
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	logger.Info("📡 [CheckTaskTests] Recebido", "sprintID", req.SprintID, "taskID", req.TaskID, "path", req.ProjectPath)

	absPath, _ := filepath.Abs(req.ProjectPath)
	core := orchestrator.NewOrchestrator(absPath, h.Plugins, h.Engine)

	state, err := core.CheckInitialState()
	if err != nil {
		sendJSONError(w, "Projeto não encontrado", http.StatusNotFound)
		return
	}

	// 1. Rodar Sensor de Validação
	expert, _ := registry.FindExpertByLanguage(state.Language, h.Plugins)
	sensor := orchestrator.NewSensorManager(absPath, state.Language, expert)
	sensor.RunValidation(nil)

	success := !sensor.HasFailures()
	report := sensor.GetFailureReport()

	// 2. Realizar Auditoria Granular (Análise dos logs via IA)
	var auditCriteria []orchestrator.AcceptanceCriterion
	var auditTests []string
	var auditFiles []string

	pConfig, ok := state.Config.(orchestrator.ProjectConfig)
	if !ok {
		logger.Warn("⚠️ [CheckTaskTests] Configuração do projeto não é ProjectConfig ou está ausente. Pulando auditoria IA.")
	} else {
		var auditErr error
		var auditSummary string
		// Aumentamos o timeout para 10 minutos para permitir reparos autônomos
		auditCtx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
		defer cancel()
		auditCriteria, auditFiles, auditTests, auditSummary, auditErr = core.AuditTaskProgress(auditCtx, pConfig, req.SprintID, req.TaskID, report)
		if auditErr != nil {
			logger.Error("❌ [CheckTaskTests] Falha na auditoria IA", auditErr)
			// Não paramos o fluxo, apenas logamos o erro e seguimos com a persistência básica
		} else if auditSummary != "" {
			// Concatenamos o resumo da auditoria ao relatório para que a IA de execução tenha o diagnóstico detalhado
			report = fmt.Sprintf("--- 🤖 RELATÓRIO DE AUDITORIA IA ---\n%s\n\n--- 🚨 LOGS DO SENSOR ---\n%s", auditSummary, report)
		}
	}

	// 3. Persistir resultado
	testStatus := "failure"
	if success {
		testStatus = "success"
	}

	err = core.UpdateTaskProgress(req.SprintID, req.TaskID, "", report, testStatus, auditCriteria, auditFiles, auditTests)
	if err != nil {
		logger.Error("❌ [CheckTaskTests] Falha ao persistir status", err)
		sendJSONError(w, "Falha ao persistir status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     success,
		"report":      report,
		"test_status": testStatus,
	})
}

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

	// Timeout de 600s (10 minutos) para a auditoria
	ctx, cancel := context.WithTimeout(r.Context(), 600*time.Second)
	defer cancel()

	result, err := core.AuditTask(ctx, req.SprintID, req.TaskID, config)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error("❌ Timeout na auditoria (600s excedidos)", err)
			sendJSONError(w, "A auditoria demorou demais para responder (timeout de 10min). Verifique se o modelo selecionado está disponível ou tente um modelo mais rápido.", http.StatusGatewayTimeout)
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

// GET /api/project/task-spec?path={path}&task_id={id}
func (h *APIHandler) GetTaskSpec(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	taskID := r.URL.Query().Get("task_id")

	if path == "" || taskID == "" {
		sendJSONError(w, "Caminho e ID da tarefa são obrigatórios", http.StatusBadRequest)
		return
	}

	specPath := filepath.Join(path, ".spec-wizard", "tasks", fmt.Sprintf("task-%s.md", taskID))
	content, err := os.ReadFile(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Retorna vazio se não existir, não é um erro crítico
			json.NewEncoder(w).Encode(map[string]string{"content": ""})
			return
		}
		sendJSONError(w, "Erro ao ler spec: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"content": string(content)})
}

// POST /api/project/task-spec
func (h *APIHandler) SaveTaskSpec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		TaskID  string `json:"task_id"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	tasksDir := filepath.Join(req.Path, ".spec-wizard", "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		sendJSONError(w, "Erro ao criar diretório: "+err.Error(), http.StatusInternalServerError)
		return
	}

	specPath := filepath.Join(tasksDir, fmt.Sprintf("task-%s.md", req.TaskID))
	if err := os.WriteFile(specPath, []byte(req.Content), 0644); err != nil {
		sendJSONError(w, "Erro ao salvar arquivo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Spec da tarefa salva com sucesso!"))
}

// POST /api/project/task-spec/bootstrap
func (h *APIHandler) BootstrapTaskSpec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		SprintID string `json:"sprint_id"`
		TaskID   string `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Payload inválido", http.StatusBadRequest)
		return
	}

	logger.Info("📡 [API] POST /api/bootstrap-task-spec", "taskID", req.TaskID, "path", req.Path)
	startTime := time.Now()

	o := orchestrator.NewOrchestrator(req.Path, h.Plugins, h.Engine)
	state, err := o.CheckInitialState()
	if err != nil {
		sendJSONError(w, "Erro ao carregar estado do projeto: "+err.Error(), http.StatusInternalServerError)
		return
	}

	config, ok := state.Config.(orchestrator.ProjectConfig)
	if !ok {
		// Tenta converter se for map (acontece quando carrega do JSON sem tipo forte)
		if m, ok := state.Config.(map[string]interface{}); ok {
			data, _ := json.Marshal(m)
			json.Unmarshal(data, &config)
		} else {
			sendJSONError(w, "Configuração do projeto inválida", http.StatusInternalServerError)
			return
		}
	}

	content, err := o.BootstrapTaskSpec(config, req.SprintID, req.TaskID)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("❌ [API] Falha no BootstrapTaskSpec", err, "duration", duration.String())
		sendJSONError(w, "Erro ao gerar bootstrap da spec: "+err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info("✅ [API] BootstrapTaskSpec concluído", "duration", duration.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"content": content,
		"message": "Spec gerada com sucesso via Architect!",
	})
}

func sendJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func cleanLLMError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Tenta extrair mensagem amigável de erros 429/OpenRouter que contém JSON
	if (strings.Contains(msg, "429") || strings.Contains(msg, "error")) && strings.Contains(msg, "{") {
		start := strings.Index(msg, "{")
		end := strings.LastIndex(msg, "}")
		if start != -1 && end != -1 && end > start {
			jsonStr := msg[start : end+1]
			var detail struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if json.Unmarshal([]byte(jsonStr), &detail) == nil && detail.Error.Message != "" {
				return detail.Error.Message
			}
		}
	}
	return msg
}
