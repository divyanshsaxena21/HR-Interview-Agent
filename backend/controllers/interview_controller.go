package controllers

import (
	"ai-recruiter/backend/config"
	"ai-recruiter/backend/utils"
	"ai-recruiter/backend/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type InterviewController struct {
	db *mongo.Database
}

type StartInterviewRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}

type UpdateCandidateInfoRequest struct {
	GitHub    string `json:"github,omitempty"`
	LinkedIn  string `json:"linkedin,omitempty"`
	Portfolio string `json:"portfolio,omitempty"`
}

type GetInterviewResponse struct {
	ID            string           `json:"id"`
	CandidateName string           `json:"candidate_name"`
	Email         string           `json:"email"`
	Role          string           `json:"role"`
	GitHub        string           `json:"github,omitempty"`
	LinkedIn      string           `json:"linkedin,omitempty"`
	Portfolio     string           `json:"portfolio,omitempty"`
	Messages      []models.Message `json:"messages"`
	Status        string           `json:"status"`
	Rejected      bool             `json:"rejected"`
	CreatedAt     time.Time        `json:"created_at"`
}

func NewInterviewController(mongoClient *mongo.Client) *InterviewController {
	return &InterviewController{
		db: config.GetDatabase(mongoClient),
	}
}

func (ic *InterviewController) StartInterview(c *gin.Context) {
	var req StartInterviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interview := models.Interview{
		CandidateName: req.Name,
		Email:         req.Email,
		Role:          req.Role,
		Messages:      []models.Message{},
		Status:        "in_progress",
		Rejected:      false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	collection := ic.db.Collection("interviews")
	result, err := collection.InsertOne(context.Background(), interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	interviewID := result.InsertedID.(primitive.ObjectID).Hex()

	c.JSON(http.StatusOK, gin.H{
		"interview_id": interviewID,
		"message":      "Interview started. Start typing your responses in the chat.",
	})
}

func (ic *InterviewController) GetInterview(c *gin.Context) {
	interviewID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(interviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	var interview models.Interview
	err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interview not found"})
		return
	}

	response := GetInterviewResponse{
		ID:            interview.ID.Hex(),
		CandidateName: interview.CandidateName,
		Email:         interview.Email,
		Role:          interview.Role,
		GitHub:        interview.GitHub,
		LinkedIn:      interview.LinkedIn,
		Portfolio:     interview.Portfolio,
		Messages:      interview.Messages,
		Status:        interview.Status,
		Rejected:      interview.Rejected,
		CreatedAt:     interview.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
}

func (ic *InterviewController) UpdateCandidateInfo(c *gin.Context) {
	interviewID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(interviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	var req UpdateCandidateInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if req.GitHub != "" {
		update["$set"].(bson.M)["github"] = req.GitHub
	}
	if req.LinkedIn != "" {
		update["$set"].(bson.M)["linkedin"] = req.LinkedIn
	}
	if req.Portfolio != "" {
		update["$set"].(bson.M)["portfolio"] = req.Portfolio
	}

	collection := ic.db.Collection("interviews")
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Candidate info updated"})
}

func (ic *InterviewController) EndInterview(c *gin.Context) {
	interviewID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(interviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	_, err = collection.UpdateOne(context.Background(),
		bson.M{"_id": objID},
		bson.M{
			"$set": bson.M{
				"status":     "completed",
				"updated_at": time.Now(),
			},
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interview ended"})
}

func (ic *InterviewController) GetAllInterviews(c *gin.Context) {
	collection := ic.db.Collection("interviews")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(context.Background())

	var interviews []models.Interview
	if err = cursor.All(context.Background(), &interviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, interviews)
}

// StartAIQuestion generates the initial AI question for an interview, saves it, and returns it.
func (ic *InterviewController) StartAIQuestion(c *gin.Context) {
	interviewID := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(interviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	var interview models.Interview
	err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interview not found"})
		return
	}

	qs := utils.GetHRInterviewQuestions(interview.Role)
	if len(qs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No questions available"})
		return
	}

	first := qs[0]
	msg := models.Message{Role: "ai", Content: first, Timestamp: time.Now().Unix()}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objID}, bson.M{"$push": bson.M{"messages": msg}, "$set": bson.M{"updated_at": time.Now()}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"question": first})
}
