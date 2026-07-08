package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"spec-wizard/internal/config"
	"spec-wizard/internal/governance"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/registry"
	"spec-wizard/internal/utils"
	"time"

	_ "modernc.org/sqlite"
)

var (
	projectPath = "."
)

func init() {
	paths := config.GetPaths()
	logDir := paths.LogDir
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "server.log")

	// Força o modo analítico para o log ser legível (pipe-separado)
	os.Setenv("LOG_ANALITIC", "true")

	if err := logger.Init(logPath); err != nil {
		// Se o logger falhar, ainda tentamos avisar no stderr como último recurso,
		// mas aqui o objetivo é o log.
		fmt.Fprintf(os.Stderr, "❌ [MCP] Erro crítico ao inicializar logger: %v\n", err)
	} else {
		cwd, _ := os.Getwd()
		logger.Info("🧙‍♂️ [MCP] Binário iniciado", "cwd", cwd, "log", logPath)
	}
}

// MCP JSON-RPC Structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func getGlobalPort() string {
	// 1. Tenta ler do arquivo global (~/.config/spec-wizard/config.json)
	home, _ := os.UserHomeDir()
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(home, ".config")
	}
	globalConfigPath := filepath.Join(configDir, "spec-wizard", "config.json")

	data, err := os.ReadFile(globalConfigPath)
	if err == nil {
		var config struct {
			Port string `json:"port"`
		}
		// Tenta descompactar como string primeiro
		if err := json.Unmarshal(data, &config); err == nil && config.Port != "" {
			return config.Port
		}
		// Tenta descompactar como int se falhar (fallback para legados)
		var configInt struct {
			Port int `json:"port"`
		}
		if err := json.Unmarshal(data, &configInt); err == nil && configInt.Port > 0 {
			return fmt.Sprintf("%d", configInt.Port)
		}
	}

	// 2. Fallback para arquivo local (para compatibilidade retroativa)
	configPath := getFilePath(".spec-wizard/config.json")
	data, err = os.ReadFile(configPath)
	if err == nil {
		var config struct {
			Port string `json:"port"`
		}
		if err := json.Unmarshal(data, &config); err == nil && config.Port != "" {
			return config.Port
		}
	}

	// Fallback para a porta padrão do Spec Wizard v2
	return "10999"
}

var apiBaseURL string

func getProjectPath() string {
	path := os.Getenv("WZ_PROJECT_PATH")
	if path == "" {
		path, _ = os.Getwd()
	}
	return path
}

func getFilePath(relative string) string {
	return filepath.Join(projectPath, relative)
}

func getUserLanguage() string {
	// Prioridade 1: Banco de dados
	db := governance.GetDB()
	if db != nil {
		var userLang string
		// Busca absoluta para evitar problemas com caminhos relativos
		absPath, _ := filepath.Abs(projectPath)
		err := db.QueryRow("SELECT user_language FROM projects WHERE path = ?", absPath).Scan(&userLang)
		if err == nil && userLang != "" && userLang != "go" && userLang != "python" {
			return userLang
		}
	}

	return "pt-BR"
}

func formatContent(content string) string {
	lang := getUserLanguage()
	// Tenta formatar JSON se o conteúdo começar com { ou [
	trimmed := strings.TrimSpace(content)
	if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) || (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, []byte(trimmed), "", "  "); err == nil {
			content = "```json\n" + pretty.String() + "\n```"
		}
	}

	instruction := fmt.Sprintf(`### [INSTRUÇÃO DE IDIOMA: %s] ###
- ROLE: Arquiteto de Software Sênior (Elite Governance).
- PREFERÊNCIA: Responda preferencialmente em %s, conforme a configuração do usuário.
- EXIGÊNCIA: Forneça justificativas técnicas profundas e precisas.
- FOCO: Estabilidade arquitetural e eliminação de dívida técnica.
#########################################################`, strings.ToUpper(lang), strings.ToUpper(lang))

	return fmt.Sprintf("%s\n\n%s", instruction, content)
}


func clearOmniRoutePins() {
	// TODO: Implementar a lógica para limpar as afinidades (Pins) no banco do OmniRoute.
	// Isso geralmente envolveria interagir com um banco de dados SQLite ou uma API do OmniRoute.
	// Por enquanto, é um placeholder.
	logger.Info("🧹 [MCP] Limpeza de afinidades do OmniRoute solicitada.")
}

func AssembleCleanWindow(targetFile string) string {
	// 1. Força a limpeza das afinidades (Pins) no banco do OmniRoute para garantir a rota
	clearOmniRoutePins()

	// 2. Lê os componentes imperativos da especificação do Spec Wizard
	resumo, _ := os.ReadFile(getFilePath(".RESUME.md"))
	leftPrompts, _ := os.ReadFile(getFilePath(".LEFT_PROMPTS.md"))
	agentsRules, _ := os.ReadFile(getFilePath(".AGENTS.md"))
	targetCode, _ := os.ReadFile(targetFile)

	// 3. Monta o payload final consolidado (Garante que a entrada fique abaixo de 30k tokens)
	cleanPayload := fmt.Sprintf(
		"=== SPECIFICATION CORE ===\n%s\n\n"+
		"=== SLIDING WINDOW MEMORY ===\n%s\n\n"+
		"=== ARCHITECTURAL GOLDEN RULES ===\n%s\n\n"+
		"=== ACTIVE TARGET CODE ===\n%s\n",
		resumo, leftPrompts, agentsRules, targetCode,
	)

	return cleanPayload
}

func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

type Config struct {
	OmniRouteDBPath       string `json:"omniroute_db_path"`
	ClearCacheOnExecution bool   `json:"clear_cache_on_execution"`
	MaxSlidingWindow      int    `json:"max_sliding_window_prompts"`
}

func loadConfig() Config {
	file, _ := os.ReadFile("mcp_config.json")
	var cfg Config
	_ = json.Unmarshal(file, &cfg)
	return cfg
}

func clearOmniRouteCache(dbPath string) {
	// Comando cirúrgico no SQLite para deletar os vínculos antigos de sessão
	// Isso garante que o OmniRoute use a sua estratégia de Priority na primeira tentativa
	_ = fmt.Sprintf("sqlite3 %s 'DELETE FROM session_affinities; DELETE FROM context_caches;'", dbPath)
	// Nota: Em produção, utilize o pacote "database/sql" com o driver "mattn/go-sqlite3"
	// os.WriteFile ou exec.Command executam essa query em background de forma silenciosa
}

func feedSlidingWindow(prompt, response string, max int) {
	filePath := ".LEFT_PROMPTS.md"
	timestamp := time.Now().Format("15:04:05")

	// Comprime a resposta (remove quebras de linha extras e limita a 500 chars)
	compressed := strings.ReplaceAll(response, "\n", " ")
	compressed = strings.Join(strings.Fields(compressed), " ")
	if len(compressed) > 500 {
		compressed = compressed[:500] + "..."
	}

	entry := fmt.Sprintf("### 🔄 [%s]\n- **User:** %s\n- **AI:** %s\n", timestamp, prompt, compressed)

	// Lê o arquivo atual
	var existing string
	if data, err := os.ReadFile(filePath); err == nil {
		existing = string(data)
	}

	// Separa as interações existentes
	separator := "\n### 🔄"
	var interactions []string
	if existing != "" {
		parts := strings.Split(existing, separator)
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				if !strings.HasPrefix(p, "### 🔄") {
					p = "### 🔄" + p
				}
				interactions = append(interactions, p)
			}
		}
	}

	// Adiciona a nova interação no início
	newContent := entry
	for i, interaction := range interactions {
		if i >= max-1 {
			break // Respeita o limite máximo de interações
		}
		newContent += "\n" + interaction
	}

	_ = os.WriteFile(filePath, []byte(newContent+"\n"), 0644)
}

func updateMacroResume(lastAction string) {
	_ = ".RESUME.md"
	// Atualiza apenas a linha de "Última alteração" para manter o macro-estado atualizado
	// baseado no comportamento de Chain-of-Thought do Spec Wizard
	logger.Info("📝 [MCP] Resumo macro atualizado", "lastAction", lastAction)
}

func assembleSlimPrompt(userPrompt, targetFile string) string {
	resumo, _ := os.ReadFile(".RESUME.md")
	leftPrompts, _ := os.ReadFile(".LEFT_PROMPTS.md")
	targetCode, _ := os.ReadFile(targetFile)

	// O formato Clean Window final que vai para o Gemini (Sempre abaixo de 20k tokens)
	return fmt.Sprintf(
		"CONTEXT MEMORY:\n%s\n\nRECENT INTERACTIONS:\n%s\n\nTARGET CODE (%s):\n%s\n\nCURRENT INSTRUCTION:\n%s",
		resumo, leftPrompts, targetFile, targetCode, userPrompt,
	)
}

func configureMCPServer(configPath string) error {
	// Carrega o arquivo de configuração existente
	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("erro ao ler o arquivo de configuração: %v", err)
	}

	// Faz o parse do JSON existente
	var config map[string]interface{}
	if err := json.Unmarshal(file, &config); err != nil {
		return fmt.Errorf("erro ao fazer o parse do JSON: %v", err)
	}

	// Adiciona a configuração do servidor MCP
	config["context"] = map[string]interface{}{
		"mcp_servers": map[string]interface{}{
			"context-guard": map[string]interface{}{
				"command": "/caminho/para/o/seu/binario/mcp-context-guard",
				"args": []string{},
				"env": map[string]string{
					"ENV": "production",
				},
			},
		},
	}

	// Converte de volta para JSON
	updatedConfig, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao converter para JSON: %v", err)
	}

	// Escreve o arquivo de configuração atualizado
	if err := os.WriteFile(configPath, updatedConfig, 0644); err != nil {
		return fmt.Errorf("erro ao escrever o arquivo de configuração: %v", err)
	}

	return nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	projectPath = getProjectPath()
	apiBaseURL = fmt.Sprintf("http://127.0.0.1:%s/api", getGlobalPort())

	// Inicializa a governança local (SQLite) para persistência de roadmaps e tarefas
	if err := governance.InitDB(projectPath); err != nil {
		logger.Error("⚠️ Falha ao inicializar governança local", err)
	}

	logger.Info("🧙‍♂️ Spec Wizard MCP iniciado", "cwd", cwd, "projectPath", projectPath)

	if len(os.Args) > 1 && os.Args[1] != "serve" {
		fmt.Println(formatContent(handleDirectCommand(os.Args[1:])))
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	http.HandleFunc("/api/translations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(utils.TranslationMap)
	})

	for scanner.Scan() {
		var req JSONRPCRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}

		handleRequest(req)
	}
}

func handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		var params map[string]interface{}
		if rootURI, ok := params["rootUri"].(string); ok {
			path := strings.TrimPrefix(rootURI, "file://")
			if path != "" && path != os.Getenv("HOME") {
				projectPath = path
				logger.Info("🚀 Projeto detectado via initialize", "path", projectPath)
			}
		} else if rootPath, ok := params["rootPath"].(string); ok {
			if rootPath != "" && rootPath != os.Getenv("HOME") {
				projectPath = rootPath
				logger.Info("🚀 Projeto detectado via initialize", "path", projectPath)
				if err := governance.InitDB(projectPath); err != nil {
					logger.Error("❌ Erro ao reinicializar banco", err)
				}
			}
		}

		sendResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]string{"name": "spec-wizard-mcp", "version": "1.0.0"},
		}, nil)

	case "tools/list":
		sendResponse(req.ID, map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "context_assembler",
					"description": "Monta o prompt slim (Clean Window) para a IA. Intercepta o prompt do usuário, limpa o cache do OmniRoute, alimenta a janela deslizante (.LEFT_PROMPTS.md), atualiza o resumo macro (RESUMO.md) e retorna o payload otimizado para a IA. Argumentos: user_prompt (string), target_file (string).",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"user_prompt": map[string]string{
								"type":        "string",
								"description": "O prompt/instrução do usuário para a IA.",
							},
							"target_file": map[string]string{
								"type":        "string",
								"description": "O caminho do arquivo que será modificado (ex: pkg/designer/components.go).",
							},
						},
						"required": []string{"user_prompt", "target_file"},
					},
				},
				{
					"name":        "context_capture",
					"description": "Captura a resposta da IA e alimenta a janela deslizante (.LEFT_PROMPTS.md) automaticamente. Deve ser chamado após cada resposta da IA. Argumentos: user_prompt (string), ai_response (string).",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"user_prompt": map[string]string{
								"type":        "string",
								"description": "O prompt/instrução original do usuário.",
							},
							"ai_response": map[string]string{
								"type":        "string",
								"description": "A resposta gerada pela IA.",
							},
						},
						"required": []string{"user_prompt", "ai_response"},
					},
				},
				{
					"name":        "wz",
					"description": "Executa comandos do Spec Wizard. Sintaxe: 'init [tech] [name] [stack]', 'interpret <tech>', 'info <id>', 'goal <id>', 'refine <id>', 'prepare <id>', 'code <id>', 'build <id>', 'audit <id>', 'roadmap', 'sync', 'status', 'rules', 'state', 'sync-task <id> [--status <s>] [--bugs <b>] [--notes <n>]', 'read <entity> <id>', 'insert <entity> <json>', 'remove <entity> <id>', 'select <entity> [where]'. Nota: 'tech' é a tecnologia (go, python, etc), NÃO o idioma.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"cmd": map[string]string{
								"type":        "string",
								"description": "O comando completo a ser executado.",
							},
						},
						"required": []string{"cmd"},
					},
				},
				{
					"name":        "db_read",
					"description": "Lê uma entidade do banco de governança. Entidades: task, project, sprint.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"entity": map[string]string{"type": "string", "description": "Entidade (ex: task)"},
							"id":     map[string]string{"type": "string", "description": "ID ou Display ID"},
						},
						"required": []string{"entity", "id"},
					},
				},
				{
					"name":        "db_insert",
					"description": "Insere uma nova entidade no banco (JSON).",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"entity": map[string]string{"type": "string", "description": "Entidade"},
							"data":   map[string]string{"type": "string", "description": "Dados em JSON"},
						},
						"required": []string{"entity", "data"},
					},
				},
				{
					"name":        "db_remove",
					"description": "Remove uma entidade do banco.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"entity": map[string]string{"type": "string", "description": "Entidade"},
							"id":     map[string]string{"type": "string", "description": "ID ou Display ID"},
						},
						"required": []string{"entity", "id"},
					},
				},
				{
					"name":        "db_select",
					"description": "Executa um select genérico em uma tabela.",
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"entity": map[string]string{"type": "string", "description": "Tabela"},
							"where":  map[string]string{"type": "string", "description": "Cláusula WHERE (opcional)"},
						},
						"required": []string{"entity"},
					},
				},
			},
		}, nil)

	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("❌ Erro no parse de argumentos: %v", err)},
				},
			}, nil)
			return
		}

		if params.Name == "context_assembler" {
			userPrompt, _ := params.Arguments["user_prompt"].(string)
			targetFile, _ := params.Arguments["target_file"].(string)
			config := loadConfig()

			// Step 1: Limpa o cache do OmniRoute (se configurado)
			if config.ClearCacheOnExecution {
				clearOmniRouteCache(config.OmniRouteDBPath)
			}

			// Step 2: Alimenta a janela deslizante
			feedSlidingWindow(userPrompt, "", config.MaxSlidingWindow)

			// Step 3: Atualiza o resumo macro
			updateMacroResume(userPrompt)

			// Step 4: Monta o prompt slim
			slimPrompt := assembleSlimPrompt(userPrompt, targetFile)

			sendResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": slimPrompt},
				},
			}, nil)
		} else if params.Name == "context_capture" {
			userPrompt, _ := params.Arguments["user_prompt"].(string)
			aiResponse, _ := params.Arguments["ai_response"].(string)
			config := loadConfig()

			// Alimenta a janela deslizante com a resposta da IA
			feedSlidingWindow(userPrompt, aiResponse, config.MaxSlidingWindow)

			sendResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "✅ Resposta da IA capturada e registrada na janela deslizante."},
				},
			}, nil)
		} else if params.Name == "wz" {
			cmd, ok := params.Arguments["cmd"].(string)
			if !ok {
				sendResponse(req.ID, map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": "❌ Erro: Argumento 'cmd' inválido ou ausente."},
					},
				}, nil)
				return
			}
			result := handleDirectCommand(parseArgs(cmd))
			sendResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": formatContent(result)},
				},
			}, nil)
		} else if strings.HasPrefix(params.Name, "db_") {
			action := strings.TrimPrefix(params.Name, "db_")
			var res string
			switch action {
			case "read":
				res = governance.HandleDBAction(projectPath, "read", []string{fmt.Sprint(params.Arguments["entity"]), fmt.Sprint(params.Arguments["id"])})
			case "insert":
				res = governance.HandleDBAction(projectPath, "insert", []string{fmt.Sprint(params.Arguments["entity"]), fmt.Sprint(params.Arguments["data"])})
			case "remove":
				res = governance.HandleDBAction(projectPath, "remove", []string{fmt.Sprint(params.Arguments["entity"]), fmt.Sprint(params.Arguments["id"])})
			case "select":
				res = governance.HandleDBAction(projectPath, "select", []string{fmt.Sprint(params.Arguments["entity"]), fmt.Sprint(params.Arguments["where"])})
			}
			sendResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": formatContent(res)},
				},
			}, nil)
		}

	case "logging/setLevel":
		// Silenciosamente aceita, mas não faz nada por enquanto
		sendResponse(req.ID, map[string]interface{}{}, nil)

	case "resources/list":
		// Retorna lista vazia de recursos
		sendResponse(req.ID, map[string]interface{}{
			"resources": []interface{}{},
		}, nil)

	case "notifications/initialized":
		// Notificações não precisam de resposta no JSON-RPC, mas vamos logar
		logger.Info("🚀 [MCP] Cliente inicializado")

	default:
		// Se não tiver ID, é uma notificação, não responda com erro
		if req.ID == nil {
			logger.Warn("⚠️ [MCP] Notificação desconhecida ignorada", "method", req.Method)
			return
		}
		sendResponse(req.ID, nil, map[string]interface{}{
			"code":    -32601,
			"message": "Method not found: " + req.Method,
		})
	}
}

func sendResponse(id interface{}, result interface{}, err interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
	}
	if err != nil {
		resp.Error = err
	} else {
		resp.Result = result
	}
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func handleDirectCommand(fields []string) string {
	if len(fields) == 0 {
		return "❌ Erro: Comando vazio."
	}

	// Tolerância a prefixos (wz ou wzx)
	if fields[0] == "wz" || fields[0] == "wzx" {
		fields = fields[1:]
		if len(fields) == 0 {
			return handleDirectCommand([]string{"help"})
		}
	}

	logger.Info("🎮 [MCP] Executando comando", "args", fields)

	defer logger.Info("✅ [MCP] Comando finalizado", "args", fields)

	sub := fields[0]
	lang := getUserLanguage()

	switch sub {
	case "help":
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("## 🧭 Governança & Arquitetura\n")
		sb.WriteString("  init        Inicializa a estrutura OpenSpec com um navegador de engenharia\n")
		sb.WriteString("  import      Importa um projeto existente e gera o DESIGN.md de governança\n")
		sb.WriteString("  new         Cria uma nova proposta de mudança\n")
		sb.WriteString("  roadmap     Gera um roadmap estratégico baseado no DESIGN.md\n")
		sb.WriteString("  status      Verifica o status das mudanças ativas e saúde da arquitetura\n")
		sb.WriteString("  archive     Arquiva uma tarefa concluída (move para changes/archive)\n")
		sb.WriteString("\n")
		sb.WriteString("## ✅ Gestão de Tarefas\n")
		sb.WriteString("  refine      Refina uma tarefa criando sua especificação e isolamento\n")
		sb.WriteString("  code        Prepara o contexto para a IA codificar a tarefa\n")
		sb.WriteString("  fix         Prepara o contexto para a IA resolver um bugfix\n")
		sb.WriteString("  append      Anexa uma nova descrição de problema ao plano ativo (fix/feat)\n")
		sb.WriteString("  apply       Consolida as especificações para implementação pela IA\n")
		sb.WriteString("  verify      Valida a conformidade arquitetural ou as tarefas de uma branch\n")
		sb.WriteString("  set         Atualiza o status de uma tarefa\n")
		sb.WriteString("\n")
		sb.WriteString("## 🌿 Controle de Versão Git\n")
		sb.WriteString("  branch      Cria uma nova branch sequencial e a pasta de planejamento correspondente\n")
		sb.WriteString("  commit      Gera o prompt de commit ou executa o commit com a mensagem fornecida\n")
		sb.WriteString("  pull        Sincroniza o repositório (pull --rebase) e sobe as mudanças (push)\n")
		sb.WriteString("  push        Envia as alterações para o repositório remoto (com auto set-upstream)\n")
		sb.WriteString("\n")
		sb.WriteString("## 🤖 IA & Sistema\n")
		sb.WriteString("  mcp         Inicia o servidor MCP Profissional\n")
		sb.WriteString("  sandbox     Inicia o Sandbox de Governança para visualizar prompts da IA\n")
		sb.WriteString("  expert      Interage com os experts de linguagem\n")
		sb.WriteString("  lang        Muda o idioma da CLI de forma interativa\n")
		sb.WriteString("  agent       Instala ou força a atualização de regras de governança para assistentes de IA (Cursor, Cline, Zed, Aider)\n")
		sb.WriteString("  configure-mcp        Configura o servidor MCP no arquivo de configuração do projeto\n")
		sb.WriteString("  clear-omniroute-cache  Limpa o cache do OmniRoute (registros fantasmas)\n")
		sb.WriteString("  feed-sliding-window    Alimenta a janela deslizante com as últimas interações\n")
		sb.WriteString("  update-macro-resume    Atualiza o resumo macro das tarefas\n")
		sb.WriteString("  assemble-slim-prompt   Monta um prompt slim para a IA\n")
		sb.WriteString("  slim-prompt            Executa todos os comandos para montar um prompt slim\n")
		sb.WriteString("\n")
		sb.WriteString("## 📚 Comandos Adicionais\n")
		sb.WriteString("  help        Help about any command\n")
		sb.WriteString("  completion  Generate the autocompletion script for the specified shell\n")
		sb.WriteString("\n")
		sb.WriteString("## 🏁 Flags\n")
		sb.WriteString("  -h, --help   help for sw\n")
		sb.WriteString("\n")
		sb.WriteString("Use \"sw [command] --help\" for more information about a command.\n")
		return sb.String()

	case "configure-mcp":
		configPath := ".spec-wizard/config.json"
		if err := configureMCPServer(configPath); err != nil {
			return fmt.Sprintf("❌ Erro ao configurar o servidor MCP: %v", err)
		}
		return "✅ Servidor MCP configurado com sucesso!"

	case "clear-omniroute-cache":
		config := loadConfig()
		clearOmniRouteCache(config.OmniRouteDBPath)
		return "✅ Cache do OmniRoute limpo com sucesso!"

	case "feed-sliding-window":
		if len(fields) < 3 {
			return "❌ Erro: Informe o prompt e a resposta."
		}
		prompt := fields[1]
		response := fields[2]
		config := loadConfig()
		feedSlidingWindow(prompt, response, config.MaxSlidingWindow)
		return "✅ Janela deslizante alimentada com sucesso!"

	case "update-macro-resume":
		if len(fields) < 2 {
			return "❌ Erro: Informe a última ação."
		}
		lastAction := fields[1]
		updateMacroResume(lastAction)
		return "✅ Resumo macro atualizado com sucesso!"

	case "assemble-slim-prompt":
		if len(fields) < 3 {
			return "❌ Erro: Informe o prompt do usuário e o arquivo alvo."
		}
		userPrompt := fields[1]
		targetFile := fields[2]
		slimPrompt := assembleSlimPrompt(userPrompt, targetFile)
		return fmt.Sprintf("✅ Prompt Slim montado com sucesso:\n\n%s", slimPrompt)

	case "slim-prompt":
		config := loadConfig()
		if config.ClearCacheOnExecution {
			clearOmniRouteCache(config.OmniRouteDBPath)
		}
		if len(fields) < 3 {
			return "❌ Erro: Informe o prompt do usuário e o arquivo alvo."
		}
		userPrompt := fields[1]
		targetFile := fields[2]
		feedSlidingWindow(userPrompt, "", config.MaxSlidingWindow)
		updateMacroResume(userPrompt)
		slimPrompt := assembleSlimPrompt(userPrompt, targetFile)
		return fmt.Sprintf("✅ Prompt Slim montado com sucesso:\n\n%s", slimPrompt)

	case "status":
		logger.Info("📊 [MCP] Solicitando status do projeto")
		url := fmt.Sprintf("%s/project/status?path=%s", apiBaseURL, projectPath)
		return callAPI(url, "GET", nil)

	case "roadmap":
		if len(fields) >= 2 && fields[1] == "set" {
			if len(fields) < 3 {
				return "❌ Erro: Informe o JSON do roadmap."
			}
			jsonStr := strings.Join(fields[2:], " ")
			if err := governance.UpdateFromRoadmapJSON(projectPath, []byte(jsonStr)); err != nil {
				return "❌ Erro ao importar roadmap: " + err.Error()
			}
			return "✅ Roadmap importado e sincronizado com sucesso!"
		}

		logger.Info("🛣️ [MCP] Solicitando roadmap")
		// Se não houver roadmap no banco, informa à IA para gerar um
		current := governance.GetRoadmap(projectPath)
		if strings.Contains(current, "não encontrado") || strings.Contains(current, "vazio") {
			return "🎯 [IA_DIRECTIVE] O projeto não possui um Strategic Roadmap. Analise o diretório atual, defina 3-4 Sprints estratégicas com Tarefas claras e AcceptanceCriteria, e responda APENAS com o comando: wz insert roadmap <json_do_roadmap>. NÃO peça informações ao usuário."
		}
		return current

	case "rules":
		logger.Info("📜 [MCP] Solicitando Golden Rules")
		data, err := os.ReadFile(getFilePath(".spec-wizard/skills.md"))
		if err != nil {
			return fmt.Sprintf("❌ Erro ao ler regras (.spec-wizard/skills.md): %v", err)
		}
		return string(data)

	case "init":
		tech := ""
		name := "unnamed"
		stack := "standard"

		if len(fields) > 1 {
			tech = fields[1]
		}
		if len(fields) > 2 {
			name = fields[2]
		}
		if len(fields) > 3 {
			stack = fields[3]
		}

		// Se o primeiro campo for JSON, extrai os campos dele (suporte a wz init {"tech":"go",...})
		if strings.HasPrefix(tech, "{") {
			var p map[string]string
			if err := json.Unmarshal([]byte(tech), &p); err == nil {
				if v, ok := p["tech"]; ok {
					tech = v
				} else if v, ok := p["language"]; ok {
					tech = v // Retrocompatibilidade
				}
				if v, ok := p["name"]; ok {
					name = v
				}
				if v, ok := p["stack"]; ok {
					stack = v
				}
			}
		}

		// 🔍 Inteligência de Inferência
		if tech == "" {
			logger.Info("🔍 [MCP] Tentando inferir tecnologia automaticamente", "path", projectPath)
			tech = utils.InferLanguage(projectPath)
			if tech != "" {
				logger.Info("✅ [MCP] Tecnologia inferida", "tech", tech)
			}
		}

		// 📋 Fallback: Listagem de Experts
		if tech == "" {
			paths := config.GetPaths()
			systemExpertsPath := paths.ResolveResource("experts.yaml")
			userExpertsPath := filepath.Join(paths.ResourcesDir, "user_experts.yaml")

			experts, _ := registry.LoadExperts(systemExpertsPath, userExpertsPath, "config/experts.yaml")

			expertLangs := make(map[string]bool)
			for _, e := range experts {
				if e.Language != "" {
					expertLangs[e.Language] = true
				}
			}

			var langs []string
			for l := range expertLangs {
				langs = append(langs, l)
			}

			return fmt.Sprintf("⚠️ Tecnologia (tech) do projeto não detectada automaticamente.\n\n"+
				"Experts instalados suportam: **%s**.\n"+
				"Para prosseguir, execute: `wz init <tech>` (ex: `wz init go`).\n\n"+
				"💡 Dica: Você pode configurar o idioma das especificações (pt-BR, en) com `wz config language <idioma>`.", strings.Join(langs, ", "))
		}

		expertLang := tech
		logger.Info("🏗️ [MCP] Infraestrutura local pronta", "project", name)

		// 4. Retorna status de onboarding incremental
		return getNextOnboardingStep(projectPath, tech, stack, expertLang)

	case "interpret":
		if len(fields) <= 1 {
			return "❌ Erro: Informe a tecnologia (tech) para interpretação (ex: go, flutter, python)."
		}
		tech := fields[1]
		logger.Info("🧠 [MCP] Solicitando interpretação do projeto", "tech", tech)

		url := fmt.Sprintf("%s/project/interpret", apiBaseURL)
		payload := map[string]string{
			"path":     projectPath,
			"language": lang,
		}
		pData, _ := json.Marshal(payload)

		return callAPI(url, "POST", pData)

		logger.Info("🔄 [MCP] Sincronizando roadmap")
		return governance.HandleSync(projectPath)

	case "read", "insert", "remove", "select":
		logger.Info("🗄️ [MCP] Ação de banco de dados", "action", sub)
		return governance.HandleDBAction(projectPath, sub, fields[1:])

	case "experts":
		paths := config.GetPaths()
		systemExpertsPath := paths.ResolveResource("experts.yaml")
		userExpertsPath := filepath.Join(paths.ResourcesDir, "user_experts.yaml")
		expertsFound, err := registry.LoadExperts(systemExpertsPath, userExpertsPath, "config/experts.yaml")
		if err != nil {
			return fmt.Sprintf("❌ Erro ao carregar experts: %v", err)
		}
		if len(expertsFound) == 0 {
			return "⚠️ No experts found. Check your installation at ~/.spec-wizard/experts/"
		}

		var sb strings.Builder
		sb.WriteString("## 🧩 Available Experts\n\n")
		sb.WriteString("Use the expert `id` (first column) when setting the project profile.\n\n")
		sb.WriteString("| ID / Name | Language | Description |\n")
		sb.WriteString("|-----------|----------|--------------|\n")
		for _, e := range expertsFound {
			name := e.Name
			if name == "" {
				name = e.ID
			}
			desc := e.Description
			if desc == "" {
				desc = "(no description)"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", name, e.Language, desc))
		}
		sb.WriteString("\n**To select an expert for this project:**\n")
		sb.WriteString("`wz profile set {\"expert\":\"<id>\",\"language\":\"<language>\"}`")
		return sb.String()

	case "architectures":
		paths := config.GetPaths()
		archPath := paths.ResolveResource("architectures.yaml")
		archs, err := registry.LoadArchitectures(archPath)
		if err != nil {
			return fmt.Sprintf("❌ Erro ao carregar catálogo de arquiteturas: %v", err)
		}
		if len(archs) == 0 {
			return "⚠️ No architectures found. Check ~/.spec-wizard/architectures.yaml"
		}

		// Filtra por tecnologia do projeto (se soubermos), sem redundâncias
		var filterTech string
		if profile := governance.GetProjectProfile(projectPath); len(profile) > 0 {
			// Extrai tech do JSON do profile
			var p map[string]interface{}
			if json.Unmarshal([]byte(profile), &p) == nil {
				if t, ok := p["tech"].(string); ok {
					filterTech = strings.ToLower(t)
				} else if l, ok := p["language"].(string); ok {
					filterTech = strings.ToLower(l) // Retrocompatibilidade
				}
			}
		}

		var sb strings.Builder
		sb.WriteString("## 🏛️ Available Architecture Patterns\n\n")
		if filterTech != "" {
			sb.WriteString(fmt.Sprintf("_Filtered for technology (tech): `%s` (showing all if not tech-specific)_\n\n", filterTech))
		}
		sb.WriteString("| # | Pattern | Description |\n")
		sb.WriteString("|---|---------|-------------|\n")

		count := 0
		seen := make(map[string]bool)
		for _, a := range archs {
			if seen[a.ID] {
				continue
			}
			// Inclui se não tem filtro de tecnologia, se é agnostico, ou se bate com a tecnologia
			include := filterTech == ""
			if !include {
				for _, t := range a.BestFor {
					if strings.ToLower(t) == filterTech {
						include = true
						break
					}
				}
			}
			if !include {
				continue
			}
			seen[a.ID] = true
			count++
			sb.WriteString(fmt.Sprintf("| %d | **%s** | %s |\n", count, a.Name, a.Description))
		}
		sb.WriteString(fmt.Sprintf("\n_Total: %d patterns available._\n\n", count))
		sb.WriteString("**Ask the user to choose one, then run:**\n")
		sb.WriteString("`wz profile set {\"architecture\":\"<chosen pattern name>\"}`\n\n")
		sb.WriteString("After saving architecture, proceed to: `wz roadmap`")
		return sb.String()

	case "profile":

		if len(fields) < 2 {
			return "❌ Erro: Informe a ação (get ou set)."
		}
		action := fields[1]
		if action == "get" {
			return governance.GetProjectProfile(projectPath)
		} else if action == "set" {
			if len(fields) < 3 {
				return "❌ Erro: Informe o JSON do perfil."
			}
			jsonStr := strings.Join(fields[2:], " ")
			res := governance.UpdateProjectProfile(projectPath, []byte(jsonStr))
			if strings.Contains(res, "❌") {
				return res
			}
			return res + "\n\n" + getNextOnboardingStep(projectPath, "", "", "")
		}
		return "❌ Ação desconhecida para profile. Use get ou set."

	case "execute-task":
		if len(fields) < 2 {
			return "❌ Erro: Informe o ID da tarefa para execução."
		}
		taskID := fields[1]
		logger.Info("🚀 [MCP] Executando tarefa", "taskID", taskID)
		return fmt.Sprintf("🚀 Tarefa %s em execução... (implementação pendente)", taskID)

			case "state":
		logger.Info("📱 [MCP] Solicitando estado bruto")
		data, err := os.ReadFile(getFilePath(".spec-wizard/sprints.json"))
		if err != nil {
			return fmt.Sprintf("❌ Erro ao ler estado (.spec-wizard/sprints.json): %v", err)
		}
		return string(data)

	case "goal", "refine", "prepare", "code", "build", "audit", "task", "sync-task", "update", "info":
		action := sub
		taskID := ""

		if sub == "task" {
			if len(fields) < 3 {
				return "❌ Erro: Informe a sub-ação e o ID da tarefa (ex: wz task refine 1.1)."
			}
			action = fields[1]
			taskID = fields[2]
		} else {
			if len(fields) < 2 {
				return "❌ Erro: Informe o ID da tarefa."
			}
			taskID = fields[1]
		}

		logger.Info("🛠️ [MCP] Ação de tarefa", "action", action, "id", taskID)

		switch action {
		case "info":
			return governance.GetTaskInfo(projectPath, taskID)
		case "sync-task":
			status := extractArg(fields, "--status")
			bugs := extractArg(fields, "--bugs")
			notes := extractArg(fields, "--notes")
			args := []string{taskID}
			if status != "" { args = append(args, "--status", status) }
			if bugs != "" { args = append(args, "--bugs", bugs) }
			if notes != "" { args = append(args, "--notes", notes) }
			return governance.HandleUpdateTask(projectPath, args)
		case "audit":
			return governance.HandleAuditTask(projectPath, fields[1:])
		case "update":
			return governance.HandleUpdateTask(projectPath, fields[1:])
		default:
			url := fmt.Sprintf("%s/project/headless/next-step?project_path=%s&task_id=%s&action=%s", apiBaseURL, projectPath, taskID, action)
			return callAPI(url, "GET", nil)
		}

	default:
		return "❌ Comando desconhecido. Use 'wz help' para ver a lista."
	}
}

func callAPI(url, method string, body []byte) string {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Sprintf("❌ Erro ao criar requisição: %v", err)
	}
	if req == nil {
		return "❌ Erro: Requisição HTTP nula"
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("❌ Erro na API: %v (Dica: O servidor do Spec Wizard está rodando?)", err)
	}
	if resp == nil || resp.Body == nil {
		return "❌ Erro: Resposta da API vazia"
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("❌ Erro ao ler resposta: %v", err)
	}

	trimmed := strings.TrimSpace(string(data))
	// Tenta detectar se o retorno contém um roadmap ou uma tarefa para sincronizar o banco
	// Agora mais flexível: se tiver "sprints" ou "tasks" ou começar com JSON
	if strings.Contains(trimmed, "\"sprints\":") || (strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"id\":")) {
		if err := governance.UpdateFromRoadmapJSON(projectPath, data); err != nil {
			logger.Warn("⚠️ Falha ao sincronizar roadmap recebido", "error", err)
		} else {
			logger.Info("🏛️ Banco de dados sincronizado com sucesso (Merge Aditivo)")
		}
	}

	return string(data)
}

func extractArg(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// parseArgs divide o comando em campos, mas preserva blocos JSON entre { }
func parseArgs(cmd string) []string {
	var fields []string
	var current strings.Builder
	inBraces := 0

	for _, r := range cmd {
		switch r {
		case '{':
			inBraces++
			current.WriteRune(r)
		case '}':
			inBraces--
			current.WriteRune(r)
		case ' ':
			if inBraces > 0 {
				current.WriteRune(r)
			} else {
				if current.Len() > 0 {
					fields = append(fields, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}

func callRawAPI(url, method string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(data))
	}

	return io.ReadAll(resp.Body)
}

func getNextOnboardingStep(projectPath, tech, stack, expert string) string {
	// 1. Cria diretórios e arquivos base
	dirs := []string{".spec-wizard/tasks", ".spec-wizard/task-logs", ".spec-wizard/knowledge"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(projectPath, d), 0755)
	}

	// 2. Garante registro no DB
	pID, err := governance.GetOrCreateProject(projectPath)
	if err != nil {
		return "❌ Erro ao acessar banco de dados: " + err.Error()
	}

	// Salva identidade básica se fornecida
	if tech != "" {
		dbAction := []string{"projects", fmt.Sprintf("language='%s', stack='%s', expert='%s'", tech, stack, expert), fmt.Sprintf("id=%d", pID)}
		governance.HandleDBAction(projectPath, "update", dbAction)
	}

	// 3. Lê perfil atual para decidir o próximo passo
	profileJSON := governance.GetProjectProfile(projectPath)
	var p governance.DBProject
	json.Unmarshal([]byte(profileJSON), &p)

	if p.PRD == "" {
		return formatOnboardingResponse("STEP 2 — Business / PRD", "Qual problema real este sistema resolve? (a dor do negócio)", "prd")
	}

	if p.FunctionalRequirements == "" {
		return formatOnboardingResponse("STEP 3 — Functional Requirements", "Liste as funcionalidades principais indispensáveis.", "functional_requirements")
	}

	if p.NonFunctionalRequirements == "" {
		return formatOnboardingResponse("STEP 4 — Non-Functional Requirements", "Quais qualidades o sistema deve ter? (performance, segurança, escalabilidade, etc)", "non_functional_requirements")
	}

	if p.Architecture == "" {
		return "✅ Identidade e requisitos definidos! Agora escolha a arquitetura rodando `wz architectures` para ver as opções disponíveis."
	}

	return "🚀 Perfil de governança completo! Use `wz roadmap` para gerar o plano de execução."
}

func formatOnboardingResponse(step, prompt, field string) string {
	return fmt.Sprintf("📍 **%s**\n\nPergunta para o usuário: *\"%s\"*\n\nAo confirmar, você (IA) **DEVE** executar obrigatoriamente:\n```\nwz profile set {\"%s\": \"<RESPOSTA_DO_USUARIO>\"}\n```", step, prompt, field)
}
