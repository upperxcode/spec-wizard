package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"spec-wizard/internal/orchestrator"
	"spec-wizard/internal/theme"
	"strings"

	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type Command struct {
	ID   string
	Desc string
}

type model struct {
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	err           error
	brain         orchestrator.Brain
	projectName   string
	renderer      *glamour.TermRenderer
	codeFormatter func(lang, code string) string
	showCommands  bool
	commands      []Command
	filteredCmds  []Command
	cursor        int
	isWaiting     bool
	isCommand     bool
	currentStatus string
	windowWidth   int
	windowHeight  int
	ctrlCCount    int
}

func NewModel(projectName string, brain orchestrator.Brain, formatter func(lang, code string) string) *model {

	ta := textarea.New()
	ta.Placeholder = "Pergunte algo ao Spec Wizard... (digite / para comandos)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 1000
	ta.SetWidth(80)
	ta.SetHeight(1)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)
	vp.SetContent(`🧙‍♂️ Bem-vindo ao Spec Wizard TUI!
Ancorado no projeto: ` + projectName + "\n\n")

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	cmds := []Command{
		{ID: "/plan", Desc: "Ativar Modo Planejamento (Pensa antes de agir)"},
		{ID: "/build", Desc: "Ativar Modo Construção (Execução direta)"},
		{ID: "/wizard", Desc: "Ativar Modo Wizard (Contexto total da .spec-wizard)"},
		{ID: "/chat", Desc: "Ativar Modo Chat (Conversa normal)"},
		{ID: "/theme", Desc: "Trocar o tema visual (Plugins)"},
		{ID: "/status", Desc: "Snapshot do estado e sessão (Claw-Wrapper)"},
		{ID: "/model", Desc: "Trocar o modelo de IA ativo (Claw-Wrapper)"},
		{ID: "/clear", Desc: "Limpar histórico da conversa (Claw-Wrapper)"},
		{ID: "/help", Desc: "Mostrar ajuda de comandos (Claw-Wrapper)"},
		{ID: "/rename", Desc: "Alterar o nome do projeto (Claw-Wrapper)"},
		{ID: "/delete", Desc: "Remover ancoragem .spec-wizard (CUIDADO!)"},
		{ID: "/exit", Desc: "Sair do Spec Wizard (Claw-Wrapper)"},
	}

	return &model{
		textarea:      ta,
		viewport:      vp,
		messages:      []string{},
		brain:         brain,
		projectName:   projectName,
		renderer:      renderer,
		codeFormatter: formatter,
		commands:      cmds,
		filteredCmds:  cmds,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tick())
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	val := m.textarea.Value()
	m.showCommands = strings.HasPrefix(val, "/")

	if m.showCommands {
		m.filteredCmds = []Command{}
		for _, c := range m.commands {
			if strings.HasPrefix(c.ID, val) {
				m.filteredCmds = append(m.filteredCmds, c)
			}
		}
		if m.cursor >= len(m.filteredCmds) {
			m.cursor = 0
		}
	} else {
		m.cursor = 0
	}

	switch msg := msg.(type) {
	case tickMsg:
		if m.isWaiting {
			return m, tick()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)

		// Recalcula altura do viewport de forma mais agressiva
		// header(1) + status(1) + input(1) + footer(1) = 4
		// Usamos -5 para dar uma margem de segurança contra quebras de linha
		m.viewport.Height = msg.Height - 5

		// Atualiza o renderer de markdown SEM auto-style para evitar "rgb trash"
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(msg.Width-4),
		)
		return m, nil

	case tea.KeyMsg:
		if msg.Type != tea.KeyCtrlC {
			m.ctrlCCount = 0
		}

		if m.showCommands && len(m.filteredCmds) > 0 {
			switch msg.Type {
			case tea.KeyUp:
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.filteredCmds) - 1
				}
				return m, nil
			case tea.KeyDown:
				m.cursor++
				if m.cursor >= len(m.filteredCmds) {
					m.cursor = 0
				}
				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				selected := m.filteredCmds[m.cursor]
				m.textarea.SetValue(selected.ID + " ")
				m.showCommands = false

				// Se for um comando que pode rodar direto, executa
				if selected.ID == "/status" || selected.ID == "/clear" || selected.ID == "/help" || selected.ID == "/exit" || selected.ID == "/model" || selected.ID == "/plan" || selected.ID == "/build" || selected.ID == "/wizard" || selected.ID == "/chat" || selected.ID == "/delete" {
					return m.handleCommand(selected.ID)
				}
				return m, nil
			}
		}

		switch msg.Type {
		case tea.KeyEsc:
			return m, nil
		case tea.KeyCtrlC:
			m.ctrlCCount++
			if m.ctrlCCount >= 2 {
				return m, tea.Quit
			}
			m.currentStatus = "Pressione Ctrl+C novamente para sair"
			return m, nil
		case tea.KeyEnter:
			if m.showCommands && len(m.filteredCmds) > 0 {
				selected := m.filteredCmds[m.cursor]
				m.textarea.SetValue(selected.ID + " ")
				m.showCommands = false
				return m, nil
			}

			v := m.textarea.Value()
			if v == "" {
				return m, nil
			}

			if strings.HasPrefix(v, "/") {
				return m.handleCommand(v)
			}

			m.messages = append(m.messages, m.renderMessage("user", v))
			m.viewport.SetContent(strings.Join(m.messages, "\n"))
			m.textarea.Reset()
			m.viewport.GotoBottom()

			return m, tea.Batch(m.askBrain(v), tick())
		}

	case responseMsg:
		m.isWaiting = false
		m.currentStatus = ""
		content := string(msg)

		if content == "" {
			content = "⚠️ **O modelo retornou uma resposta vazia.** Verifique os logs para mais detalhes."
		}

		role := "wizard"

		// Detecta troca de tema para atualizar estilos em tempo real
		if strings.HasPrefix(content, "🎨 **TEMA ATUALIZADO:**") {
			themeName := strings.TrimSpace(strings.TrimPrefix(content, "🎨 **TEMA ATUALIZADO:**"))
			if themeName == "dracula" {
				UpdateStyles(theme.DefaultDracula())
			} else {
				t, err := theme.LoadTheme(filepath.Join("./themes", themeName+".json"))
				if err == nil {
					UpdateStyles(t)
				}
			}
		}

		m.messages = append(m.messages, m.renderMessage(role, content))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, nil

	case progressMsg:
		m.currentStatus = string(msg)
		return m, nil

	case errMsg:
		m.isWaiting = false
		m.currentStatus = ""
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

type responseMsg string
type progressMsg string
type tickMsg time.Time
type errMsg error

func (m *model) askBrain(prompt string) tea.Cmd {
	m.isWaiting = true
	m.isCommand = strings.HasPrefix(prompt, "/")
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := m.brain.ProcessQuery(ctx, prompt)
		if err != nil {
			return errMsg(err)
		}
		return responseMsg(resp)
	}
}

func (m *model) handleCommand(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	root := parts[0]

	switch root {
	case "/exit":
		return m, tea.Quit
	case "/clear":
		m.messages = []string{}
		m.viewport.SetContent("🧙‍♂️ **Histórico limpo.**")
		m.textarea.Reset()
		return m, nil
	case "/help":
		var helpLines []string
		helpLines = append(helpLines, "✨ **Comandos Disponíveis:**\n")
		for _, c := range m.commands {
			helpLines = append(helpLines, fmt.Sprintf("• `%s`: %s", c.ID, c.Desc))
		}
		helpText := strings.Join(helpLines, "\n")
		m.messages = append(m.messages, m.renderMessage("wizard", helpText))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		m.textarea.Reset()
		return m, nil
	case "/delete":
		err := m.brain.DeleteProjectAnchor()
		if err != nil {
			m.messages = append(m.messages, m.renderMessage("wizard", "❌ **Erro ao remover ancoragem:** "+err.Error()))
			return m, nil
		}
		m.messages = append(m.messages, m.renderMessage("wizard", "✅ **Ancoragem .spec-wizard removida com sucesso.** Saindo..."))
		return m, tea.Quit
	case "/rename":
		if len(parts) < 2 {
			m.messages = append(m.messages, m.renderMessage("wizard", "⚠️ **Uso:** `/rename <novo_nome>`"))
			return m, nil
		}
		newName := strings.Join(parts[1:], " ")
		err := m.brain.RenameProject(newName)
		if err != nil {
			m.messages = append(m.messages, m.renderMessage("wizard", "❌ **Erro ao renomear:** "+err.Error()))
			return m, nil
		}
		m.projectName = newName
		m.messages = append(m.messages, m.renderMessage("wizard", "✅ **Projeto renomeado para:** "+newName))
		m.textarea.Reset()
		return m, nil
	case "/plan", "/build", "/wizard", "/chat", "/theme", "/status", "/diff", "/model", "/config", "/login", "/cost":
		m.textarea.Reset()
		return m, m.askBrain(cmd)
	default:
		// Se não conhecemos, tratamos como chat normal
		m.messages = append(m.messages, m.renderMessage("user", cmd))
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()
		return m, m.askBrain(cmd)
	}
}

func (m *model) View() string {
	var (
		header      = m.headerView()
		viewport    = m.viewport.View()
		statusLine  = ""
		commandMenu = ""
		input       = m.textarea.View()
		footer      = m.footerView()
	)

	if m.showCommands && len(m.filteredCmds) > 0 {
		var rows []string
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			Width(12)
		cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Width(12)
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)

		for i, c := range m.filteredCmds {
			style := cmdStyle
			if i == m.cursor {
				style = selectedStyle
			}

			row := lipgloss.JoinHorizontal(lipgloss.Top,
				style.Render(c.ID),
				descStyle.Render(c.Desc),
			)
			rows = append(rows, row)
		}

		menuBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Render(strings.Join(rows, "\n"))

		commandMenu = menuBox
	}

	if m.isWaiting {
		msg := "Pensando"
		if m.currentStatus != "" {
			msg = m.currentStatus
		} else if m.isCommand {
			msg = "Processando comando"
		}

		dots := strings.Repeat(".", int(time.Now().UnixNano()/500000000%4))
		msg = "⏳ " + msg + dots

		statusLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Italic(true).
			Bold(true).
			PaddingLeft(2).
			Render(msg)
	}

	// Se o menu de comandos estiver aberto, ele "flutua" acima do input
	// Para manter o layout fixo, o viewport diminui se o menu aparecer (ajuste manual se necessário)

	mainLayout := lipgloss.JoinVertical(lipgloss.Left,
		header,
		viewport,
		statusLine,
		commandMenu,
		input,
		footer,
	)

	// Força o preenchimento de toda a altura da tela
	return lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.windowHeight).
		MaxHeight(m.windowHeight).
		Render(mainLayout)
}

func (m *model) footerView() string {
	status := m.brain.GetStatus()

	// Segmentos
	wizard := footerWizardStyle.Render("Spec Wizard")
	mode := footerModeStyle.Render(status.Mode)
	provider := footerProviderStyle.Render("🌐 " + status.Provider)

	// Modelo com truncamento
	modelName := status.ModelName
	maxModelWidth := m.viewport.Width / 4
	if len(modelName) > maxModelWidth && maxModelWidth > 3 {
		modelName = modelName[:maxModelWidth-3] + "..."
	}
	model := footerModelStyle.Render("🤖 " + modelName)

	ctx := footerStatStyle.Render(fmt.Sprintf("ctx: %d", status.ContextSize))
	tokens := footerStatStyle.Render(fmt.Sprintf("tokens: %d", status.TokensUsed))

	// Monta a barra
	barContent := lipgloss.JoinHorizontal(lipgloss.Top, wizard, mode, provider, model, ctx, tokens)

	// Preenchimento para garantir largura total
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#282A36")).
		Width(m.viewport.Width).
		Render(barContent)
}

func (m *model) headerView() string {
	title := titleStyle.Render("🧙‍♂️ SPEC WIZARD")
	project := projectStyle.Render(" [" + m.projectName + "]")
	return title + project
}

func Start(projectName string, brain orchestrator.Brain, formatter func(lang, code string) string) error {
	p := tea.NewProgram(NewModel(projectName, brain, formatter), tea.WithAltScreen())

	// Registra o callback de progresso para enviar mensagens ao programa
	brain.SetProgressCallback(func(status string) {
		p.Send(progressMsg(status))
	})

	_, err := p.Run()
	return err
}
