package orchestrator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/xuri/excelize/v2"
)

// Normalizer define a interface para transformar fontes de dados em Markdown
type Normalizer interface {
	Normalize(source KnowledgeSource, outputPath string) error
}

// CentralNormalizer é a implementação padrão e central do Spec Wizard
type CentralNormalizer struct{}

func (n *CentralNormalizer) Normalize(source KnowledgeSource, outputPath string) error {
	os.MkdirAll(filepath.Dir(outputPath), 0755)

	switch source.Type {
	case "file":
		return n.normalizeFile(source.Path, outputPath)
	case "link":
		return n.normalizeLink(source.URL, outputPath)
	default:
		return fmt.Errorf("tipo de fonte desconhecido: %s", source.Type)
	}
}

func (n *CentralNormalizer) normalizeFile(inputPath, outputPath string) error {
	ext := strings.ToLower(filepath.Ext(inputPath))
	switch ext {
	case ".xlsx", ".xls":
		return n.processExcel(inputPath, outputPath)
	case ".csv":
		return n.processCSV(inputPath, outputPath)
	case ".pdf":
		// Para PDF, por enquanto, usaremos um fallback ou uma lib Go.
		// Note: PDFs são complexos em Go puro sem CGO.
		return n.processGenericFile(inputPath, outputPath, "PDF")
	default:
		return n.processGenericFile(inputPath, outputPath, "Text")
	}
}

func (n *CentralNormalizer) normalizeLink(url, outputPath string) error {
	fmt.Printf("🌐 [Normalizer] Baixando conteúdo de: %s\n", url)
	
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("erro ao acessar URL: %s (Status: %d)", url, resp.StatusCode)
	}

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(string(html))
	if err != nil {
		return err
	}

	content := fmt.Sprintf("# Source: %s\n\n%s", url, markdown)
	return os.WriteFile(outputPath, []byte(content), 0644)
}

func (n *CentralNormalizer) processExcel(inputPath, outputPath string) error {
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Data Source: %s\n\n", filepath.Base(inputPath)))

	for _, name := range f.GetSheetList() {
		sb.WriteString(fmt.Sprintf("## Sheet: %s\n\n", name))
		rows, err := f.GetRows(name)
		if err != nil {
			continue
		}

		for _, row := range rows {
			sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
		}
		sb.WriteString("\n")
	}

	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

func (n *CentralNormalizer) processCSV(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	
	// Simples conversão de CSV para tabela MD
	lines := strings.Split(string(data), "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Data Source: %s\n\n", filepath.Base(inputPath)))
	
	for _, line := range lines {
		if strings.TrimSpace(line) == "" { continue }
		cols := strings.Split(line, ",")
		sb.WriteString("| " + strings.Join(cols, " | ") + " |\n")
	}
	
	return os.WriteFile(outputPath, []byte(sb.String()), 0644)
}

func (n *CentralNormalizer) processGenericFile(inputPath, outputPath string, typeLabel string) error {
	// Fallback para arquivos que não temos parser nativo robusto ainda
	// Poderíamos usar o LLM para extrair o texto se for pequeno, 
	// ou apenas ler o conteúdo bruto se for texto.
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("# [%s Source] %s\n\nProcessed at: %s\n\n%s", 
		typeLabel, filepath.Base(inputPath), time.Now().Format(time.RFC3339), string(data))
	
	return os.WriteFile(outputPath, []byte(content), 0644)
}
