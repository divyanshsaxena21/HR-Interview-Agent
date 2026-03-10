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
	// Compute per-message scores and average them so empty answers count as zero
	candidateMessages := []string{}
	aiMessages := []string{}
	for _, msg := range transcript {
		if msg.Role == "candidate" {
			candidateMessages = append(candidateMessages, msg.Content)
		} else if msg.Role == "ai" {
			aiMessages = append(aiMessages, msg.Content)
		}
	}

	// Default result
	result := ScoreResult{
		CommunicationScore:  0,
		TechnicalScore:      0,
		ConfidenceScore:     0,
		ProblemSolvingScore: 0,
		AvgAnswerLength:     0,
		FollowupsNeeded:     len(aiMessages),
		ClarityRating:       5,
		CandidateTalkRatio:  0.0,
	}

	if len(candidateMessages) == 0 {
		// no candidate answers — return zeros as requested
		return result
	}

	// Per-message scoring
	techKeywords := []string{"go", "python", "java", "javascript", "react", "nodejs", "docker", "aws", "sql", "mongo", "typescript", "rust", "database", "api", "microservice"}
	hedgeKeywords := []string{"maybe", "perhaps", "might", "not sure", "i think", "could be", "sort of"}
	problemKeywords := []string{"challenge", "problem", "debug", "fix", "issue", "optimize", "improve", "solution", "overcome", "resolved"}

	commSum := 0
	techSum := 0
	confSum := 0
	probSum := 0
	totalWords := 0

	for _, msg := range candidateMessages {
		trimmed := strings.TrimSpace(msg)
		words := len(strings.Fields(trimmed))
		totalWords += words

		if trimmed == "" {
			// empty answer => zero for this question
			commSum += 0
			techSum += 0
			confSum += 0
			probSum += 0
			continue
		}

		// Communication: simple scale by words
		comm := 0
		if words >= 60 {
			comm = 9
		} else if words >= 40 {
			comm = 8
		} else if words >= 20 {
			comm = 6
		} else if words >= 10 {
			comm = 5
		} else if words > 0 {
			comm = 4
		}
		commSum += comm

		// Technical: keyword hits in this message
		techHits := 0
		lower := strings.ToLower(trimmed)
		for _, kw := range techKeywords {
			if strings.Contains(lower, kw) {
				techHits++
			}
		}
		techScore := 4
		if techHits > 0 {
			if techHits >= 3 {
				techScore = 9
			} else if techHits == 2 {
				techScore = 7
			} else {
				techScore = 6
			}
		}
		techSum += techScore

		// Confidence: hedging reduces score
		hedgeHits := 0
		for _, hw := range hedgeKeywords {
			if strings.Contains(lower, hw) {
				hedgeHits++
			}
		}
		confScore := 7
		if hedgeHits > 3 {
			confScore = 4
		} else if hedgeHits > 0 {
			confScore = 5
		} else if words > 30 {
			confScore = 8
		}
		confSum += confScore

		// Problem solving
		probHits := 0
		for _, pk := range problemKeywords {
			if strings.Contains(lower, pk) {
				probHits++
			}
		}
		probScore := 5
		if probHits > 2 {
			probScore = 8
		} else if probHits > 0 {
			probScore = 7
		}
		probSum += probScore
	}

	n := len(candidateMessages)
	// average the per-message scores
	result.CommunicationScore = commSum / n
	result.TechnicalScore = techSum / n
	result.ConfidenceScore = confSum / n
	result.ProblemSolvingScore = probSum / n
	result.AvgAnswerLength = 0
	if n > 0 {
		result.AvgAnswerLength = totalWords / n
	}

	// Clarity rating heuristics
	if len(aiMessages) >= 5 && len(candidateMessages) >= 5 {
		result.ClarityRating = 7
	} else if len(aiMessages) >= 3 {
		result.ClarityRating = 6
	} else {
		result.ClarityRating = 5
	}

	totalMsgs := len(candidateMessages) + len(aiMessages)
	if totalMsgs > 0 {
		result.CandidateTalkRatio = float64(len(candidateMessages)) / float64(totalMsgs)
	}

	result.FollowupsNeeded = len(aiMessages)

	return result
}

func (ie *IncrementalEvaluator) parseObjectID(idStr string) (interface{}, error) {
	// Parse as MongoDB ObjectID from hex string
	return primitive.ObjectIDFromHex(idStr)
}
