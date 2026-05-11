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
	pathConfig "spec-wizard/internal/config"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/orchestrator"
	"spec-wizard/internal/patterns"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/theme"
	"spec-wizard/internal/tui"
	"time"

	"os/exec"
	"strings"
	"syscall"
)

func main() {
	// 1.5. Configuração Global do Aplicativo (LLM, Temas, Projetos)
	paths := pathConfig.GetPaths()
	paths.EnsureDirs()

	// 1. Inicializa Logs Estruturados
	logFileName := "server.log"
	customLogPath := ""
	for i, arg := range os.Args {
		if arg == "--tui" {
			logFileName = "tui.log"
		}
		if arg == "--log" && i+1 < len(os.Args) {
			customLogPath = os.Args[i+1]
		}
	}

	logFile := filepath.Join(paths.ResourcesDir, "logs", logFileName)
	if customLogPath != "" {
		logFile = customLogPath
	}

	logger.Init(logFile)
	logger.Info("🧙‍♂️ Sistema Iniciado", "mode", logFile)

	globalConfigPath := paths.ResolveConfig("config.json")
	systemExpertsPath := paths.ResolveResource("experts.yaml")
	userExpertsPath := filepath.Join(paths.ResourcesDir, "user_experts.yaml") // Permite override do usuário

	// Inicializa o gerenciador de plugins com o diretório de recursos
	registry.GlobalManager.ResourcesDir = paths.ResourcesDir

	// 2. Carregamento de plugins e repositório
	// Tenta carregar do diretório de recursos (instalado) e de um possível arquivo de override
	plugins, err := registry.LoadExperts(systemExpertsPath, userExpertsPath, "config/experts.yaml")
	if err != nil {
		fmt.Printf("⚠️ Erro ao carregar experts: %v\n", err)
	}

	themesDir := filepath.Join(paths.ResourcesDir, "themes")
	fmt.Printf("📂 Usando configuração global: %s\n", globalConfigPath)
	engine, err := config.NewEngine(globalConfigPath, themesDir)
	if err != nil {
		logger.Error("❌ Falha ao carregar configuração global", err)
		os.Exit(1)
	}

	// Exibe um resumo da configuração ativa para transparência
	active := engine.GetConfig().Active
	if active.Provider != "" {
		fmt.Printf("🤖 LLM Ativo: %s (%s)\n", active.Provider, active.Model)
		// Busca detalhes do provedor
		for _, p := range engine.GetConfig().Providers {
			if p.Name == active.Provider {
				fmt.Printf("🔗 API URL:   %s\n", p.APIURL)
				keyStatus := "✅ Configurada"
				if p.APIKey == "" {
					if os.Getenv(p.EnvAPIKey) != "" {
						keyStatus = fmt.Sprintf("✅ Resolvida via ENV (%s)", p.EnvAPIKey)
					} else {
						keyStatus = fmt.Sprintf("❌ Faltando (Esperado ENV: %s)", p.EnvAPIKey)
					}
				}
				fmt.Printf("🔑 Auth:      %s\n", keyStatus)
				break
			}
		}
	}

	// 1.6. Resolução de Porta (Prioridade: Flag > Config > Env > Default)
	port := engine.GetConfig().Port
	if port == "" {
		port = "10808"
		if envPort := os.Getenv("SPEC_WIZARD_PORT"); envPort != "" {
			port = envPort
		}
	}

	// 1.7. Parsing de Flags
	useTUI := false
	killServer := false
	projectPath := "."
	portFromFlag := ""
	for i, arg := range os.Args {
		if arg == "--tui" {
			useTUI = true
		}
		if arg == "--kill" {
			killServer = true
		}
		if arg == "--path" && i+1 < len(os.Args) {
			projectPath = os.Args[i+1]
		}
		if arg == "--port" && i+1 < len(os.Args) {
			portFromFlag = os.Args[i+1]
		}
	}

	// 1.8. Execução de Comandos Especiais
	if killServer {
		fmt.Printf("🛑 [Kill] Tentando encerrar processos na porta %s...\n", port)
		if err := killProcessOnPort(port); err != nil {
			fmt.Printf("⚠️  Aviso: %v\n", err)
		} else {
			fmt.Printf("✅ [Kill] Porta %s liberada!\n", port)
		}
		os.Exit(0)
	}

	if portFromFlag != "" {
		port = portFromFlag
		// Salva a nova porta na configuração global
		engine.SetPort(port)
	} else if engine.GetConfig().Port == "" {
		// Salva a porta padrão se não houver nenhuma salva
		engine.SetPort(port)
	}

	patternRepo := patterns.NewRepository()

	// 1.9. Limpeza automática de instâncias anteriores na mesma porta (Agnóstico a flags)
	_ = killProcessOnPort(port)

	handler := &api.APIHandler{
		Plugins:     plugins,
		Engine:      engine,
		PatternRepo: patternRepo,
	}

	// Se um path foi passado, garante que ele está registrado e ativo na config global
	if projectPath != "." {
		absPath, _ := filepath.Abs(projectPath)
		engine.AddProject(filepath.Base(absPath), absPath)
		engine.SetActiveProject(absPath)
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
			t, err := theme.LoadTheme(filepath.Join(engine.GetThemesDir(), savedTheme+".json"))
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
	http.HandleFunc("/api/health", handler.HealthCheck)
	http.HandleFunc("/api/languages", handler.GetLanguages)
	http.HandleFunc("/api/patterns/", handler.GetPatterns)
	http.HandleFunc("/api/initialize", handler.InitializeProject)
	http.HandleFunc("/api/llm/status", handler.CheckLLMStatus)
	http.HandleFunc("/api/plugins/status", handler.GetPluginsStatus)
	http.HandleFunc("/api/llm/config", handler.GetLLMConfig)
	http.HandleFunc("/api/llm/active", handler.SetActiveModel)
	http.HandleFunc("/api/llm/provider/status", handler.SetProviderStatus)
	http.HandleFunc("/api/llm/provider", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.AddOrUpdateProvider(w, r)
		case http.MethodDelete:
			handler.DeleteProvider(w, r)
		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/llm/provider/model", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handler.AddModel(w, r)
		case http.MethodDelete:
			handler.DeleteModel(w, r)
		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/llm/reload", handler.ReloadLLMConfig)
	http.HandleFunc("/api/execute-task", handler.ExecuteTask)
	http.HandleFunc("/api/knowledge", handler.GetKnowledge)
	http.HandleFunc("/api/knowledge/upload", handler.UploadKnowledge)
	http.HandleFunc("/api/knowledge/link", handler.AddKnowledgeLink)

	http.HandleFunc("/api/project/status", handler.GetProjectStatus)
	http.HandleFunc("/api/project/generate-prompt", handler.GenerateTaskPrompt)
	http.HandleFunc("/api/project/interpret", handler.InterpretProject)
	http.HandleFunc("/api/project/headless/next-step", handler.GetNextHeadlessStep)
	http.HandleFunc("/api/project/infer-purpose", handler.InferPurpose)
	http.HandleFunc("/api/project/save-spec", handler.SaveProjectSpec)
	http.HandleFunc("/api/project/harness-prompt", handler.GetHarnessPrompt)
	http.HandleFunc("/api/project/generate-roadmap-from-code", handler.GenerateRoadmapFromCode)
	http.HandleFunc("/api/project/roadmap-prompt", handler.GetRoadmapPrompt)
	http.HandleFunc("/api/project/save-roadmap", handler.SaveRoadmap)
	http.HandleFunc("/api/project/audit-task", handler.AuditTaskStatus)
	http.HandleFunc("/api/project/check-tests", handler.CheckTaskTests)
	http.HandleFunc("/api/project/config/exclude", handler.UpdateExcludePatterns)
	http.HandleFunc("/api/project/task-spec", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetTaskSpec(w, r)
		case http.MethodPost:
			handler.SaveTaskSpec(w, r)
		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/project/task-spec/bootstrap", handler.BootstrapTaskSpec)
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

	// 3. Servidor de Arquivos Estáticos para a UI (SPA)
	uiDir := paths.UIDir
	if _, err := os.Stat(uiDir); err == nil {
		fmt.Printf("🌐 GUI Web disponível em: http://localhost:%s\n", port)
		fs := http.FileServer(http.Dir(uiDir))
		http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Se não for API e o arquivo não existir fisicamente, serve o index.html (padrão SPA)
			if !strings.HasPrefix(r.URL.Path, "/api") {
				path := filepath.Join(uiDir, r.URL.Path)
				if _, err := os.Stat(path); os.IsNotExist(err) {
					http.ServeFile(w, r, filepath.Join(uiDir, "index.html"))
					return
				}
			}
			fs.ServeHTTP(w, r)
		}))
	} else {
		fmt.Printf("⚠️  Aviso: Interface Web não instalada em %s. Use scripts/run ui para desenvolvimento.\n", uiDir)
	}

	fmt.Printf("🧙‍♂️ Spec Wizard API ativa em http://localhost:%s\n", port)

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

	// 3. Pré-inicialização de experts críticos (Agnostic) para disponibilidade imediata
	for _, p := range plugins {
		if p.Language == "agnostic" {
			logger.Info("🚀 [Main] Pré-iniciando Expert Agnostico...", "id", p.ID)
			go registry.GlobalManager.EnsureExpertRunning(p)
		}
	}

	serverErr := http.ListenAndServe(":"+port, corsHandler)
	if serverErr != nil {
		fmt.Printf("❌ Falha ao iniciar o servidor na porta %s: %v\n", port, serverErr)
	}
}

func killProcessOnPort(port string) error {
	// 1. Tenta identificar se é o Spec Wizard
	client := http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/api/health", port))
	if err != nil {
		return fmt.Errorf("não foi possível validar o serviço na porta %s: %v", port, err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Wizard-Id") != "spec-wizard" {
		return fmt.Errorf("segurança: processo na porta %s não é uma instância reconhecida do Spec Wizard", port)
	}

	// 2. Se for o Spec Wizard, mata usando lsof (Linux/macOS)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -ti:%s | xargs kill -9", port))
	return cmd.Run()
}
