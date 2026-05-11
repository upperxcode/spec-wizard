package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"spec-wizard/internal/registry"
	"time"
)

// ExpertClient gerencia a comunicação HTTP com os plugins MCP
type ExpertClient struct {
	HTTPClient *http.Client
}

func NewExpertClient() *ExpertClient {
	return &ExpertClient{
		HTTPClient: &http.Client{Timeout: 600 * time.Second},
	}
}

// RequestPayload define a estrutura genérica para enviar ações ao Expert
type RequestPayload struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// CallExpert envia uma solicitação genérica para um plugin MCP
func (c *ExpertClient) CallExpert(plugin *registry.ExpertPlugin, action string, data interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/%s", plugin.Endpoint, action)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar payload: %v", err)
	}

	fmt.Printf("📡 Enviando '%s' para o Expert %s (%s)...\n", action, plugin.ID, url)

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("falha na conexão com o expert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expert retornou erro: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta do expert: %v", err)
	}

	return result, nil
}

// GetOptions busca as opções suportadas diretamente do endpoint /options do expert
func (c *ExpertClient) GetOptions(plugin *registry.ExpertPlugin) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/options", plugin.Endpoint)

	fmt.Printf("📡 Buscando opções globais do Expert %s (%s)...\n", plugin.ID, url)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("falha na conexão com o expert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expert retornou erro: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta do expert: %v", err)
	}

	return result, nil
}

// FormatCode solicita a formatação de um bloco de código ao Expert
func (c *ExpertClient) FormatCode(plugin *registry.ExpertPlugin, code string) (string, error) {
	url := fmt.Sprintf("%s/Format", plugin.Endpoint)

	reqBody := map[string]string{"code": code}
	jsonData, _ := json.Marshal(reqBody)

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expert error: status %d", resp.StatusCode)
	}

	var result struct {
		FormattedCode string `json:"formatted_code"`
		Error         string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf(result.Error)
	}

	return result.FormattedCode, nil
}
