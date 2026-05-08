package llm

import (
	"context"
	"fmt"
)

type AnthropicProvider struct {
	BaseURL string
	APIKey  string
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	return nil, fmt.Errorf("provedor Anthropic ainda não implementado")
}
