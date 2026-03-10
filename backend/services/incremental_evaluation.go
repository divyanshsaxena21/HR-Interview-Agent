package services

import (
	"ai-recruiter/backend/models"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type IncrementalEvaluator struct {
	db *mongo.Database
}

func NewIncrementalEvaluator(mongoClient *mongo.Client) *IncrementalEvaluator {
	db := mongoClient.Database("ai_recruiter")
	return &IncrementalEvaluator{db: db}
}

func (ie *IncrementalEvaluator) EvaluateAndUpdate(ctx context.Context, interviewID string) error {
	// Fetch interview
	interviewColl := ie.db.Collection("interviews")
	interview := models.Interview{}

	objID, err := ie.parseObjectID(interviewID)
	if err != nil {
		return fmt.Errorf("invalid interview id: %v", err)
	}

	err = interviewColl.FindOne(ctx, bson.M{"_id": objID}).Decode(&interview)
	if err != nil {
		return fmt.Errorf("interview not found: %v", err)
	}

	// Compute incremental scores
	scores := ie.computeScores(interview.Transcript)

	// Update or create analytics record
	analyticsColl := ie.db.Collection("analytics")
	opts := options.Update().SetUpsert(true)

	_, err = analyticsColl.UpdateOne(ctx, bson.M{"interview_id": objID}, bson.M{
		"$set": bson.M{
			"communication_score":   scores.CommunicationScore,
			"technical_score":       scores.TechnicalScore,
			"confidence_score":      scores.ConfidenceScore,
			"problem_solving_score": scores.ProblemSolvingScore,
			"avg_answer_length":     scores.AvgAnswerLength,
			"followups_needed":      scores.FollowupsNeeded,
			"clarity_rating":        scores.ClarityRating,
			"candidate_talk_ratio":  scores.CandidateTalkRatio,
			"updated_at":            time.Now(),
		},
	}, opts)

	if err != nil {
		log.Printf("Failed to update analytics for interview %s: %v", interviewID, err)
		return err
	}

	log.Printf("Incremental evaluation complete for interview %s: comm=%d tech=%d conf=%d", interviewID, scores.CommunicationScore, scores.TechnicalScore, scores.ConfidenceScore)
	return nil
}

type ScoreResult struct {
	CommunicationScore  int
	TechnicalScore      int
	ConfidenceScore     int
	ProblemSolvingScore int
	AvgAnswerLength     int
	FollowupsNeeded     int
	ClarityRating       int
	CandidateTalkRatio  float64
}

func (ie *IncrementalEvaluator) computeScores(transcript []models.Message) ScoreResult {
	result := ScoreResult{
		CommunicationScore:  5,
		TechnicalScore:      5,
		ConfidenceScore:     5,
		ProblemSolvingScore: 5,
		AvgAnswerLength:     0,
		FollowupsNeeded:     0,
		ClarityRating:       5,
		CandidateTalkRatio:  0.5,
	}

	candidateMessages := []string{}
	aiMessages := []string{}
	totalWords := 0

	for _, msg := range transcript {
		if msg.Role == "candidate" {
			candidateMessages = append(candidateMessages, msg.Content)
			totalWords += len(strings.Fields(msg.Content))
		} else if msg.Role == "ai" {
			aiMessages = append(aiMessages, msg.Content)
		}
	}

	if len(candidateMessages) == 0 {
		return result
	}

	// Average answer length
	avgWords := totalWords / len(candidateMessages)
	result.AvgAnswerLength = avgWords

	// Communication score (based on answer length)
	if avgWords >= 60 {
		result.CommunicationScore = 9
	} else if avgWords >= 40 {
		result.CommunicationScore = 8
	} else if avgWords >= 20 {
		result.CommunicationScore = 6
	} else if avgWords >= 10 {
		result.CommunicationScore = 5
	} else if avgWords > 0 {
		result.CommunicationScore = 4
	}

	// Technical depth (keyword density)
	techKeywords := []string{"go", "python", "java", "javascript", "react", "nodejs", "docker", "aws", "sql", "mongo", "typescript", "rust", "database", "api", "microservice"}
	techHits := 0
	for _, msg := range candidateMessages {
		lower := strings.ToLower(msg)
		for _, kw := range techKeywords {
			if strings.Contains(lower, kw) {
				techHits++
			}
		}
	}
	if totalWords > 0 {
		density := float64(techHits) / float64(totalWords)
		if density > 0.05 {
			result.TechnicalScore = 9
		} else if density > 0.03 {
			result.TechnicalScore = 8
		} else if density > 0.015 {
			result.TechnicalScore = 7
		} else if techHits > 0 {
			result.TechnicalScore = 6
		} else {
			result.TechnicalScore = 4
		}
	}

	// Confidence (inverse of hedging words)
	hedgeKeywords := []string{"maybe", "perhaps", "might", "not sure", "i think", "could be", "sort of"}
	hedgeHits := 0
	for _, msg := range candidateMessages {
		lower := strings.ToLower(msg)
		for _, hw := range hedgeKeywords {
			if strings.Contains(lower, hw) {
				hedgeHits++
			}
		}
	}
	if hedgeHits > 3 {
		result.ConfidenceScore = 4
	} else if hedgeHits > 0 {
		result.ConfidenceScore = 5
	} else if avgWords > 30 {
		result.ConfidenceScore = 8
	} else if avgWords > 15 {
		result.ConfidenceScore = 7
	}

	// Problem solving (keywords around challenges/solutions)
	problemKeywords := []string{"challenge", "problem", "debug", "fix", "issue", "optimize", "improve", "solution", "overcome", "resolved"}
	probHits := 0
	for _, msg := range candidateMessages {
		lower := strings.ToLower(msg)
		for _, pk := range problemKeywords {
			if strings.Contains(lower, pk) {
				probHits++
			}
		}
	}
	if probHits > 2 {
		result.ProblemSolvingScore = 8
	} else if probHits > 0 {
		result.ProblemSolvingScore = 7
	} else {
		result.ProblemSolvingScore = 5
	}

	// Clarity rating (question count as a proxy for engagement)
	result.FollowupsNeeded = len(aiMessages)
	if len(aiMessages) >= 5 && len(candidateMessages) >= 5 {
		// Multiple exchanges suggest engaged conversation
		result.ClarityRating = 7
	} else if len(aiMessages) >= 3 {
		result.ClarityRating = 6
	} else {
		result.ClarityRating = 5
	}

	// Candidate talk ratio
	if len(aiMessages) > 0 {
		totalMsgs := len(candidateMessages) + len(aiMessages)
		result.CandidateTalkRatio = float64(len(candidateMessages)) / float64(totalMsgs)
	}

	return result
}

func (ie *IncrementalEvaluator) parseObjectID(idStr string) (interface{}, error) {
	// Parse as MongoDB ObjectID from hex string
	return primitive.ObjectIDFromHex(idStr)
}
