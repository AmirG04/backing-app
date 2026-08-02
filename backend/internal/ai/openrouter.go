package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"

// defaultModel se usa si no se configura OPENROUTER_MODEL. Cualquier
// modelo del catalogo de OpenRouter que soporte "tool calling" sirve -
// ver https://openrouter.ai/models para elegir otro (incluye opciones
// gratuitas con limitaciones de uso).
const defaultModel = "anthropic/claude-sonnet-4.5"

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(apiKey string, model string) *Client {
	if model == "" {
		model = defaultModel
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Tool describe una herramienta en el formato "function calling" que usa
// OpenRouter (compatible con la API de OpenAI). Parameters se deja como
// JSON crudo para poder reenviar directamente el schema que entrega un
// servidor MCP (via tools/list) sin tener que re-modelarlo en Go.
type Tool struct {
	Type     string       `json:"type"` // siempre "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Message representa un turno de la conversacion. Role puede ser
// "system", "user", "assistant" o "tool".
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall es como el modelo indica que quiere invocar una herramienta.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON codificado como string
}

type request struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
	MaxTokens int       `json:"max_tokens"`
}

type Response struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// SendMessage manda una conversacion completa (con historial y tools) a
// OpenRouter y retorna la respuesta cruda del modelo.
func (c *Client) SendMessage(messages []Message, tools []Tool) (*Response, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY no configurada")
	}

	reqBody := request{Model: c.model, Messages: messages, Tools: tools, MaxTokens: 1024}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error serializando peticion: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, openRouterAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error creando peticion: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	// Headers opcionales que recomienda OpenRouter para atribucion en su dashboard.
	req.Header.Set("HTTP-Referer", "https://github.com/")
	req.Header.Set("X-Title", "Banking App Chat")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error llamando a la API de OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter api error (status %d): %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	return &result, nil
}
