package patterns

import "strings"

type Pattern struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Category         string   `json:"category"`
	Group            string   `json:"group"`
	Scope            string   `json:"scope"` // "global" ou "language"
	Description      string   `json:"description"`
	IncompatibleWith []string `json:"incompatibleWith"`
	GoldenRules      []string `json:"goldenRules"`
}

type PatternRepository struct {
	Store map[string][]Pattern
}

func NewRepository() *PatternRepository {
	repo := &PatternRepository{
		Store: make(map[string][]Pattern),
	}
	repo.setupDefaults()
	return repo
}

func (r *PatternRepository) setupDefaults() {
	// 1. Arquiteturas
	architectures := []Pattern{
		{ID: "clean_architecture", Name: "Clean Architecture", Category: "Architecture", Description: "Independência de frameworks e alta testabilidade."},
		{ID: "crud", Name: "CRUD", Category: "Architecture", Description: "Manipulação direta de entidades."},
		{ID: "event_sourcing", Name: "Event Sourcing", Category: "Architecture", Description: "Histórico de eventos imutáveis."},
		{ID: "mvc", Name: "MVC", Category: "Architecture", Description: "Coordenação via Controller."},
		{ID: "mvp", Name: "MVP", Category: "Architecture", Description: "Presenter e View passiva.", Scope: "mobile"},
		{ID: "mvi", Name: "MVI", Category: "Architecture", Description: "Fluxo de dados unidirecional.", Scope: "mobile"},
		{ID: "adr", Name: "ADR", Category: "Architecture", Description: "Action-Domain-Responder.", Scope: "backend"},
		{ID: "viper", Name: "VIPER", Category: "Architecture", Description: "Separação modular extrema.", Scope: "mobile"},
		{ID: "cqrs", Name: "CQRS", Category: "Architecture", Description: "Command Query Responsibility Segregation.", Scope: "backend"},
		{ID: "custom", Name: "Custom / Service Locator", Category: "Architecture", Description: "Estrutura personalizada."},
	}

	// 2. Filosofias (Global)
	philosophies := []Pattern{
		{ID: "solid", Name: "SOLID", Category: "Philosophy", Description: "Alta testabilidade e baixo acoplamento."},
		{ID: "dry", Name: "DRY", Category: "Philosophy", Description: "Evita duplicação de lógica."},
		{ID: "kiss", Name: "KISS", Category: "Philosophy", Description: "Mantenha o código simples."},
		{ID: "yagni", Name: "YAGNI", Category: "Philosophy", Description: "Não implemente o que não é necessário."},
	}

	// 3. Design Patterns (Global)
	designPatterns := []Pattern{
		{ID: "factory", Name: "Factory Method", Category: "DesignPattern", Group: "Creational"},
		{ID: "builder", Name: "Builder", Category: "DesignPattern", Group: "Creational"},
		{ID: "singleton", Name: "Singleton", Category: "DesignPattern", Group: "Creational"},
		{ID: "adapter", Name: "Adapter", Category: "DesignPattern", Group: "Structural"},
		{ID: "facade", Name: "Facade", Category: "DesignPattern", Group: "Structural"},
		{ID: "observer", Name: "Observer", Category: "DesignPattern", Group: "Behavioral"},
		{ID: "strategy", Name: "Strategy", Category: "DesignPattern", Group: "Behavioral"},
	}

	// 4. Data Patterns
	dataPatterns := []Pattern{
		{ID: "repository", Name: "Repository", Category: "Data", Group: "Access"},
		{ID: "dao", Name: "DAO", Category: "Data", Group: "Access"},
		{ID: "dto", Name: "DTO", Category: "Data", Group: "Representation"},
		{ID: "entity", Name: "Entity", Category: "Data", Group: "Representation"},
		{ID: "active_record", Name: "Active Record", Category: "Data", Group: "Access", Scope: "backend"},
	}

	// 5. State Management (O mais sensível a linguagem)
	stateManagements := []Pattern{
		// Mobile Specific
		{ID: "bloc", Name: "BLoC", Category: "StateManagement", Scope: "mobile"},
		{ID: "provider", Name: "Provider", Category: "StateManagement", Scope: "mobile"},
		{ID: "riverpod", Name: "Riverpod", Category: "StateManagement", Scope: "mobile"},
		{ID: "getx", Name: "GetX", Category: "StateManagement", Scope: "mobile"},
		{ID: "mobx", Name: "MobX", Category: "StateManagement", Scope: "mobile"},
		{ID: "signals", Name: "Signals", Category: "StateManagement", Scope: "mobile"},

		// Web Specific
		{ID: "redux", Name: "Redux", Category: "StateManagement", Scope: "web"},
		{ID: "context_api", Name: "Context API", Category: "StateManagement", Scope: "web"},
		{ID: "vuex", Name: "Vuex / Pinia", Category: "StateManagement", Scope: "web"},

		// Generic
		{ID: "none", Name: "None / Custom", Category: "StateManagement"},
	}

	// 6. Data Strategy (Global)
	dataStrategies := []Pattern{
		{ID: "sql", Name: "Relational (SQL)", Category: "DataStrategy"},
		{ID: "nosql", Name: "Non-Relational (NoSQL)", Category: "DataStrategy"},
		{ID: "remote", Name: "Remote Only (API)", Category: "DataStrategy"},
		{ID: "custom", Name: "Custom / Mixed", Category: "DataStrategy"},
	}

	allPatterns := [][]Pattern{architectures, philosophies, designPatterns, dataPatterns, stateManagements, dataStrategies}

	supportedLangs := []string{"flutter", "dart", "go", "python", "javascript", "typescript"}

	for _, lang := range supportedLangs {
		var langPatterns []Pattern

		// Determina o tipo de linguagem
		isMobile := lang == "flutter" || lang == "dart"
		isWeb := lang == "javascript" || lang == "typescript"
		isBackend := lang == "go" || lang == "python"

		for _, group := range allPatterns {
			for _, p := range group {
				// Regras de filtragem
				if p.Scope == "" {
					langPatterns = append(langPatterns, p)
				} else if p.Scope == "mobile" && isMobile {
					langPatterns = append(langPatterns, p)
				} else if p.Scope == "web" && (isWeb || isMobile) { // Flutter as vezes usa padrões web
					langPatterns = append(langPatterns, p)
				} else if p.Scope == "backend" && isBackend {
					langPatterns = append(langPatterns, p)
				}
			}
		}
		r.Store[lang] = langPatterns
	}
}

func (r *PatternRepository) GetMultiplePatternRules(lang string, patternIDs []string) map[string][]string {
	result := make(map[string][]string)
	langPatterns := r.Store[strings.ToLower(lang)]

	for _, id := range patternIDs {
		for _, p := range langPatterns {
			if p.ID == id {
				result[p.Name] = p.GoldenRules
				break
			}
		}
	}
	return result
}
