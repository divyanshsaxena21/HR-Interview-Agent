package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// LLMService defines the interface for LLM operations
type LLMService interface {
	Generate(ctx context.Context, message, systemPrompt string) (string, error)
}

// GroqLLMService implements LLMService using Groq API
type GroqLLMService struct {
	apiKey  string
	baseURL string
	model   string
}

type GroqRequestPayload struct {
	Model    string        `json:"model"`
	Messages []GroqMsgItem `json:"messages"`
	MaxTokens int          `json:"max_tokens,omitempty"`
}

type GroqMsgItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponsePayload struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewGroqLLMService() *GroqLLMService {
	return &GroqLLMService{
		apiKey:  os.Getenv("GROQ_API_KEY"),
		baseURL: "https://api.groq.com/openai/v1",
		model:   "mixtral-8x7b-32768",
	}
}

func (g *GroqLLMService) Generate(ctx context.Context, message, systemPrompt string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}

	messages := []GroqMsgItem{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: message,
		},
	}

	payload := GroqRequestPayload{
		Model:    g.model,
		Messages: messages,
		MaxTokens: 500,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/chat/completions", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Groq API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Groq API error: %s", string(body))
		return "", fmt.Errorf("groq api returned status %d: %s", resp.StatusCode, string(body))
	}

	var response GroqResponsePayload
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}
