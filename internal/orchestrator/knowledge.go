package orchestrator

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type KnowledgeManager struct {
	ProjectPath string
	WizardPath  string
	Normalizer  Normalizer
}

func NewKnowledgeManager(projectPath string) *KnowledgeManager {
	return &KnowledgeManager{
		ProjectPath: projectPath,
		WizardPath:  filepath.Join(projectPath, ".spec-wizard"),
		Normalizer:  &CentralNormalizer{},
	}
}

// IngestFile processa um arquivo local (PDF, Excel, etc)
func (km *KnowledgeManager) IngestFile(fileName string, r io.Reader) (*KnowledgeSource, error) {
	rawDir := filepath.Join(km.WizardPath, "knowledge", "raw")
	os.MkdirAll(rawDir, 0755)

	filePath := filepath.Join(rawDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return nil, err
	}

	source := KnowledgeSource{
		ID:           fmt.Sprintf("kb-%d", time.Now().Unix()),
		Type:         "file",
		Name:         fileName,
		Path:         filePath,
		Status:       "pending",
		LastModified: time.Now().Format(time.RFC3339),
	}

	// Inicia processamento assíncrono
	go km.ProcessSource(source)

	return &source, nil
}

// IngestLink processa uma URL
func (km *KnowledgeManager) IngestLink(url string) (*KnowledgeSource, error) {
	source := KnowledgeSource{
		ID:           fmt.Sprintf("kb-%d", time.Now().Unix()),
		Type:         "link",
		Name:         url,
		URL:          url,
		Status:       "pending",
		LastModified: time.Now().Format(time.RFC3339),
	}

	go km.ProcessSource(source)

	return &source, nil
}

// ProcessSource utiliza o normalizador central para transformar o dado em Markdown
func (km *KnowledgeManager) ProcessSource(source KnowledgeSource) {
	processedDir := filepath.Join(km.WizardPath, "knowledge", "processed")
	os.MkdirAll(processedDir, 0755)

	outputFileName := fmt.Sprintf("%s.md", source.ID)
	outputPath := filepath.Join(processedDir, outputFileName)

	fmt.Printf("🔄 [Knowledge] Processando %s centralmente...\n", source.Name)
	err := km.Normalizer.Normalize(source, outputPath)

	// Atualiza o status no config.json do projeto
	km.updateSourceStatus(source.ID, err)
}

func (km *KnowledgeManager) updateSourceStatus(id string, err error) {
	configPath := filepath.Join(km.WizardPath, "config.json")
	data, _ := os.ReadFile(configPath)
	var config ProjectConfig
	json.Unmarshal(data, &config)

	for i, s := range config.KnowledgeBase {
		if s.ID == id {
			if err != nil {
				config.KnowledgeBase[i].Status = "failed"
			} else {
				config.KnowledgeBase[i].Status = "processed"
			}
			config.KnowledgeBase[i].LastModified = time.Now().Format(time.RFC3339)
			break
		}
	}

	newData, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, newData, 0644)
}

func (km *KnowledgeManager) GetRelevantContext(taskTitle, taskDesc string) string {
	processedDir := filepath.Join(km.WizardPath, "knowledge", "processed")
	files, err := os.ReadDir(processedDir)
	if err != nil {
		return ""
	}

	var relevantSnippets []string
	keywords := strings.Fields(strings.ToLower(taskTitle + " " + taskDesc))

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			content, err := os.ReadFile(filepath.Join(processedDir, file.Name()))
			if err != nil {
				continue
			}

			// Busca simples por keywords
			// Futuro: Usar embeddings ou mini-LLM call para decidir relevância
			found := false
			lowerContent := strings.ToLower(string(content))
			for _, kw := range keywords {
				if len(kw) > 3 && strings.Contains(lowerContent, kw) {
					found = true
					break
				}
			}

			if found {
				// Limita o tamanho do snippet para não estourar o prompt
				snippet := string(content)
				if len(snippet) > 2000 {
					snippet = snippet[:2000] + "... [Truncated]"
				}
				relevantSnippets = append(relevantSnippets, fmt.Sprintf("### Knowledge Ref: %s\n%s", file.Name(), snippet))
			}
		}
	}

	if len(relevantSnippets) == 0 {
		return ""
	}

	return "\n## 📚 Relevant Knowledge Base Snippets\n" + strings.Join(relevantSnippets, "\n\n")
}
