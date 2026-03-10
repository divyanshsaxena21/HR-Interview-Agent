package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ChatService struct {
	interviewCollection *mongo.Collection
	hrMemoryCollection  *mongo.Collection
	langchainAgent      *LangChainAgent
}

func NewChatService(interviewCollection, hrMemoryCollection *mongo.Collection) *ChatService {
	return &ChatService{
		interviewCollection: interviewCollection,
		hrMemoryCollection:  hrMemoryCollection,
		langchainAgent:      NewLangChainAgent(),
	}
}

func (cs *ChatService) ProcessMessage(ctx context.Context, interviewID primitive.ObjectID, message string) (string, error) {
	interview := struct {
		Messages []struct {
			Role    string
			Content string
		} `bson:"messages"`
		Email     string
		GitHub    string
		LinkedIn  string
		Portfolio string
		Role      string
	}{}

	err := cs.interviewCollection.FindOne(ctx, bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		return "", err
	}

	systemPrompt := `You are an HR recruiter conducting a screening interview. 
Your goals:
1. Ask questions to assess candidate fit and skills
2. Collect missing candidate information (GitHub, LinkedIn, Portfolio)
3. Be conversational and professional
4. If a candidate answer indicates they fail a dealbreaker requirement, internally mark them as rejected but continue the interview naturally
5. Provide constructive feedback`

	if interview.Email == "" {
		systemPrompt += "\n\nCandidate has not yet provided: email"
	}
	if interview.GitHub == "" {
		systemPrompt += "\n\nCandidate has not provided GitHub profile"
	}
	if interview.LinkedIn == "" {
		systemPrompt += "\n\nCandidate has not provided LinkedIn profile"
	}
	if interview.Portfolio == "" {
		systemPrompt += "\n\nCandidate has not provided portfolio"
	}

	response, err := cs.langchainAgent.GenerateResponse(systemPrompt, message, interview.Role)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (cs *ChatService) SaveMessage(ctx context.Context, interviewID primitive.ObjectID, role, content string) error {
	msg := bson.M{
		"role":      role,
		"content":   content,
		"timestamp": time.Now().Unix(),
	}

	_, err := cs.interviewCollection.UpdateOne(ctx,
		bson.M{"_id": interviewID},
		bson.M{
			"$push": bson.M{"messages": msg},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	return err
}

func (cs *ChatService) CheckDealbreaker(ctx context.Context, interviewID primitive.ObjectID, message string) (bool, string, error) {
	dealbreakers, err := cs.getDealbreakers(ctx)
	if err != nil {
		return false, "", err
	}

	for _, db := range dealbreakers {
		if matchesDealbreaker(message, db.Question) {
			return true, db.Question, nil
		}
	}
	return false, "", nil
}

func (cs *ChatService) MarkAsRejected(ctx context.Context, interviewID primitive.ObjectID, reason string) error {
	_, err := cs.interviewCollection.UpdateOne(ctx,
		bson.M{"_id": interviewID},
		bson.M{
			"$set": bson.M{
				"rejected":         true,
				"rejection_reason": reason,
				"status":           "completed",
				"updated_at":       time.Now(),
			},
		},
	)
	return err
}

type DealBreakerQuestion struct {
	ID       primitive.ObjectID `bson:"_id"`
	Question string             `bson:"question"`
}

func (cs *ChatService) getDealbreakers(ctx context.Context) ([]DealBreakerQuestion, error) {
	filter := bson.M{"is_dealbreaker": true, "active": true}
	cursor, err := cs.hrMemoryCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dealbreakers []DealBreakerQuestion
	if err := cursor.All(ctx, &dealbreakers); err != nil {
		return nil, err
	}
	return dealbreakers, nil
}

func matchesDealbreaker(message, question string) bool {
	if len(message) > 0 && (message[0:1] == "n" || message[0:1] == "N") {
		return true
	}
	return false
}
