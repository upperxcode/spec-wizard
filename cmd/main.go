package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"spec-wizard/api"
	"spec-wizard/config"
	"spec-wizard/internal/adapters"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/orchestrator"
	"spec-wizard/internal/patterns"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/theme"
	"spec-wizard/internal/tui"
	"spec-wizard/internal/workspace"
	"syscall"
)

func main() {
	// 1. Inicializa Logs Estruturados
	logFile := "server.log"
	for _, arg := range os.Args {
		if arg == "--tui" {
			logFile = "tui.log"
			break
		}
	}
	logger.Init(logFile)
	logger.Info("🧙‍♂️ Sistema Iniciado", "mode", logFile)

	// 2. Carregamento de plugins e repositório
	plugins, err := registry.LoadExperts("config/experts.yaml")
	if err != nil {
		fmt.Printf("⚠️ Erro ao carregar experts: %v\n", err)
	}

	// 1.5. Configuração Global (Engine)
	configPath := os.Getenv("HOME") + "/.spec-wizard/llm_config.json"
	if _, err := os.Stat(".spec-wizard/llm_config.json"); err == nil {
		configPath = ".spec-wizard/llm_config.json"
	}

	fmt.Printf("📂 Usando configuração: %s\n", configPath)
	engine, _ := config.NewEngine(configPath)

	patternRepo := patterns.NewRepository()
	ws := workspace.NewWorkspace(os.Getenv("HOME") + "/.spec-wizard")

	handler := &api.APIHandler{
		Plugins:     plugins,
		Engine:      engine,
		PatternRepo: patternRepo,
		Workspace:   ws,
	}

	// Verificar se modo TUI está ativo
	useTUI := false
	projectPath := "."
	for i, arg := range os.Args {
		if arg == "--tui" {
			useTUI = true
		}
		if arg == "--path" && i+1 < len(os.Args) {
			projectPath = os.Args[i+1]
		}
	}

	if useTUI {
		fmt.Println("🚀 Iniciando TUI...")

		// Setup Orchestrator for anchoring
		core := orchestrator.NewOrchestrator(projectPath, plugins, engine)
		state, _ := core.CheckInitialState()

		// O Orchestrator (core) agora atua como o Brain central, gerenciando o Claw-code e os prompts

		projectName := filepath.Base(projectPath)
		if state != nil && state.IsInitialized {
			projectName = state.ProjectName
			logger.Info("📦 Projeto detectado", "language", state.Language, "project", projectName)
		}

		expertClient := adapters.NewExpertClient()

		// Pré-inicia o expert de Go se o projeto for Go
		if state != nil && state.Language == "go" {
			for _, p := range plugins {
				if p.ID == "go-expert" {
					logger.Info("🚀 Pré-iniciando Expert de Go para suporte a formatação...")
					registry.GlobalManager.EnsureExpertRunning(p)
					break
				}
			}
		}

		// Função de formatação dinâmica
		formatter := func(lang, code string) string {
			if lang == "go" {
				for _, p := range plugins {
					if p.ID == "go-expert" {
						// Se o expert ainda não foi iniciado (endpoint vazio), inicia agora
						if p.Endpoint == "" {
							logger.Info("🔄 [TUI] Iniciando Expert de Go sob demanda para formatação...")
							if err := registry.GlobalManager.EnsureExpertRunning(p); err != nil {
								logger.Error("❌ [TUI] Falha ao iniciar expert sob demanda", err)
								return code
							}
						}

						formatted, err := expertClient.FormatCode(p, code)
						if err == nil {
							return formatted
						}
						logger.Debug("⚠️ Falha ao formatar código", "error", err)
						return code
					}
				}
			}
			return code
		}

		// 4. Carrega o tema salvo
		savedTheme := engine.GetConfig().Theme
		if savedTheme != "" && savedTheme != "dracula" {
			t, err := theme.LoadTheme(filepath.Join("./themes", savedTheme+".json"))
			if err == nil {
				tui.UpdateStyles(t)
			} else {
				logger.Error("❌ [Main] Erro ao carregar tema salvo", err)
				tui.UpdateStyles(theme.DefaultDracula())
			}
		} else {
			tui.UpdateStyles(theme.DefaultDracula())
		}

		if err := tui.Start(projectName, core, formatter); err != nil {
			fmt.Printf("❌ Erro na TUI: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 2. Registro ÚNICO de Handlers
	http.HandleFunc("/api/languages", handler.GetLanguages)
	http.HandleFunc("/api/patterns/", handler.GetPatterns)
	http.HandleFunc("/api/initialize", handler.InitializeProject)
	http.HandleFunc("/api/roadmap", handler.GenerateRoadmap)
	http.HandleFunc("/api/llm/status", handler.CheckLLMStatus)
	http.HandleFunc("/api/llm/config", handler.GetLLMConfig)
	http.HandleFunc("/api/llm/active", handler.SetActiveModel)
	http.HandleFunc("/api/llm/provider/status", handler.SetProviderStatus)
	http.HandleFunc("/api/llm/reload", handler.ReloadLLMConfig)
	http.HandleFunc("/api/execute-task", handler.ExecuteTask)
	http.HandleFunc("/api/knowledge", handler.GetKnowledge)
	http.HandleFunc("/api/knowledge/upload", handler.UploadKnowledge)
	http.HandleFunc("/api/knowledge/link", handler.AddKnowledgeLink)

	http.HandleFunc("/api/project/status", handler.GetProjectStatus)
	http.HandleFunc("/api/project/generate-prompt", handler.GenerateTaskPrompt)
	http.HandleFunc("/api/project/interpret", handler.InterpretProject)
	http.HandleFunc("/api/project/infer-purpose", handler.InferPurpose)
	http.HandleFunc("/api/project/save-spec", handler.SaveProjectSpec)
	http.HandleFunc("/api/project/harness-prompt", handler.GetHarnessPrompt)
	http.HandleFunc("/api/project/generate-roadmap-from-code", handler.GenerateRoadmapFromCode)
	http.HandleFunc("/api/project/roadmap-prompt", handler.GetRoadmapPrompt)
	http.HandleFunc("/api/project/save-roadmap", handler.SaveRoadmap)
	http.HandleFunc("/api/project/audit-task", handler.AuditTaskStatus)
	http.HandleFunc("/api/project/config/exclude", handler.UpdateExcludePatterns)
	http.HandleFunc("/api/project/rename", handler.RenameProject)
	http.HandleFunc("/api/project", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			handler.DeleteProject(w, r)
		} else {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.ListProjects(w, r)
		case http.MethodPost:
			handler.AddProject(w, r)
		case http.MethodDelete:
			handler.RemoveProject(w, r)
		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})

	fmt.Println("🧙‍♂️ Spec Wizard API ativa em http://localhost:8080")

	// 3. Início do servidor com CORS simples
	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Language")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.DefaultServeMux.ServeHTTP(w, r)
	})

	// 4. Captura de sinal para encerramento gracioso
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		fmt.Println("\n🛑 [Main] Encerrando sistema e liberando recursos...")
		registry.GlobalManager.StopCurrent()
		os.Exit(0)
	}()

	serverErr := http.ListenAndServe(":8080", corsHandler)
	if serverErr != nil {
		fmt.Printf("❌ Falha ao iniciar o servidor: %v\n", serverErr)
	}
}
