package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// generateFileTree gera uma representação visual da estrutura do projeto
func (o *Orchestrator) generateFileTree() string {
	var sb strings.Builder
	// Inicia a caminhada na raiz do projeto com profundidade 0
	o.walkDir(o.ProjectPath, "", 0, &sb)
	return sb.String()
}

// walkDir percorre os diretórios recursivamente respeitando limites e filtros
func (o *Orchestrator) walkDir(path, indent string, depth int, sb *strings.Builder) {
	// Limite de profundidade para evitar árvores gigantescas e estouro de pilha
	if depth > 6 {
		return
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for i, f := range files {
		name := f.Name()
		// Pula pastas ocultas e artefatos de build
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" ||
			name == "build" || name == "bin" || name == "obj" || name == "ios" || name == "android" {
			continue
		}

		isLast := i == len(files)-1
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
			o.walkDir(filepath.Join(path, name), newIndent, depth+1, sb)
		}
	}
}
