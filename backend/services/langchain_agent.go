package services

import (
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/utils"
)

type LangChainAgent struct {
	memory []models.Message
}

func NewLangChainAgent() *LangChainAgent {
	return &LangChainAgent{
		memory: []models.Message{},
	}
}

func (la *LangChainAgent) GenerateInitialQuestion(role string) (string, error) {
	// Return the first HR question
	questions := utils.GetHRInterviewQuestions(role)
	if len(questions) > 0 {
		return questions[0], nil
	}
	return "Tell me about yourself and your background.", nil
}

func (la *LangChainAgent) GenerateQuestion(transcript []models.Message, role string) (string, error) {
	// Count how many AI questions have been asked (questions are at even indices in transcript, starting from 0)
	questionsAsked := 0
	for _, msg := range transcript {
		if msg.Role == "ai" {
			questionsAsked++
		}
	}

	// Get the structured HR questions
	questions := utils.GetHRInterviewQuestions(role)

	// If we've asked all 10 questions, return end-of-interview marker
	if questionsAsked >= len(questions) {
		return "END_INTERVIEW", nil
	}

	// Return the next question in the sequence
	if questionsAsked < len(questions) {
		return questions[questionsAsked], nil
	}

	// Fallback
	return "Do you have any final questions for us?", nil
}

func (la *LangChainAgent) UpdateMemory(role string, content string) {
	la.memory = append(la.memory, models.Message{
		Role:    role,
		Content: content,
	})
}
