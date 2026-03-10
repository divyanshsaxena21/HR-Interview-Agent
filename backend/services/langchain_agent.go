package services

import (
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type LangChainAgent struct {
	memory []models.Message
}

type GroqRequest struct {
	Model     string        `json:"model"`
	Messages  []GroqMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewLangChainAgent() *LangChainAgent {
	return &LangChainAgent{
		memory: []models.Message{},
	}
}

func (la *LangChainAgent) GenerateInitialQuestion(role string) (string, error) {
	questions := utils.GetHRInterviewQuestions(role)
	if len(questions) > 0 {
		return questions[0], nil
	}
	return "Tell me about yourself and your background.", nil
}

func (la *LangChainAgent) GenerateResponse(systemPrompt, userMessage, role string) (string, error) {
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not set")
	}

	messages := []GroqMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userMessage,
		},
	}

	payload := GroqRequest{
		Model:     "mixtral-8x7b-32768",
		Messages:  messages,
		MaxTokens: 500,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+groqAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("groq api error: %s", string(respBody))
	}

	var groqResp GroqResponse
	if err := json.Unmarshal(respBody, &groqResp); err != nil {
		return "", err
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response from groq api")
	}

	return groqResp.Choices[0].Message.Content, nil
}

func (la *LangChainAgent) GenerateQuestion(transcript []models.Message, role string) (string, error) {
	questionsAsked := 0
	for _, msg := range transcript {
		if msg.Role == "ai" {
			questionsAsked++
		}
	}

	questions := utils.GetHRInterviewQuestions(role)

	if questionsAsked >= len(questions) {
		return "END_INTERVIEW", nil
	}

	if questionsAsked < len(questions) {
		return questions[questionsAsked], nil
	}

	return "Do you have any final questions for us?", nil
}

func (la *LangChainAgent) UpdateMemory(role string, content string) {
	la.memory = append(la.memory, models.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	})
}
