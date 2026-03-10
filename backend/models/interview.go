package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Candidate struct {
	ID    primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name  string             `bson:"name" json:"name"`
	Email string             `bson:"email" json:"email"`
	Role  string             `bson:"role" json:"role"`
}

type Interview struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	CandidateID     primitive.ObjectID `bson:"candidate_id" json:"candidate_id"`
	CandidateName   string             `bson:"candidate_name" json:"candidate_name"`
	Role            string             `bson:"role" json:"role"`
	AudioURL        string             `bson:"audio_url,omitempty" json:"audio_url,omitempty"`
	Transcript      []Message          `bson:"transcript" json:"transcript"`
	Status          string             `bson:"status" json:"status"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
	EvaluationID    primitive.ObjectID `bson:"evaluation_id,omitempty" json:"evaluation_id,omitempty"`
	AnalyticsID     primitive.ObjectID `bson:"analytics_id,omitempty" json:"analytics_id,omitempty"`
}

type Message struct {
	Role      string `bson:"role" json:"role"`
	Content   string `bson:"content" json:"content"`
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
}

type Evaluation struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	InterviewID          primitive.ObjectID `bson:"interview_id"`
	CandidateName        string             `bson:"candidate_name" json:"candidate_name"`
	Role                 string             `bson:"role" json:"role"`
	CommunicationScore   int                `bson:"communication_score" json:"communication_score"`
	TechnicalScore       int                `bson:"technical_score" json:"technical_score"`
	ConfidenceScore      int                `bson:"confidence_score" json:"confidence_score"`
	ProblemSolvingScore  int                `bson:"problem_solving_score" json:"problem_solving_score"`
	Strengths            []string           `bson:"strengths" json:"strengths"`
	Weaknesses           []string           `bson:"weaknesses" json:"weaknesses"`
	Summary              string             `bson:"summary" json:"summary"`
	Fit                  string             `bson:"fit" json:"fit"`
	CreatedAt            time.Time          `bson:"created_at" json:"created_at"`
}

type Analytics struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	InterviewID      primitive.ObjectID `bson:"interview_id"`
	AvgAnswerLength  int                `bson:"avg_answer_length" json:"avg_answer_length"`
	FollowupsNeeded  int                `bson:"followups_needed" json:"followups_needed"`
	ClarityRating    int                `bson:"clarity_rating" json:"clarity_rating"`
	CandidateTalkRatio float64          `bson:"candidate_talk_ratio" json:"candidate_talk_ratio"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
}
