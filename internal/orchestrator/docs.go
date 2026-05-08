package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"spec-wizard/internal/patterns"
	"strings"
)

// GeneratePRD generates the content of PRD.md (Requirements and Domain)
func GeneratePRD(config ProjectConfig) string {
	var sb strings.Builder
	sb.WriteString("# 📄 PRODUCT REQUIREMENT DOCUMENT (PRD)\n\n")
	sb.WriteString("## 🎯 Identity & Purpose\n")
	sb.WriteString(fmt.Sprintf("- **Project**: %s\n", config.ProjectName))
	sb.WriteString(fmt.Sprintf("- **Domain**: %s\n\n", config.Domain))

	sb.WriteString("## 🚀 Business Requirements\n")
	sb.WriteString("### ✅ Functional\n")
	sb.WriteString(config.FunctionalRequirements + "\n\n")
	sb.WriteString("### ⚙️ Non-Functional\n")
	sb.WriteString(config.NonFunctionalRequirements + "\n\n")

	return sb.String()
}

// GenerateSPEC generates the content of SPEC.md (Architecture and Stack)
func GenerateSPEC(config ProjectConfig, allRules map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("# 🏗️ SYSTEM SPECIFICATION (SPEC)\n\n")
	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", config.Language))
	sb.WriteString(fmt.Sprintf("- **Core Architecture**: %s\n", config.Architecture))
	sb.WriteString(fmt.Sprintf("- **Data Strategy**: %s\n", config.DataStrategy))
	sb.WriteString(fmt.Sprintf("- **State Management**: %s\n", config.StateManagement))
	sb.WriteString(fmt.Sprintf("- **API Contract**: %s\n\n", config.ApiContract))

	sb.WriteString("## 📂 Patterns & Standards\n")
	for _, p := range config.Patterns {
		sb.WriteString(fmt.Sprintf("- %s\n", p))
	}

	if len(allRules) > 0 {
		sb.WriteString("\n## 🛠️ Architecture Constraints\n")
		for pattern, rules := range allRules {
			sb.WriteString(fmt.Sprintf("### %s\n", pattern))
			for _, rule := range rules {
				sb.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			sb.WriteString("\n")
		}
	}

	if config.Customization != "" {
		sb.WriteString("\n### 🛠️ Technical Customizations\n")
		sb.WriteString(config.Customization + "\n")
	}

	return sb.String()
}

// GenerateSkills generates the content of skills.md (Golden Rules)
func GenerateSkills(config ProjectConfig, allRules map[string][]string) string {
	var sb strings.Builder
	sb.WriteString("# 📏 SKILLS & GOLDEN RULES\n\n")
	sb.WriteString("> Imperative rules that must be followed for every task.\n\n")

	if config.Instructions != "" {
		sb.WriteString("## 📝 Custom Instructions\n")
		sb.WriteString(config.Instructions + "\n\n")
	}

	if len(allRules) > 0 {
		sb.WriteString("## ⚓ Core Technical Rules\n")
		for pattern, rules := range allRules {
			sb.WriteString(fmt.Sprintf("### %s\n", pattern))
			for _, rule := range rules {
				sb.WriteString(fmt.Sprintf("- %s\n", rule))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// GenerateSummary generates the executive summary SUMMARY.md
func GenerateSummary(config ProjectConfig) string {
	var sb strings.Builder
	sb.WriteString("# 📋 EXECUTIVE IMPLEMENTATION SUMMARY\n\n")
	sb.WriteString(fmt.Sprintf("## 🚀 Project: %s\n\n", config.ProjectName))

	sb.WriteString("### 🎯 Primary Objective\n")
	sb.WriteString(config.Domain + "\n\n")

	sb.WriteString("### 🏗️ Architecture & Stack\n")
	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", config.Language))
	sb.WriteString(fmt.Sprintf("- **Architecture**: %s\n", config.Architecture))
	sb.WriteString(fmt.Sprintf("- **State**: %s\n", config.StateManagement))
	sb.WriteString(fmt.Sprintf("- **Data**: %s\n\n", config.DataStrategy))

	sb.WriteString("### 🏁 Roadmap Progress\n")
	sb.WriteString("Refer to the [roadmap.md](./roadmap.md) file for sprint details.\n")

	return sb.String()
}

// GenerateWikiHome generates the Wiki Home
func GenerateWikiHome(config ProjectConfig) string {
	return fmt.Sprintf(`# 🏠 Project Wiki: %s

Welcome to the centralized project documentation. This space serves as the long-term memory for human developers and AI agents.

## 📌 Starting Point
- **[PRD.md](../PRD.md)**: Business Requirements.
- **[SPEC.md](../SPEC.md)**: Technical Specification.
- **[skills.md](../skills.md)**: Golden Rules.
- **[SUMMARY.md](../SUMMARY.md)**: Executive Summary.
- **[Execution Roadmap](../roadmap.md)**: What we are building and what has already been done.

## 🏗️ General Architecture
This system was designed under the **%s** paradigm with a focus on **%s**.
`, config.ProjectName, config.Architecture, config.Domain)
}

// GenerateRoadmapMarkdown converts the structured roadmap to Markdown
func GenerateRoadmapMarkdown(rawRoadmap interface{}) string {
	var roadmap ProjectRoadmap

	// Try to convert interface{} to ProjectRoadmap
	data, _ := json.Marshal(rawRoadmap)
	if err := json.Unmarshal(data, &roadmap); err != nil {
		return "# 🗺️ Project Roadmap\n\nError processing the structured roadmap."
	}

	var sb strings.Builder
	sb.WriteString("# 🗺️ STRATEGIC EXECUTION ROADMAP\n\n")
	sb.WriteString("> This document details the development phases and the current progress of the system.\n\n")

	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", roadmap.Language))
	sb.WriteString(fmt.Sprintf("- **Pattern**: %s\n\n", roadmap.Pattern))

	for _, sprint := range roadmap.Sprints {
		sb.WriteString(fmt.Sprintf("## Sprint %d: %s\n", sprint.ID, sprint.Goal))
		for _, task := range sprint.Tasks {
			statusIcon := "⏳"
			if task.Status == "completed" {
				statusIcon = "✅"
			} else if task.Status == "in_progress" {
				statusIcon = "🚀"
			} else if task.Status == "failed" {
				statusIcon = "❌"
			}

			priorityTag := ""
			if task.Priority != "" {
				priorityTag = fmt.Sprintf(" `[%s]`", task.Priority)
			}

			sb.WriteString(fmt.Sprintf("### %s %s%s\n", statusIcon, task.Title, priorityTag))
			sb.WriteString(task.Description + "\n\n")

			if len(task.AcceptanceCriteria) > 0 {
				sb.WriteString("**Acceptance Criteria:**\n")
				for _, crit := range task.AcceptanceCriteria {
					sb.WriteString(fmt.Sprintf("- [ ] %s\n", crit))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("\n*End of Strategic Roadmap*")
	return sb.String()
}

// SaveAgentsDocs persiste toda a documentação em .spec-wizard/
func (o *Orchestrator) SaveAgentsDocs(config ProjectConfig, patternRepo *patterns.PatternRepository) error {
	wizardPath := filepath.Join(o.ProjectPath, ".spec-wizard")
	os.MkdirAll(filepath.Join(wizardPath, "wiki"), 0755)
	os.MkdirAll(filepath.Join(wizardPath, "task-logs"), 0755)

	allRules := make(map[string][]string)
	if patternRepo != nil {
		allRules = patternRepo.GetMultiplePatternRules(config.Language, config.Patterns)
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(filepath.Join(wizardPath, "config.json"), configData, 0644)

	os.WriteFile(filepath.Join(wizardPath, "PRD.md"), []byte(GeneratePRD(config)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "SPEC.md"), []byte(GenerateSPEC(config, allRules)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "skills.md"), []byte(GenerateSkills(config, allRules)), 0644)
	os.WriteFile(filepath.Join(wizardPath, "SUMMARY.md"), []byte(GenerateSummary(config)), 0644)

	wikiContent := GenerateWikiHome(config)
	os.WriteFile(filepath.Join(wizardPath, "wiki", "home.md"), []byte(wikiContent), 0644)

	if config.Roadmap != nil {
		roadmapMD := GenerateRoadmapMarkdown(config.Roadmap)
		os.WriteFile(filepath.Join(wizardPath, "roadmap.md"), []byte(roadmapMD), 0644)
	}

	// Limpeza legada
	os.Remove(filepath.Join(wizardPath, "project_config.json"))
	os.Remove(filepath.Join(wizardPath, "sprints.json"))

	return nil
}

// FinalizeAnchoring monta o conteúdo do skills.md com regras imperativas
func (o *Orchestrator) FinalizeAnchoring(config ProjectConfig, patternRepo *patterns.PatternRepository) error {
	allRules := patternRepo.GetMultiplePatternRules(config.Language, config.Patterns)
	var content strings.Builder
	content.WriteString("# Skills & Golden Rules - " + strings.Title(config.Language) + "\n\n")

	if config.ProjectName != "" {
		content.WriteString("## Contexto do Projeto: " + config.ProjectName + "\n")
	}

	// ... (Simplificado para brevidade, mas mantendo a lógica de regras)
	for patternName, rules := range allRules {
		content.WriteString("## " + patternName + "\n")
		for _, rule := range rules {
			content.WriteString("- " + rule + "\n")
		}
		content.WriteString("\n")
	}

	skillsFile := filepath.Join(o.ProjectPath, ".spec-wizard", "skills.md")
	return os.WriteFile(skillsFile, []byte(content.String()), 0644)
}

// readProjectDoc lê um documento da pasta .spec-wizard do projeto
func (o *Orchestrator) readProjectDoc(root, filename string) (string, error) {
	path := filepath.Join(root, ".spec-wizard", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("[Documento %s não encontrado]", filename), err
	}
	return string(data), nil
}

// isEmpty utilitário para verificar se um valor de interface é vazio
func isEmpty(val interface{}) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return v == "" || v == "custom" || v == "Domínio não identificado"
	case []interface{}:
		return len(v) == 0
	case []string:
		return len(v) == 0
	}
	return false
}
