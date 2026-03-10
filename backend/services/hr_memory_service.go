package services

import (
	"ai-recruiter/backend/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type HRMemoryService struct {
	collection *mongo.Collection
}

func NewHRMemoryService(collection *mongo.Collection) *HRMemoryService {
	return &HRMemoryService{
		collection: collection,
	}
}

func (s *HRMemoryService) CreateQuestion(ctx context.Context, req models.HRMemoryRequest) (*models.HRMemory, error) {
	memory := models.HRMemory{
		ID:            primitive.NewObjectID(),
		Category:      req.Category,
		Question:      req.Question,
		Tags:          req.Tags,
		IsDealbreaker: req.IsDealbreaker,
		Active:        req.Active,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	result, err := s.collection.InsertOne(ctx, memory)
	if err != nil {
		return nil, err
	}

	memory.ID = result.InsertedID.(primitive.ObjectID)
	return &memory, nil
}

func (s *HRMemoryService) GetAllQuestions(ctx context.Context) ([]models.HRMemory, error) {
	filter := bson.M{"active": true}
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.HRMemory
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func (s *HRMemoryService) GetAllQuestionsAdmin(ctx context.Context) ([]models.HRMemory, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.HRMemory
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func (s *HRMemoryService) GetQuestionsByCategory(ctx context.Context, category string) ([]models.HRMemory, error) {
	filter := bson.M{"category": category, "active": true}
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.HRMemory
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func (s *HRMemoryService) UpdateQuestion(ctx context.Context, id primitive.ObjectID, req models.HRMemoryRequest) (*models.HRMemory, error) {
	update := bson.M{
		"$set": bson.M{
			"category":       req.Category,
			"question":       req.Question,
			"tags":           req.Tags,
			"is_dealbreaker": req.IsDealbreaker,
			"active":         req.Active,
			"updated_at":     time.Now(),
		},
	}

	var updated models.HRMemory
	err := s.collection.FindOneAndUpdate(ctx, bson.M{"_id": id}, update).Decode(&updated)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *HRMemoryService) DeleteQuestion(ctx context.Context, id primitive.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
