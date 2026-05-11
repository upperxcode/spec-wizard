package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"spec-wizard/internal/adapters"
	"spec-wizard/internal/logger"
	"spec-wizard/internal/registry"
)

// ExpertProvider implementa a interface llm.Provider delegando a execução para um plugin Expert (CLI Wrapper)
type ExpertProvider struct {
	ExpertID string
	Client   *adapters.ExpertClient
	Plugins  []*registry.ExpertPlugin
}

func NewExpertProvider(expertID string, plugins []*registry.ExpertPlugin) *ExpertProvider {
	return &ExpertProvider{
		ExpertID: expertID,
		Client:   adapters.NewExpertClient(),
		Plugins:  plugins,
	}
}

func (p *ExpertProvider) Name() string {
	return "expert-" + p.ExpertID
}

func (p *ExpertProvider) Chat(ctx context.Context, model string, messages []Message, tools []Tool) (*Response, error) {
	// 1. Localiza o plugin no registro
	var targetPlugin *registry.ExpertPlugin
	for _, pl := range p.Plugins {
		if pl.ID == p.ExpertID {
			targetPlugin = pl
			break
		}
	}

	if targetPlugin == nil {
		return nil, fmt.Errorf("expert plugin '%s' não encontrado no registro", p.ExpertID)
	}

	// 2. Garante que o expert está rodando (PluginManager)
	if err := registry.GlobalManager.EnsureExpertRunning(targetPlugin); err != nil {
		return nil, fmt.Errorf("falha ao iniciar expert %s: %v", p.ExpertID, err)
	}

	// 3. Prepara o payload para o contrato do plugin
	data := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"tools":    tools,
	}

	logger.Info("📡 [ExpertProvider] Delegando Chat para o Expert", "expert", p.ExpertID, "model", model)

	// 4. Faz a chamada RPC/HTTP para o expert
	respMap, err := p.Client.CallExpert(targetPlugin, "Chat", data)
	if err != nil {
		logger.Error("❌ [ExpertProvider] Erro na chamada ao Expert", err)
		return nil, err
	}

	// 5. Converte o resultado de volta para a estrutura Response
	respJSON, err := json.Marshal(respMap)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar resposta do expert: %v", err)
	}

	var response Response
	if err := json.Unmarshal(respJSON, &response); err != nil {
		// Tenta mapear campos manualmente se o Unmarshal direto falhar por diferenças de case
		logger.Warn("⚠️ [ExpertProvider] Falha no unmarshal direto, tentando mapeamento manual", "error", err)
		
		if content, ok := respMap["content"].(string); ok {
			response.Content = content
		}
		if role, ok := respMap["role"].(string); ok {
			response.Role = role
		}
		// ... outros campos podem ser mapeados conforme necessário
	}

	return &response, nil
}
