package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Theme define as cores e estilos de toda a interface Spec Wizard
type Theme struct {
	Name             string `json:"name"`
	WizardBorder     string `json:"wizard_border"`
	UserBorder       string `json:"user_border"`
	CommandBorder    string `json:"command_border"`
	Title            string `json:"title"`
	Project          string `json:"project"`
	FooterBg         string `json:"footer_bg"`
	FooterWizardBg   string `json:"footer_wizard_bg"`
	FooterModeBg     string `json:"footer_mode_bg"`
	FooterProviderBg string `json:"footer_provider_bg"`
	FooterModelBg    string `json:"footer_model_bg"`
	Text             string `json:"text"`
}

// DefaultDracula retorna o tema padrão baseado na paleta Dracula
func DefaultDracula() Theme {
	return Theme{
		Name:             "dracula",
		WizardBorder:     "#7D56F4",
		UserBorder:       "#04B575",
		CommandBorder:    "#FFCC00",
		Title:            "#7D56F4",
		Project:          "#04B575",
		FooterBg:         "#282A36",
		FooterWizardBg:   "#6272A4",
		FooterModeBg:     "#44475A",
		FooterProviderBg: "#BD93F9",
		FooterModelBg:    "#F8F8F2",
		Text:             "#F8F8F2",
	}
}

// LoadTheme tenta carregar um tema de um arquivo JSON
func LoadTheme(path string) (Theme, error) {
	var t Theme
	data, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	err = json.Unmarshal(data, &t)
	return t, err
}

// ListThemes lista todos os temas (.json) em um diretório
func ListThemes(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var themes []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			themes = append(themes, f.Name()[:len(f.Name())-5])
		}
	}
	return themes, nil
}
