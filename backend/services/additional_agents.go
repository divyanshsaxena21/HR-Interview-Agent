package services

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InformationAgent ensures candidate provides required fields
type InformationAgent struct {
	db *mongo.Client
}

func NewInformationAgent(db *mongo.Client) *InformationAgent {
	return &InformationAgent{db: db}
}

func (ia *InformationAgent) Name() string {
	return "InformationAgent"
}

func (ia *InformationAgent) Execute(ctx context.Context, state *AgentState) (*AgentState, error) {
	log.Printf("InformationAgent: Verifying candidate information for %s", state.CandidateID)

	candidateID, _ := primitive.ObjectIDFromHex(state.CandidateID)
	coll := ia.db.Database("ai_recruiter").Collection("candidates")

	var candidate bson.M
	err := coll.FindOne(ctx, bson.M{"_id": candidateID}).Decode(&candidate)
	if err != nil {
		return nil, fmt.Errorf("candidate not found: %w", err)
	}

	// Check for missing fields
	missing := []string{}
	if candidate["github"] == "" || candidate["github"] == nil {
		missing = append(missing, "github")
	}
	if candidate["linkedin"] == "" || candidate["linkedin"] == nil {
		missing = append(missing, "linkedin")
	}

	state.Context["missing_fields"] = missing

	if len(missing) > 0 {
		log.Printf("InformationAgent: Missing fields for candidate: %v", missing)
		// This will be handled by the interview agent asking for missing info
	}

	return state, nil
}

// DocumentAgent handles document uploads
type DocumentAgent struct {
	db *mongo.Client
}

func NewDocumentAgent(db *mongo.Client) *DocumentAgent {
	return &DocumentAgent{db: db}
}

func (da *DocumentAgent) Name() string {
	return "DocumentAgent"
}

func (da *DocumentAgent) Execute(ctx context.Context, state *AgentState) (*AgentState, error) {
	log.Printf("DocumentAgent: Processing documents for session %s", state.SessionID)

	// Initialize documents list if not present
	if state.Context["documents"] == nil {
		state.Context["documents"] = []interface{}{}
	}

	// Documents will be handled via WebSocket file uploads
	return state, nil
}

// SummaryAgent generates final evaluation
type SummaryAgent struct {
	db *mongo.Client
	llm LLMService
}

func NewSummaryAgent(db *mongo.Client, llm LLMService) *SummaryAgent {
	return &SummaryAgent{
		db:  db,
		llm: llm,
	}
}

func (sa *SummaryAgent) Name() string {
	return "SummaryAgent"
}

func (sa *SummaryAgent) Execute(ctx context.Context, state *AgentState) (*AgentState, error) {
	log.Printf("SummaryAgent: Generating summary for session %s", state.SessionID)

	// Retrieve all interview messages
	msgColl := sa.db.Database("ai_recruiter").Collection("interview_messages")
	cursor, err := msgColl.Find(ctx, bson.M{"session_id": state.SessionID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []bson.M
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode messages: %w", err)
	}

	// Build conversation summary
	conversationText := ""
	for _, msg := range messages {
		role := msg["role"].(string)
		message := msg["message"].(string)
		conversationText += fmt.Sprintf("%s: %s\n", role, message)
	}

	// Generate evaluation using LLM
	evaluationPrompt := fmt.Sprintf(`Based on this interview conversation, provide a structured evaluation:

%s

Please provide:
1. Communication Score (0-100)
2. Technical Score (0-100)
3. Confidence Score (0-100)
4. Key Strengths (list)
5. Areas for Improvement (list)
6. Overall Summary (2-3 sentences)

Format as JSON.`, conversationText)

	response, err := sa.llm.Generate(ctx, evaluationPrompt, "You are an expert recruiter evaluating candidates.")
	if err != nil {
		return nil, fmt.Errorf("failed to generate evaluation: %w", err)
	}

	state.Context["evaluation"] = response
	state.IsComplete = true

	// Store evaluation result
	resultColl := sa.db.Database("ai_recruiter").Collection("interview_results")
	result := bson.M{
		"session_id": state.SessionID,
		"candidate_id": state.CandidateID,
		"evaluation": response,
		"created_at": bson.M{"$currentDate": true},
	}
	resultColl.InsertOne(ctx, result)

	return state, nil
}
