package adatools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolDefinition representa o schema de uma ferramenta para o provedor de IA
type ToolDefinition struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// ToolResult representa o resultado da execução de uma ferramenta
type ToolResult struct {
	Content string
	Error   error
}

// ToolFunc é a assinatura para funções de ferramentas
type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

// Registry gerencia as ferramentas disponíveis no sistema
type Registry struct {
	mu    sync.RWMutex
	tools map[string]toolEntry
	root  string
}

type toolEntry struct {
	definition ToolDefinition
	handler    ToolFunc
}

// NewRegistry cria um novo registro de ferramentas
func NewRegistry(root string) *Registry {
	return &Registry{
		tools: make(map[string]toolEntry),
		root:  root,
	}
}

// SetRoot atualiza o diretório raiz para ferramentas de sistema de arquivos
func (r *Registry) SetRoot(root string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.root = root
}

// Root retorna o diretório raiz atual
func (r *Registry) Root() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.root
}

// Register adiciona uma nova ferramenta ao registro
func (r *Registry) Register(name, description string, paramsSchema string, handler ToolFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	def := ToolDefinition{
		Type: "function",
	}
	def.Function.Name = name
	def.Function.Description = description
	def.Function.Parameters = json.RawMessage(paramsSchema)

	r.tools[name] = toolEntry{
		definition: def,
		handler:    handler,
	}
}

// ToProviderDefs retorna as definições no formato esperado pelos provedores de LLM
func (r *Registry) ToProviderDefs() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []ToolDefinition
	for _, t := range r.tools {
		defs = append(defs, t.definition)
	}
	return defs
}

// List retorna os nomes das ferramentas registradas
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute executa uma ferramenta pelo nome com os argumentos fornecidos
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	r.mu.RLock()
	entry, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("ferramenta '%s' não encontrada", name)
	}

	return entry.handler(ctx, args)
}
