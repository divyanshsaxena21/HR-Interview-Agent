package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HRMemory struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Category      string             `bson:"category" json:"category"`
	Question      string             `bson:"question" json:"question"`
	Tags          []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	IsDealbreaker bool               `bson:"is_dealbreaker" json:"is_dealbreaker"`
	Active        bool               `bson:"active" json:"active"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

type HRMemoryRequest struct {
	Category      string   `json:"category" binding:"required"`
	Question      string   `json:"question" binding:"required"`
	Tags          []string `json:"tags"`
	IsDealbreaker bool     `json:"is_dealbreaker"`
	Active        bool     `json:"active"`
}
