package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TranslationMap armazena as chaves de tradução por idioma
var TranslationMap = make(map[string]map[string]string)

func init() {
	LoadTranslations()
}

// LoadTranslations carrega as traduções dos arquivos JSON em configs/i18n
func LoadTranslations() {
	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".local/share/spec-wizard/configs/i18n")

	searchPaths := []string{"configs/i18n", "../configs/i18n", "../../configs/i18n", globalPath}

	found := false
	for _, p := range searchPaths {
		files, err := os.ReadDir(p)
		if err != nil {
			continue
		}

		for _, f := range files {
			if filepath.Ext(f.Name()) == ".json" {
				lang := strings.TrimSuffix(f.Name(), ".json")
				content, err := os.ReadFile(filepath.Join(p, f.Name()))
				if err != nil {
					fmt.Printf("⚠️ Erro ao ler tradução %s: %v\n", f.Name(), err)
					continue
				}

				var translations map[string]string
				if err := json.Unmarshal(content, &translations); err != nil {
					fmt.Printf("⚠️ Erro ao decodificar tradução %s: %v\n", f.Name(), err)
					continue
				}

				TranslationMap[lang] = translations
				found = true
			}
		}
		if found {
			break
		}
	}

	if !found {
		fmt.Println("⚠️ Nenhuma tradução carregada. Usando mapas vazios.")
	}
}

// T traduz uma chave para o idioma especificado
func T(lang, key string, args ...interface{}) string {
	if lang == "" {
		lang = "pt-BR" // Default
	}

	// Tenta o idioma exato
	translations, ok := TranslationMap[lang]
	if !ok {
		// Tenta o prefixo (ex: pt-BR -> pt)
		prefix := strings.Split(lang, "-")[0]
		translations, ok = TranslationMap[prefix]
		if !ok {
			translations = TranslationMap["pt-BR"] // Fallback final
		}
	}

	msg, ok := translations[key]
	if !ok {
		// Se não achar a chave, tenta no pt-BR como fallback
		if ptTrans, exists := TranslationMap["pt-BR"]; exists {
			msg, ok = ptTrans[key]
		}

		if !ok {
			return key // Se nem no pt-BR achar, retorna a chave
		}
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
