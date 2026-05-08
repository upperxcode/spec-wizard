package llm

import (
	"context"
	"fmt"
	"google.golang.org/genai"
)

type GeminiProvider struct {
	BaseURL string // Mantido por compatibilidade de interface, não usado pelo SDK
	APIKey  string
}

func NewGeminiProvider(baseURL, apiKey string) *GeminiProvider {
	return &GeminiProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	fmt.Printf("📡 [Gemini SDK] Gerando conteúdo com modelo: %s\n", model)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  p.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao criar cliente Gemini: %v", err)
	}

	// 1. Mapear mensagens para o formato do SDK (genai.Content)
	var contents []*genai.Content
	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: m.Content},
			},
		})
	}

	// 2. Configuração (opcional)
	var genaiConfig *genai.GenerateContentConfig

	// 3. Chamar a API usando GenerateContent
	result, err := client.Models.GenerateContent(ctx, model, contents, genaiConfig)
	if err != nil {
		return nil, fmt.Errorf("erro na API do Gemini: %v", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Gemini não retornou candidatos válidos")
	}

	// 4. Converter resposta do SDK para nossa struct Response simplificada
	return &Response{
		Content: result.Text(),
		Role:    "assistant",
	}, nil
}
