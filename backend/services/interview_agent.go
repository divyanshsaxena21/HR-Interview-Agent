package services

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// InterviewAgent handles the main interview conversation
type InterviewAgent struct {
	db *mongo.Client
	llm LLMService
}

func NewInterviewAgent(db *mongo.Client, llm LLMService) *InterviewAgent {
	return &InterviewAgent{
		db:  db,
		llm: llm,
	}
}

func (ia *InterviewAgent) Name() string {
	return "InterviewAgent"
}

func (ia *InterviewAgent) Execute(ctx context.Context, state *AgentState) (*AgentState, error) {
	log.Printf("InterviewAgent: Processing session %s", state.SessionID)

	// Get last candidate message
	if len(state.Messages) == 0 {
		// Start interview with greeting
		greeting := "Hello! Thank you for joining this interview. I'm your AI interviewer. Let's get started with some questions about your background and experience."
		state.Messages = append(state.Messages, map[string]interface{}{
			"role":    "assistant",
			"content": greeting,
		})
		return state, nil
	}

	lastMsg := state.Messages[len(state.Messages)-1]
	if lastMsg["role"] != "user" {
		return state, nil
	}

	userMessage := lastMsg["content"].(string)

	// Get HR memory questions for this role
	questions, err := ia.getHRQuestions(ctx)
	if err != nil {
		log.Printf("Warning: Failed to fetch HR questions: %v", err)
	}

	// Build prompt with questions context
	systemPrompt := ia.buildSystemPrompt(questions, state)

	// Get LLM response
	response, err := ia.llm.Generate(ctx, userMessage, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm generation failed: %w", err)
	}

	// Add response to messages
	state.Messages = append(state.Messages, map[string]interface{}{
		"role":    "assistant",
		"content": response,
	})

	// Store messages in database
	ia.storeMessages(ctx, state)

	return state, nil
}

func (ia *InterviewAgent) getHRQuestions(ctx context.Context) ([]bson.M, error) {
	coll := ia.db.Database("ai_recruiter").Collection("hr_memory")
	cursor, err := coll.Find(ctx, bson.M{"active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []bson.M
	if err = cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

func (ia *InterviewAgent) buildSystemPrompt(questions []bson.M, state *AgentState) string {
	prompt := `You are an AI interviewer for a recruiting platform. Your job is to:
1. Ask relevant interview questions
2. Follow up on candidate responses
3. Collect missing information (GitHub, LinkedIn, Portfolio)
4. Assess the candidate's fit for the role
5. Be professional and encouraging

Guidelines:
- Ask one question at a time
- Listen carefully to responses
- Ask clarifying follow-up questions when needed
- Try to naturally collect GitHub, LinkedIn, and Portfolio links
- Be conversational and friendly
- After you have enough information, end the interview gracefully

Question Bank:
`

	for _, q := range questions {
		if question, ok := q["question"].(string); ok {
			prompt += fmt.Sprintf("- %s\n", question)
		}
	}

	return prompt
}

func (ia *InterviewAgent) storeMessages(ctx context.Context, state *AgentState) {
	coll := ia.db.Database("ai_recruiter").Collection("interview_messages")

	// Store all new messages
	for _, msg := range state.Messages {
		msgDoc := bson.M{
			"session_id": state.SessionID,
			"role":       msg["role"],
			"message":    msg["content"],
			"timestamp":  bson.M{"$currentDate": true},
		}
		coll.InsertOne(ctx, msgDoc)
	}
}
