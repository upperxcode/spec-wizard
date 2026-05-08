package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"spec-wizard/internal/analytic"
	"spec-wizard/internal/logger"
	adatools "spec-wizard/tools"
	"strings"
	"time"
)

var (
	ErrEmptyResponse = fmt.Errorf("a IA retornou uma resposta vazia")
)

type LLMClient struct {
	Provider     Provider
	Model        string
	SystemPrompt string
	Timeout      time.Duration
	Registry     interface {
		ToProviderDefs() []adatools.ToolDefinition
		Execute(ctx context.Context, name string, args map[string]any) (string, error)
	}
	OnProgress func(status string)
}

func NewLLMClient(provider Provider, model string, registry any) *LLMClient {
	var reg interface {
		ToProviderDefs() []adatools.ToolDefinition
		Execute(ctx context.Context, name string, args map[string]any) (string, error)
	}
	if r, ok := registry.(interface {
		ToProviderDefs() []adatools.ToolDefinition
		Execute(ctx context.Context, name string, args map[string]any) (string, error)
	}); ok {
		reg = r
	}

	return &LLMClient{
		Provider: provider,
		Model:    model,
		Timeout:  240 * time.Second,
		Registry: reg,
	}
}

// NewProvider is a factory for LLM providers
func NewProvider(name, baseURL, apiKey, envKey string) Provider {
	// 1. Fallbacks de URL padrão se estiver vazio
	if baseURL == "" {
		switch name {
		case "openai":
			baseURL = "https://api.openai.com/v1"
		case "ollama":
			baseURL = "http://localhost:11434"
		case "openrouter":
			baseURL = "https://openrouter.ai/api/v1"
		}
	}

	// 2. Normalizações específicas (ex: OpenRouter precisa de /v1)
	if name == "openrouter" {
		if baseURL == "https://openrouter.ai/api" {
			baseURL = "https://openrouter.ai/api/v1"
		}
		if !strings.HasSuffix(baseURL, "/v1") && !strings.HasSuffix(baseURL, "/v1/") {
			if strings.HasSuffix(baseURL, "/api") {
				baseURL += "/v1"
			} else if !strings.Contains(baseURL, "/v1") {
				baseURL = strings.TrimSuffix(baseURL, "/") + "/v1"
			}
		}
	}

	// 3. Busca API Key em variáveis de ambiente se estiver vazia
	if apiKey == "" {
		// 3.1. Tenta pela chave de ambiente configurada
		if envKey != "" {
			apiKey = os.Getenv(envKey)
		}

		// 3.2. Fallbacks automáticos se ainda estiver vazia
		if apiKey == "" {
			switch name {
			case "openai":
				apiKey = os.Getenv("OPENAI_API_KEY")
			case "anthropic":
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			case "gemini":
				apiKey = os.Getenv("GEMINI_API_KEY")
			case "openrouter":
				apiKey = os.Getenv("OPENROUTER_API_KEY")
			}
		}
	}

	// 4. Tenta usar o GoclawProvider (Novo motor de IA)
	gp, err := NewGoclawProvider(name, baseURL, apiKey, "")
	if err == nil {
		return gp
	}

	fmt.Printf("⚠️ Erro ao iniciar GoclawProvider (%v), usando fallback nativo...\n", err)

	switch name {
	case "ollama":
		return NewOllamaProvider(baseURL)
	case "anthropic":
		return &AnthropicProvider{BaseURL: baseURL, APIKey: apiKey}
	case "gemini":
		return NewGeminiProvider(baseURL, apiKey)
	case "openrouter":
		return NewOpenRouterProvider(baseURL, apiKey)
	case "openai", "lmstudio":
		return NewOpenAIProvider(baseURL, apiKey)
	default:
		return NewOpenAIProvider(baseURL, apiKey)
	}
}

func (c *LLMClient) Ask(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()
	session := analytic.NewSession("internal-ask")
	return c.AskWithContext(ctx, prompt, session)
}

func (c *LLMClient) AskWithContext(ctx context.Context, prompt string, session *analytic.Session) (string, error) {
	messages := []Message{}
	if c.SystemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: c.SystemPrompt})
	}
	messages = append(messages, Message{Role: "user", Content: prompt})

	var tools []Tool
	if c.Registry != nil {
		defs := c.Registry.ToProviderDefs()
		for _, d := range defs {
			t := Tool{Type: "function"}
			t.Function.Name = d.Function.Name
			t.Function.Description = d.Function.Description
			// Converter json.RawMessage para map[string]any para a interface Tool
			var params map[string]any
			json.Unmarshal(d.Function.Parameters, &params)
			t.Function.Parameters = params
			tools = append(tools, t)
		}
	}

	for i := 0; i < 10; i++ { // Limit of 10 tool iterations
		if c.OnProgress != nil {
			c.OnProgress("Pensando...")
		}

		startLLM := analytic.StartLLMCall(ctx, c.Provider.Name(), c.Model)
		startTime := time.Now()
		resp, err := c.Provider.Chat(ctx, c.Model, messages, tools)
		duration := time.Since(startTime)

		if err != nil {
			analytic.EndLLMCall(session, fmt.Sprintf("LLM Call %d", i+1), startLLM, 0, 0, err)
			return "", err
		}

		logger.Info("✨ IA respondeu", "duration", duration.String(), "provider", c.Provider.Name())
		analytic.EndLLMCall(session, fmt.Sprintf("LLM Call %d", i+1), startLLM, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, nil)

		messages = append(messages, Message{
			Role:      resp.Role,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			if resp.Content == "" {
				if i < 2 { // Tenta até 2 vezes um "nudge" se vier vazio
					logger.Info("⚠️ IA retornou resposta vazia. Tentando 'nudge'...", "iteration", i)
					messages = append(messages, Message{Role: "user", Content: "Please provide your analysis or the requested JSON output."})
					continue
				}
				return "", ErrEmptyResponse
			}
			return resp.Content, nil
		}

		// Execute requested tools
		for _, toolCall := range resp.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return "", fmt.Errorf("error decoding tool arguments: %v", err)
			}

			logger.Info("🔧 Executando ferramenta", "name", toolCall.Function.Name, "args", args)
			if c.OnProgress != nil {
				c.OnProgress(fmt.Sprintf("Executando ferramenta: %s", toolCall.Function.Name))
			}

			startTool := time.Now()
			output, err := c.Registry.Execute(ctx, toolCall.Function.Name, args)
			analytic.RecordTool(session, toolCall.Function.Name, startTool, args, err)

			if err != nil {
				output = fmt.Sprintf("Error executing tool: %v", err)
			}

			messages = append(messages, Message{
				Role:       "tool",
				Content:    output,
				ToolCallID: toolCall.ID,
			})
		}
	}

	return "", fmt.Errorf("tool iteration limit exceeded")
}

// TranslateToEnglish translates the given text to English using the LLM.
// If the text is already in English, it should return it as is.
func (c *LLMClient) TranslateToEnglish(text string) (string, error) {
	if text == "" {
		return "", nil
	}

	prompt := fmt.Sprintf(`SYSTEM: You are a technical translator. 
TASK: Translate the following text to English.
RULES:
1. If the text is already in English, return it EXACTLY as it is.
2. If it's in another language (like Portuguese), translate it to technical English.
3. DO NOT add explanations, comments, or "Here is the translation".
4. RETURN ONLY THE TRANSLATED TEXT.

TEXT TO TRANSLATE:
%s`, text)

	session := analytic.NewSession("translate")
	translated, err := c.AskWithContext(context.Background(), prompt, session)
	if err != nil {
		return "", fmt.Errorf("fail to translate: %v", err)
	}

	return translated, nil
}
