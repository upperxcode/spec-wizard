package tui

import (
	"regexp"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/theme"

	"strings"

	"github.com/charmbracelet/lipgloss"
)

var ActiveTheme theme.Theme = theme.DefaultDracula()

var codeBlockRegex = regexp.MustCompile("(?s)```([a-z]*)\\n(.*?)\n```")

var (
	wizardStyle         lipgloss.Style
	userStyle           lipgloss.Style
	commandStyle        lipgloss.Style
	titleStyle          lipgloss.Style
	projectStyle        lipgloss.Style
	footerWizardStyle   lipgloss.Style
	footerModeStyle     lipgloss.Style
	footerProviderStyle lipgloss.Style
	footerModelStyle    lipgloss.Style
	footerStatStyle     lipgloss.Style
)

func UpdateStyles(t theme.Theme) {
	ActiveTheme = t

	wizardStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.WizardBorder)).
		Padding(0, 1)

	userStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.UserBorder)).
		Padding(0, 1)

	commandStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.CommandBorder)).
		Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(t.Title))

	projectStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Project))

	footerWizardStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.FooterWizardBg)).
		Foreground(lipgloss.Color(t.FooterBg)).
		Padding(0, 1).
		Bold(true)

	footerModeStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.FooterModeBg)).
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1).
		Italic(true)

	footerProviderStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.FooterProviderBg)).
		Foreground(lipgloss.Color(t.FooterBg)).
		Padding(0, 1)

	footerModelStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.FooterModelBg)).
		Foreground(lipgloss.Color(t.FooterBg)).
		Padding(0, 1)

	footerStatStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.FooterWizardBg)).
		Foreground(lipgloss.Color(t.Text)).
		Padding(0, 1)
}

func init() {
	UpdateStyles(theme.DefaultDracula())
}

func (m model) renderMessage(role, content string) string {
	if content == "" {
		return ""
	}

	prefix := "👤 "
	style := userStyle

	switch role {
	case "wizard":
		prefix = "🧙‍♂️ "
		style = wizardStyle
	case "command":
		prefix = "⌨️ "
		style = commandStyle
	}

	// Intercept and format code blocks
	if m.codeFormatter != nil {
		content = codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
			submatch := codeBlockRegex.FindStringSubmatch(match)
			if len(submatch) >= 3 {
				lang := submatch[1]
				code := submatch[2]
				logger.Debug("🎨 [TUI] Formatando bloco de código", "lang", lang, "len", len(code))
				formatted := m.codeFormatter(lang, code)
				return "```" + lang + "\n" + formatted + "\n```"
			}
			return match
		})
	}

	rendered, err := m.renderer.Render(content)
	if err != nil {
		logger.Error("❌ [TUI] Erro ao renderizar Markdown", err)
		return style.Render(prefix + strings.TrimSpace(content)) // Fallback to raw content
	}

	return style.Render(prefix + strings.TrimSpace(rendered))
}
