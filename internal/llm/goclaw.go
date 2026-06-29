package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"spec-wizard/internal/logger"
	"strings"

	"github.com/nextlevelbuilder/goclaw/pkg/providers"
	"sync"
)

var (
	endpointLocks = make(map[string]*sync.Mutex)
	locksMu       sync.Mutex
)

func getEndpointLock(url string) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	
	// Normaliza a URL para chave do mapa
	key := strings.TrimRight(strings.ToLower(url), "/")
	if l, ok := endpointLocks[key]; ok {
		return l
	}
	l := &sync.Mutex{}
	endpointLocks[key] = l
	return l
}

// GoclawProvider implements the llm.Provider interface using the goclaw engine.
type GoclawProvider struct {
	provider   providers.Provider
	name       string
	baseURL    string
	mu         *sync.Mutex
	sequential bool
}

func NewGoclawProvider(name, baseURL, apiKey, defaultModel string, sequential bool) (*GoclawProvider, error) {
	var p providers.Provider

	nameLower := strings.ToLower(name)
	switch nameLower {
	case "anthropic":
		p = providers.NewAnthropicProvider(apiKey,
			providers.WithAnthropicBaseURL(baseURL),
			providers.WithAnthropicModel(defaultModel),
			providers.WithAnthropicName(nameLower),
		)
	case "openai", "lmstudio", "deepseek", "gemini":
		p = providers.NewOpenAIProvider(nameLower, apiKey, baseURL, defaultModel)
	case "openrouter":
		op := providers.NewOpenAIProvider(nameLower, apiKey, baseURL, defaultModel)
		// OpenRouter requer headers de identificação para alguns modelos e rankings
		op.WithSiteInfo("https://github.com/upperxcode/spec-wizard", "Spec Wizard")
		p = op
	default:
		p = providers.NewOpenAIProvider(nameLower, apiKey, baseURL, defaultModel)
	}

	logger.Debug("🔌 [Goclaw] Provedor instanciado", "name", nameLower, "baseURL", baseURL)

	return &GoclawProvider{
		provider:   p,
		name:       nameLower,
		baseURL:    baseURL,
		mu:         getEndpointLock(baseURL),
		sequential: sequential,
	}, nil
}

func (p *GoclawProvider) Name() string {
	return "goclaw-" + p.provider.Name()
}

func (p *GoclawProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	// 1. Converte mensagens para o formato goclaw
	var goclawMessages []providers.Message
	for _, m := range messages {
		msg := providers.Message{
			Role:    m.Role,
			Content: m.Content,
		}

		// Converte ToolCalls se houver
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				var args map[string]any
				if tc.Function.Arguments != "" {
					json.Unmarshal([]byte(tc.Function.Arguments), &args)
				}
				msg.ToolCalls = append(msg.ToolCalls, providers.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: args,
				})
			}
		}

		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}

		goclawMessages = append(goclawMessages, msg)
	}

	// 2. Converte ferramentas para o formato goclaw
	var goclawTools []providers.ToolDefinition
	for _, t := range tools {
		dt := providers.ToolDefinition{
			Type: "function",
			Function: &providers.ToolFunctionSchema{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		}
		goclawTools = append(goclawTools, dt)
	}

	// 3. Cria o Request
	req := providers.ChatRequest{
		Model:    model,
		Messages: goclawMessages,
		Tools:    goclawTools,
	}

	// Calcula tamanhos para telemetria
	totalMsgChars := 0
	for _, m := range goclawMessages {
		totalMsgChars += len(m.Content)
	}
	toolsSize, _ := json.Marshal(goclawTools)

	logger.Debug("🚀 [Goclaw] Enviando requisição",
		"model", model,
		"messages_count", len(goclawMessages),
		"total_msg_chars", totalMsgChars,
		"tools_payload_size", len(toolsSize))

	logger.Info("⏳ [Goclaw] Aguardando resposta do provedor...", "model", model)

	// 4. Executa o Chat com Proteção de Fila Opcional (Lock)
	var resp *providers.ChatResponse
	var err error
	if p.sequential {
		func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			resp, err = p.provider.Chat(ctx, req)
		}()
	} else {
		resp, err = p.provider.Chat(ctx, req)
	}

	if err != nil {
		logger.Error("❌ [Goclaw] Falha na requisição", err)
		return nil, err
	}

	logger.Debug("✅ [Goclaw] Resposta recebida", "content_length", len(resp.Content))

	if len(resp.Content) == 0 && len(resp.ToolCalls) == 0 {
		msg := "a IA retornou uma resposta vazia"
		if p.name == "lmstudio" || p.name == "ollama" {
			msg = fmt.Sprintf("a IA (%s) retornou uma resposta vazia. Verifique se o modelo está carregado no provedor e se a URL (%s) está correta.", p.name, p.baseURL)
		}
		logger.Warn("⚠️ [Goclaw] " + msg)
		return nil, errors.New(msg)
	}

	// 5. Converte resposta de volta
	var toolCalls []ToolCall
	for _, tc := range resp.ToolCalls {
		argsBytes, _ := json.Marshal(tc.Arguments)
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Name,
				Arguments: string(argsBytes),
			},
		})
	}

	response := &Response{
		Role:      "assistant",
		Content:   resp.Content,
		ToolCalls: toolCalls,
	}

	if resp.Usage != nil {
		response.Usage = Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return response, nil
}
