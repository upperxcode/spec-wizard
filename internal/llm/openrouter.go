package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"spec-wizard/internal/logger"
)

type OpenRouterProvider struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func NewOpenRouterProvider(baseURL, apiKey string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return &OpenRouterProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{},
	}
}

func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

func (p *OpenRouterProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	logger.Info("📡 [OpenRouter] Enviando requisição", "url", p.BaseURL, "model", model)

	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	if len(tools) > 0 {
		payload["tools"] = tools
		payload["tool_choice"] = "auto"
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	// OpenRouter specific headers
	req.Header.Set("HTTP-Referer", "https://github.com/upperxcode/spec-wizard")
	req.Header.Set("X-Title", "Spec Wizard")

	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("openrouter error (%d): %s", resp.StatusCode, errResp.Error.Message)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Role      string     `json:"role"`
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from openrouter")
	}

	choice := result.Choices[0].Message
	return &Response{
		Role:      choice.Role,
		Content:   choice.Content,
		ToolCalls: choice.ToolCalls,
	}, nil
}
