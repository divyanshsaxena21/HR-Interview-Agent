package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SchedulingAgent handles interview scheduling
type SchedulingAgent struct {
	db *mongo.Client
	llm LLMService
}

func NewSchedulingAgent(db *mongo.Client, llm LLMService) *SchedulingAgent {
	return &SchedulingAgent{
		db:  db,
		llm: llm,
	}
}

func (sa *SchedulingAgent) Name() string {
	return "SchedulingAgent"
}

func (sa *SchedulingAgent) Execute(ctx context.Context, state *AgentState) (*AgentState, error) {
	log.Printf("SchedulingAgent: Processing candidate %s", state.CandidateID)

	// Generate session ID
	sessionID := uuid.New().String()

	// Convert candidate ID to ObjectID
	candidateObjID, err := primitive.ObjectIDFromHex(state.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("invalid candidate ID format: %w", err)
	}

	// Get candidate info
	coll := sa.db.Database("ai_recruiter").Collection("candidates")
	var candidate bson.M
	err = coll.FindOne(ctx, bson.M{"_id": candidateObjID}).Decode(&candidate)
	if err != nil {
		return nil, fmt.Errorf("candidate not found: %w", err)
	}

	// Create interview record
	interviewColl := sa.db.Database("ai_recruiter").Collection("interviews")
	interview := bson.M{
		"session_id":    sessionID,
		"candidate_id":  state.CandidateID,
		"candidate_name": candidate["name"],
		"email":         candidate["email"],
		"role":          candidate["role"],
		"github":        candidate["github"],
		"linkedin":      candidate["linkedin"],
		"status":        "scheduled",
		"messages":      []interface{}{},
		"rejected":      false,
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	}

	result, err := interviewColl.InsertOne(ctx, interview)
	if err != nil {
		return nil, fmt.Errorf("failed to create interview: %w", err)
	}

	log.Printf("SchedulingAgent: Created interview session %s", sessionID)

	// Send email notification
	emailService := NewEmailService()
	candidateEmail := candidate["email"].(string)
	candidateName := candidate["name"].(string)
	go emailService.SendInterviewEmail(candidateEmail, candidateName, sessionID)

	// Update state
	state.SessionID = sessionID
	state.Context["interview_id"] = result.InsertedID
	state.Context["session_created_at"] = time.Now()

	// Update candidate status to in_progress (interview scheduled but not yet completed)
	candidateColl := sa.db.Database("ai_recruiter").Collection("candidates")
	_, err = candidateColl.UpdateOne(ctx, bson.M{"_id": candidateObjID}, bson.M{
		"$set": bson.M{"status": "in_progress"},
	})
	if err != nil {
		log.Printf("Warning: Failed to update candidate status: %v", err)
	}

	return state, nil
}
