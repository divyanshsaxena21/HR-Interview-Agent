package services

import (
	"ai-recruiter/backend/models"
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsService struct{}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

func (as *AnalyticsService) ComputeAnalytics(interview models.Interview) (*models.Analytics, error) {
	avgAnswerLength := as.computeAvgAnswerLength(interview.Transcript)
	followupsNeeded := as.computeFollowups(interview.Transcript)
	clarityRating := as.computeClarityRating(interview.Transcript)
	candidateTalkRatio := as.computeCandidateTalkRatio(interview.Transcript)

	analytics := &models.Analytics{
		ID:               primitive.NewObjectID(),
		InterviewID:      interview.ID,
		AvgAnswerLength:  avgAnswerLength,
		FollowupsNeeded:  followupsNeeded,
		ClarityRating:    clarityRating,
		CandidateTalkRatio: candidateTalkRatio,
		CreatedAt:        time.Now(),
	}

	return analytics, nil
}

func (as *AnalyticsService) computeAvgAnswerLength(transcript []models.Message) int {
	totalLength := 0
	candidateAnswers := 0

	for _, msg := range transcript {
		if msg.Role == "candidate" {
			totalLength += len(msg.Content)
			candidateAnswers++
		}
	}

	if candidateAnswers == 0 {
		return 0
	}

	return totalLength / candidateAnswers
}

func (as *AnalyticsService) computeFollowups(transcript []models.Message) int {
	followups := 0
	for i := 0; i < len(transcript)-1; i++ {
		if transcript[i].Role == "candidate" && transcript[i+1].Role == "ai" {
			if len(transcript[i].Content) < 100 {
				followups++
			}
		}
	}
	return followups
}

func (as *AnalyticsService) computeClarityRating(transcript []models.Message) int {
	// Simplified clarity rating based on answer lengths
	avgLength := as.computeAvgAnswerLength(transcript)
	if avgLength > 150 {
		return 9
	} else if avgLength > 100 {
		return 8
	} else if avgLength > 50 {
		return 6
	}
	return 4
}

func (as *AnalyticsService) computeCandidateTalkRatio(transcript []models.Message) float64 {
	candidateWords := 0
	totalWords := 0

	for _, msg := range transcript {
		words := len(msg.Content)
		totalWords += words
		if msg.Role == "candidate" {
			candidateWords += words
		}
	}

	if totalWords == 0 {
		return 0
	}

	return float64(candidateWords) / float64(totalWords)
}
