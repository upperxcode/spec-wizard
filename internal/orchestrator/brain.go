package orchestrator

import (
	"context"
	"path/filepath"
	"spec-wizard/internal/analytic"
	"spec-wizard/internal/llm"
	"spec-wizard/internal/theme"
	"strings"
	"time"
)

// BrainStatus contém as informações para o rodapé informativo da TUI
type BrainStatus struct {
	Mode        string
	Provider    string
	ModelName   string
	ContextSize int
	TokensUsed  int
	Theme       string
}

// Brain define a interface de comunicação entre as interfaces (TUI/Web) e o servidor
type Brain interface {
	ProcessQuery(ctx context.Context, query string) (string, error)
	SetProgressCallback(cb func(string))
	GetStatus() BrainStatus
	ListThemes() []string
	SetTheme(name string) error
	DeleteProjectAnchor() error
	RenameProject(newName string) error
	GetThemesDir() string
}

func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, error) {
	// 0. Inicia Sessão Analítica
	session := analytic.NewSession("query-" + time.Now().Format("150405"))
	defer session.Finalize("") // No final do método, o defer finaliza

	// 1. Resolve o cliente LLM (usando Claw-code internamente)
	p, modelName, _ := o.Engine.GetActiveModel()
	provider, err := llm.NewProvider(p.Name, p.APIURL, p.APIKey, p.EnvAPIKey, p.Sequential, p.UseCLI, o.Plugins)
	if err != nil {
		return "", err
	}
	client := llm.NewLLMClient(provider, modelName.Name, o.Tools)
	client.OnProgress = o.onProgress

	// 2. Define o System Prompt Base (O Cérebro detém a instrução)
	systemPrompt := `You are Spec Wizard, a senior software engineer and architecture expert.
Your mission is to help the developer navigate, understand, and modify the current project's codebase.
GUIDELINES:
1. Be technical, precise, and direct.
2. When listing files or content, always show the FULL original code within Markdown code blocks.
3. Use the available tools (list_dir, read_file, etc.) whenever you need real information from the project.
4. If the user asks to list a file, READ the file and display its content, not just a summary.
5. IMPORTANT: Always respond to the user in Brazilian Portuguese (Português do Brasil), unless the technical context requires English terms.`

	client.SystemPrompt = systemPrompt

	// 3. Gerenciamento de Estado (Comutação de Modo)
	if strings.HasPrefix(query, "/") {
		parts := strings.Fields(query)
		cmd := parts[0]

		switch cmd {
		case "/plan":
			o.Mode = ModePlanning
			return "🎯 **MODO PLANEJAMENTO ATIVADO**\n\nAgora, todas as minhas ações serão precedidas por um plano detalhado. Eu esperarei sua validação antes de executar qualquer mudança no código.", nil

		case "/build":
			o.Mode = ModeBuilding
			return "⚡ **MODO CONSTRUÇÃO ATIVADO**\n\nFoco em execução rápida e direta. Vou implementar as mudanças o mais breve possível.", nil

		case "/wizard":
			o.Mode = ModeWizard
			return "🧙‍♂️ **MODO WIZARD ATIVADO**\n\nAgora estou operando com contexto total do projeto. Vou considerar automaticamente o conteúdo da pasta `.spec-wizard` (SPEC, PRD e Skills) em todas as nossas interações.", nil

		case "/chat":
			o.Mode = ModeNormal
			return "💬 **MODO CHAT ATIVADO**\n\nVoltamos à conversa normal.", nil

		case "/theme":
			name := ""
			if len(parts) > 1 {
				name = parts[1]
			}

			if name == "" {
				themes := o.ListThemes()
				return "🎨 **TEMAS DISPONÍVEIS:**\n\n- " + strings.Join(themes, "\n- ") + "\n\nUse `/theme <nome>` para trocar.", nil
			}

			err := o.SetTheme(name)
			if err != nil {
				return "❌ Erro ao carregar tema: " + err.Error(), nil
			}
			return "🎨 **TEMA ATUALIZADO:** " + name, nil

		default:
			// Wrapper para outros comandos do Claw (/login, /status, etc.)
			resp, err := client.AskWithContext(ctx, query, session)
			if err == nil {
				session.Finalize(resp)
			}
			return resp, err
		}
	}

	// 4. Aplicação de Diretivas Baseadas no Estado Atual
	switch o.Mode {
	case ModePlanning:
		client.SystemPrompt += "\nDIRECTIVE: You are in PLANNING MODE. For any user request, first ANALYZE the codebase and propose a step-by-step PLAN. DO NOT execute changes (write_file, etc.) without explicit user approval of the plan. Be detailed and architectural."
	case ModeBuilding:
		client.SystemPrompt += "\nDIRECTIVE: You are in BUILDING MODE. Execute requested changes immediately and accurately. Focus on direct code output and tool usage. Be concise and fast."
	case ModeWizard:
		client.SystemPrompt += "\nDIRECTIVE: You are in WIZARD MODE. You MUST use the following project context as your source of truth for architecture, rules, and business logic."
		ctxContent := o.getWizardContext()
		if ctxContent != "" {
			client.SystemPrompt += "\n\n### PROJECT CONTEXT (.spec-wizard)\n" + ctxContent
		}
	}

	// 5. Delega a execução ao Claw-code
	resp, err := client.AskWithContext(ctx, query, session)
	if err == nil {
		session.Finalize(resp)
	}
	return resp, err
}

// GetStatus retorna o estado atual para o rodapé da TUI
func (o *Orchestrator) GetStatus() BrainStatus {
	modeStr := "Chat"
	switch o.Mode {
	case ModePlanning:
		modeStr = "Plan"
	case ModeBuilding:
		modeStr = "Build"
	case ModeWizard:
		modeStr = "Wizard"
	}

	p, m, err := o.Engine.GetActiveModel()
	if err != nil {
		return BrainStatus{Mode: modeStr, Provider: "N/A", ModelName: "Desconectado"}
	}

	return BrainStatus{
		Mode:        modeStr,
		Provider:    strings.ToLower(p.Name),
		ModelName:   strings.ToLower(m.Label),
		ContextSize: m.Context,
		TokensUsed:  0,
		Theme:       o.Engine.GetConfig().Theme,
	}
}

func (o *Orchestrator) ListThemes() []string {
	themes, _ := theme.ListThemes(o.GetThemesDir())
	// Adiciona o tema padrão sempre
	return append([]string{"dracula"}, themes...)
}

func (o *Orchestrator) GetThemesDir() string {
	return o.Engine.GetThemesDir()
}

func (o *Orchestrator) SetTheme(name string) error {
	if name == "dracula" {
		return o.Engine.SetTheme("dracula")
	}

	// Tenta carregar o arquivo
	path := filepath.Join(o.GetThemesDir(), name+".json")
	_, err := theme.LoadTheme(path)
	if err != nil {
		return err
	}

	// Salva no config
	return o.Engine.SetTheme(name)
}

func (o *Orchestrator) SetProgressCallback(cb func(string)) {
	o.onProgress = cb
}

func (o *Orchestrator) getWizardContext() string {
	ca := NewContextAssembler(o.ProjectPath)

	var sb strings.Builder

	// Tenta carregar os arquivos principais
	if prd, err := ca.loadFile("PRD.md"); err == nil {
		sb.WriteString("#### BUSINESS CONTEXT (PRD):\n" + prd + "\n\n")
	}
	if spec, err := ca.loadFile("SPEC.md"); err == nil {
		sb.WriteString("#### TECHNICAL SPECIFICATION (SPEC):\n" + spec + "\n\n")
	}
	if skills, err := ca.loadFile("SKILLS.md"); err == nil {
		sb.WriteString("#### GOLDEN RULES (SKILLS):\n" + skills + "\n\n")
	} else if skills, err := ca.loadFile("skills.md"); err == nil {
		sb.WriteString("#### GOLDEN RULES (SKILLS):\n" + skills + "\n\n")
	}

	return sb.String()
}
