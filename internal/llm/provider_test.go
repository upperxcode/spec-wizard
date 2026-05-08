package llm

import (
	"context"
	"os"
	"testing"
)

func TestURLNormalization(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		baseURL  string
		expected string
	}{
		{
			name:     "OpenAI Default",
			provider: "openai",
			baseURL:  "",
			expected: "https://api.openai.com/v1",
		},
		{
			name:     "Ollama Default",
			provider: "ollama",
			baseURL:  "",
			expected: "http://localhost:11434",
		},
		{
			name:     "OpenRouter Default",
			provider: "openrouter",
			baseURL:  "",
			expected: "https://openrouter.ai/api/v1",
		},
		{
			name:     "OpenRouter Fix No V1",
			provider: "openrouter",
			baseURL:  "https://openrouter.ai/api",
			expected: "https://openrouter.ai/api/v1",
		},
		{
			name:     "OpenRouter Fix Suffix",
			provider: "openrouter",
			baseURL:  "https://custom.proxy/openrouter",
			expected: "https://custom.proxy/openrouter/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvider(tt.provider, tt.baseURL, "key", "")
			// We need to access the internal field, but since we are in the same package 'llm', we can.
			// However, NewProvider returns the interface 'Provider'.
			// We cast it to OpenAIProvider or OllamaProvider to check the BaseURL.

			var actual string
			if op, ok := p.(*OpenAIProvider); ok {
				actual = op.BaseURL
			} else if ol, ok := p.(*OllamaProvider); ok {
				actual = ol.BaseURL
			}

			if actual != tt.expected {
				t.Errorf("expected URL %s, got %s", tt.expected, actual)
			}
		})
	}
}

func TestOpenRouterConnection(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("Pulei teste de conexão real: OPENROUTER_API_KEY não configurada")
	}

	provider := NewProvider("openrouter", "", apiKey, "OPENROUTER_API_KEY")
	client := NewLLMClient(provider, "openrouter/openrouter/free", nil)

	ctx := context.Background()
	resp, err := client.AskWithContext(ctx, "Hello, respond with only the word 'OK' if you can hear me.", nil)
	if err != nil {
		t.Fatalf("Erro na conexão com OpenRouter: %v", err)
	}

	if resp == "" {
		t.Fatal("Resposta vazia do OpenRouter")
	}

	t.Logf("Resposta do OpenRouter: %s", resp)
}
