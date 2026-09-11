package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Options struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

type ChatClient interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

func NewClient(options Options) *Client {
	baseURL := strings.TrimSuffix(options.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		apiKey:     options.APIKey,
		baseURL:    baseURL,
		model:      options.Model,
		httpClient: httpClient,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", fmt.Errorf("openrouter api key is not configured")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("chat messages are required")
	}

	body, err := json.Marshal(struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature"`
	}{Model: c.model, Messages: messages, Temperature: 0.3})
	if err != nil {
		return "", fmt.Errorf("failed to encode openrouter request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build openrouter request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("HTTP-Referer", "http://localhost:5173")
	request.Header.Set("X-Title", "ArenaGO Padel Assistant")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to call openrouter: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read openrouter response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("openrouter returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}

	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("failed to decode openrouter response: %w", err)
	}
	if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("openrouter returned an empty response")
	}
	return result.Choices[0].Message.Content, nil
}
