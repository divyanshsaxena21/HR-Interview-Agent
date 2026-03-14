package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Candidate represents a job candidate
type Candidate struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Name      string             `bson:"name" json:"name"`
	Email     string             `bson:"email" json:"email"`
	Phone     string             `bson:"phone,omitempty" json:"phone,omitempty"`
	Role      string             `bson:"role" json:"role"`
	GitHub    string             `bson:"github,omitempty" json:"github,omitempty"`
	LinkedIn  string             `bson:"linkedin,omitempty" json:"linkedin,omitempty"`
	Resume    string             `bson:"resume,omitempty" json:"resume,omitempty"`
	Status    string             `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Message struct {
	Role      string `bson:"role" json:"role"`
	Content   string `bson:"content" json:"content"`
	Timestamp int64  `bson:"timestamp" json:"timestamp"`
}

type Document struct {
	FileName    string `bson:"file_name" json:"file_name"`
	ContentType string `bson:"content_type" json:"content_type"`
	Data        string `bson:"data" json:"data"`
	UploadedAt  int64  `bson:"uploaded_at" json:"uploaded_at"`
}

// Interview represents a candidate interview session
type Interview struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	SessionID       string             `bson:"session_id" json:"session_id"`
	CandidateID     primitive.ObjectID `bson:"candidate_id" json:"candidate_id"`
	CandidateName   string             `bson:"candidate_name" json:"candidate_name"`
	Email           string             `bson:"email" json:"email"`
	Role            string             `bson:"role" json:"role"`
	GitHub          string             `bson:"github,omitempty" json:"github,omitempty"`
	LinkedIn        string             `bson:"linkedin,omitempty" json:"linkedin,omitempty"`
	Portfolio       string             `bson:"portfolio,omitempty" json:"portfolio,omitempty"`
	Messages        []Message          `bson:"messages" json:"messages"`
	Documents       []Document         `bson:"documents,omitempty" json:"documents,omitempty"`
	Status          string             `bson:"status" json:"status"`
	Rejected        bool               `bson:"rejected" json:"rejected"`
	RejectionReason string             `bson:"rejection_reason,omitempty" json:"rejection_reason,omitempty"`
	HRQuestionsAsked int               `bson:"hr_questions_asked,omitempty" json:"hr_questions_asked,omitempty"`
	ResultID        primitive.ObjectID `bson:"result_id,omitempty" json:"result_id,omitempty"`
	EvaluationID    primitive.ObjectID `bson:"evaluation_id,omitempty" json:"evaluation_id,omitempty"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

// InterviewResult contains structured evaluation results
type InterviewResult struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	InterviewID       primitive.ObjectID `bson:"interview_id" json:"interview_id"`
	CandidateID       primitive.ObjectID `bson:"candidate_id" json:"candidate_id"`
	CandidateName     string             `bson:"candidate_name" json:"candidate_name"`
	Role              string             `bson:"role" json:"role"`
	CommunicationScore int               `bson:"communication_score" json:"communication_score"`
	TechnicalScore    int                `bson:"technical_score" json:"technical_score"`
	ConfidenceScore   int                `bson:"confidence_score" json:"confidence_score"`
	Strengths         []string           `bson:"strengths" json:"strengths"`
	Weaknesses        []string           `bson:"weaknesses" json:"weaknesses"`
	Summary           string             `bson:"summary" json:"summary"`
	CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
}

// Evaluation (kept for backward compatibility, but InterviewResult is primary)
type Evaluation struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	InterviewID         primitive.ObjectID `bson:"interview_id"`
	CandidateName       string             `bson:"candidate_name" json:"candidate_name"`
	Role                string             `bson:"role" json:"role"`
	CommunicationScore  int                `bson:"communication_score" json:"communication_score"`
	TechnicalScore      int                `bson:"technical_score" json:"technical_score"`
	ConfidenceScore     int                `bson:"confidence_score" json:"confidence_score"`
	ProblemSolvingScore int                `bson:"problem_solving_score" json:"problem_solving_score"`
	Strengths           []string           `bson:"strengths" json:"strengths"`
	Weaknesses          []string           `bson:"weaknesses" json:"weaknesses"`
	Summary             string             `bson:"summary" json:"summary"`
	Fit                 string             `bson:"fit" json:"fit"`
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`
}

type Analytics struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	InterviewID        primitive.ObjectID `bson:"interview_id"`
	AvgAnswerLength    int                `bson:"avg_answer_length" json:"avg_answer_length"`
	FollowupsNeeded    int                `bson:"followups_needed" json:"followups_needed"`
	ClarityRating      int                `bson:"clarity_rating" json:"clarity_rating"`
	CandidateTalkRatio float64            `bson:"candidate_talk_ratio" json:"candidate_talk_ratio"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
}
