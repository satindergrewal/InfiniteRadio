package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Client talks to a local Ollama API for LLM-powered caption and name generation.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates an Ollama client. Pass empty model to auto-detect later.
func NewClient(baseURL, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // first call loads model into VRAM (~60s for 32B)
		},
	}
}

// generateRequest is the Ollama /api/generate request body.
type generateRequest struct {
	Model  string         `json:"model"`
	Prompt string         `json:"prompt"`
	System string         `json:"system,omitempty"`
	Stream bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

// generateResponse is the Ollama /api/generate response.
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Available checks if Ollama is reachable and the model is loaded.
func (c *Client) Available(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// Generate sends a prompt with a system message and returns the LLM response.
func (c *Client) Generate(ctx context.Context, system, prompt string) (string, error) {
	body := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		System: system,
		Stream: false,
		Options: map[string]any{
			"temperature":    0.9,
			"top_p":          0.95,
			"num_predict":    128, // captions are short, cap output
			"repeat_penalty": 1.1,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	return strings.TrimSpace(result.Response), nil
}

// Model returns the configured model name.
func (c *Client) Model() string {
	return c.model
}

// WaitForReady polls Ollama until it responds or context expires.
// Unlike ACE-Step, this is non-fatal -- Ollama is optional.
func (c *Client) WaitForReady(ctx context.Context) bool {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if c.Available(ctx) {
				log.Printf("Ollama ready (model: %s)", c.model)
				return true
			}
		}
	}
}
