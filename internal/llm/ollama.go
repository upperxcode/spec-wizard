package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OllamaProvider struct {
	BaseURL string
	HTTP    *http.Client
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
	return &OllamaProvider{
		BaseURL: baseURL,
		HTTP:    &http.Client{},
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	// Ollama native chat payload
	payload := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	if len(tools) > 0 {
		payload["tools"] = tools
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama error status: %d", resp.StatusCode)
	}

	var result struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Response{
		Role:      result.Message.Role,
		Content:   result.Message.Content,
		ToolCalls: result.Message.ToolCalls,
	}, nil
}
