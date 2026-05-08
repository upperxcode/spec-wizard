package llm

import (
	"context"
	"encoding/json"
	"spec-wizard/internal/logger"
	"strings"

	"github.com/nextlevelbuilder/goclaw/pkg/providers"
)

// GoclawProvider implements the llm.Provider interface using the goclaw engine.
type GoclawProvider struct {
	provider providers.Provider
}

func NewGoclawProvider(name, baseURL, apiKey, defaultModel string) (*GoclawProvider, error) {
	var p providers.Provider

	switch strings.ToLower(name) {
	case "anthropic":
		p = providers.NewAnthropicProvider(apiKey,
			providers.WithAnthropicBaseURL(baseURL),
			providers.WithAnthropicModel(defaultModel),
			providers.WithAnthropicName(name),
		)
	case "openai", "lmstudio", "openrouter", "deepseek":
		p = providers.NewOpenAIProvider(name, apiKey, baseURL, defaultModel)
	default:
		p = providers.NewOpenAIProvider(name, apiKey, baseURL, defaultModel)
	}

	return &GoclawProvider{provider: p}, nil
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

	logger.Debug("🚀 [Goclaw] Enviando requisição", "model", model, "messages_count", len(goclawMessages))

	// 4. Executa o Chat
	resp, err := p.provider.Chat(ctx, req)
	if err != nil {
		logger.Error("❌ [Goclaw] Falha na requisição", err)
		return nil, err
	}

	logger.Debug("✅ [Goclaw] Resposta recebida", "content_length", len(resp.Content))

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

	return &Response{
		Role:      "assistant",
		Content:   resp.Content,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}
