package services

import (
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type LangChainAgent struct {
	memory              []models.Message
	hrMemoryCollection  *mongo.Collection
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
		memory:             []models.Message{},
		hrMemoryCollection: nil,
	}
}

func NewLangChainAgentWithMemory(hrMemoryCollection *mongo.Collection) *LangChainAgent {
	return &LangChainAgent{
		memory:             []models.Message{},
		hrMemoryCollection: hrMemoryCollection,
	}
}

func (la *LangChainAgent) GenerateInitialQuestion(role string) (string, error) {
	questions := utils.GetHRInterviewQuestions(role)
	if len(questions) > 0 {
		return questions[0], nil
	}
	return "Tell me about your background and professional experience.", nil
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
		Model:     "llama-3.3-70b-versatile",
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

	// Try to fetch questions from HRMemory first
	var questions []string
	if la.hrMemoryCollection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Fetch active questions from HRMemory for this role
		filter := bson.M{"active": true, "category": role}
		cursor, err := la.hrMemoryCollection.Find(ctx, filter)
		if err == nil {
			defer cursor.Close(ctx)
			var hrQuestions []models.HRMemory
			if err := cursor.All(ctx, &hrQuestions); err == nil && len(hrQuestions) > 0 {
				// Convert HRMemory questions to strings
				for _, q := range hrQuestions {
					questions = append(questions, q.Question)
				}
			}
		}
	}

	// If no HR Memory questions found, use static questions
	if len(questions) == 0 {
		questions = utils.GetHRInterviewQuestions(role)
	}

	if questionsAsked >= len(questions) {
		return "END_INTERVIEW", nil
	}

	if questionsAsked < len(questions) {
		return questions[questionsAsked], nil
	}

	return "Do you have any final questions for us?", nil
}

// GenerateQuestionWithTracking returns the next unanswered mandatory HR question
// Combines both static questions and HR Memory questions (dealbreakers, role-specific)
func (la *LangChainAgent) GenerateQuestionWithTracking(interview models.Interview) (string, error) {
	// Start with mandatory static questions (8 HR screening questions)
	staticQuestions := utils.GetHRInterviewQuestions(interview.Role)
	allQuestions := []string{}
	for _, q := range staticQuestions {
		allQuestions = append(allQuestions, q)
	}

	// Add HR memory questions (dealbreakers, role-specific, etc.)
	if la.hrMemoryCollection != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		filter := bson.M{"active": true, "category": interview.Role}
		cursor, err := la.hrMemoryCollection.Find(ctx, filter)
		if err == nil {
			defer cursor.Close(ctx)
			var hrQuestions []models.HRMemory
			if cursorErr := cursor.All(ctx, &hrQuestions); cursorErr == nil {
				// Add HR Memory questions (these include dealbreakers and role-specific questions)
				for _, hq := range hrQuestions {
					allQuestions = append(allQuestions, hq.Question)
				}
			}
		}
	}

	// Get the next question based on counter
	if interview.HRQuestionsAsked >= len(allQuestions) {
		return "END_INTERVIEW", nil
	}

	nextQuestion := allQuestions[interview.HRQuestionsAsked]
	return nextQuestion, nil
}

func (la *LangChainAgent) UpdateMemory(role string, content string) {
	la.memory = append(la.memory, models.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Unix(),
	})
}
