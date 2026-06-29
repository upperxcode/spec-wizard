package utils

import (
	"os"
	"path/filepath"
)

// InferLanguage tenta detectar a linguagem de programação predominante no diretório
func InferLanguage(projectPath string) string {
	// Mapeamento de arquivos para linguagens
	indicators := map[string]string{
		"go.mod":           "go",
		"pubspec.yaml":     "flutter",
		"package.json":     "javascript",
		"requirements.txt":  "python",
		"pyproject.toml":   "python",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"Cargo.toml":       "rust",
		"composer.json":    "php",
		"Gemfile":          "ruby",
		"Makefile":         "c/c++",
	}

	for file, lang := range indicators {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			return lang
		}
	}

	return ""
}
