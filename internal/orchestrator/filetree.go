package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateFileTree gera uma representação visual da estrutura do projeto
func (o *Orchestrator) generateFileTree(excludePatterns []string) string {
	var sb strings.Builder
	absPath, _ := filepath.Abs(o.ProjectPath)
	sb.WriteString(fmt.Sprintf("PROJECT ROOT: %s\n", filepath.Base(absPath)))

	count := 0
	// Inicia a caminhada na raiz do projeto com profundidade 0
	o.walkDir(o.ProjectPath, "", 0, &sb, excludePatterns, &count)

	if count >= 1000 {
		sb.WriteString("\n... (lista de arquivos truncada por tamanho excessivo)\n")
	}

	return sb.String()
}

// walkDir percorre os diretórios recursivamente respeitando limites e filtros
func (o *Orchestrator) walkDir(path, indent string, depth int, sb *strings.Builder, excludePatterns []string, count *int) {
	// Limite de segurança para não explodir o contexto
	if *count >= 1000 {
		return
	}

	// Limite de profundidade para evitar árvores gigantescas e estouro de pilha
	if depth > 5 {
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	// Pré-filtra as entradas para saber qual será o último item VISÍVEL
	var visibleEntries []os.DirEntry
	for _, e := range entries {
		name := e.Name()

		// 1. Filtros Padrão (Hardcoded para pastas que NUNCA devem ir para a IA)
		if strings.HasPrefix(name, ".") && name != ".spec-wizard" {
			continue
		}

		if name == "node_modules" || name == "vendor" || name == "build" ||
			name == "bin" || name == "obj" || name == "ios" || name == "android" ||
			name == "dist" || name == ".next" || name == "target" || name == "venv" ||
			name == ".venv" || name == "__pycache__" || name == "coverage" ||
			name == "out" || name == ".terraform" || name == ".git" {
			continue
		}

		// 2. Filtros Dinâmicos (ExcludePatterns do projeto)
		shouldExclude := false
		for _, pattern := range excludePatterns {
			if matched, _ := filepath.Match(pattern, name); matched {
				shouldExclude = true
				break
			}
			if strings.Contains(name, pattern) {
				shouldExclude = true
				break
			}
		}
		if !shouldExclude {
			visibleEntries = append(visibleEntries, e)
		}
	}

	for i, f := range visibleEntries {
		if *count >= 1000 {
			return
		}
		*count++

		name := f.Name()
		isLast := i == len(visibleEntries)-1
		prefix := "├── "
		if isLast {
			prefix = "└── "
		}

		displaySize := ""
		if !f.IsDir() {
			if info, err := f.Info(); err == nil {
				displaySize = fmt.Sprintf(" (%d bytes)", info.Size())
			}
		}

		sb.WriteString(indent + prefix + name + displaySize + "\n")

		if f.IsDir() {
			newIndent := indent
			if isLast {
				newIndent += "    "
			} else {
				newIndent += "│   "
			}
			o.walkDir(filepath.Join(path, name), newIndent, depth+1, sb, excludePatterns, count)
		}
	}
}
