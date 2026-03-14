package controllers

import (
	"ai-recruiter/backend/config"
	"ai-recruiter/backend/services"
	"ai-recruiter/backend/utils"
	"ai-recruiter/backend/models"
	"context"
	"encoding/base64"
	"io"
	"log"
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

	collection := ic.db.Collection("interviews")
	var interview models.Interview
	
	// Try to find by session_id first (UUID from interview link)
	err := collection.FindOne(context.Background(), bson.M{"session_id": interviewID}).Decode(&interview)
	if err != nil {
		// If not found by session_id, try to find by MongoDB _id (ObjectID)
		objID, parseErr := primitive.ObjectIDFromHex(interviewID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
			return
		}
		err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&interview)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Interview not found"})
			return
		}
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
	
	// Try to find by session_id first, then by _id
	result, err := collection.UpdateOne(context.Background(), bson.M{"session_id": interviewID}, update)
	if err != nil || result.MatchedCount == 0 {
		// Try by ObjectID
		objID, parseErr := primitive.ObjectIDFromHex(interviewID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
			return
		}
		_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objID}, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Candidate info updated"})
}

func (ic *InterviewController) EndInterview(c *gin.Context) {
	interviewID := c.Param("id")

	collection := ic.db.Collection("interviews")
	ctx := context.Background()

	// Fetch the interview - try session_id first
	var interview models.Interview
	err := collection.FindOne(ctx, bson.M{"session_id": interviewID}).Decode(&interview)
	if err != nil {
		// Try by ObjectID
		objID, parseErr := primitive.ObjectIDFromHex(interviewID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
			return
		}
		err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&interview)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Interview not found"})
			return
		}
	}

	// Evaluate the interview
	evaluationService := services.NewEvaluationService()
	evaluation, err := evaluationService.EvaluateInterview(interview)
	if err != nil {
		log.Printf("[INTERVIEW] Error evaluating interview %s: %v", interviewID, err)
		// Continue even if evaluation fails
	} else if evaluation != nil {
		// Save evaluation to database
		evaluationCollection := ic.db.Collection("evaluations")
		result, err := evaluationCollection.InsertOne(ctx, evaluation)
		if err != nil {
			log.Printf("[INTERVIEW] Error saving evaluation for %s: %v", interviewID, err)
		} else {
			log.Printf("[INTERVIEW] ✓ Created evaluation %s for interview %s", result.InsertedID, interviewID)
			// Update interview with evaluation ID
			_, err = collection.UpdateOne(ctx,
				bson.M{"_id": interview.ID},
				bson.M{
					"$set": bson.M{
						"status":          "completed",
						"evaluation_id":   result.InsertedID,
						"updated_at":      time.Now(),
					},
				},
			)
			if err != nil {
				log.Printf("[INTERVIEW] Error updating interview with evaluation ID: %v", err)
			}
			c.JSON(http.StatusOK, gin.H{"message": "Interview ended", "evaluation_id": result.InsertedID})
			return
		}
	}

	// Fallback: if evaluation fails, just mark as completed
	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": interview.ID},
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

	// Build response with message count
	type InterviewResponse struct {
		ID              string             `json:"id"`
		CandidateName   string             `json:"candidate_name"`
		Email           string             `json:"email"`
		Role            string             `json:"role"`
		GitHub          string             `json:"github,omitempty"`
		LinkedIn        string             `json:"linkedin,omitempty"`
		Portfolio       string             `json:"portfolio,omitempty"`
		Documents       []models.Document  `json:"documents,omitempty"`
		Status          string             `json:"status"`
		Rejected        bool               `json:"rejected"`
		RejectionReason string             `json:"rejection_reason,omitempty"`
		CreatedAt       time.Time          `json:"created_at"`
		MessageCount    int                `json:"message_count"`
		EvaluationID    string             `json:"evaluation_id,omitempty"`
	}

	responses := make([]InterviewResponse, len(interviews))
	for i, interview := range interviews {
		if len(interview.Documents) > 0 {
			log.Printf("[ADMIN] Interview %s has %d documents", interview.ID.Hex(), len(interview.Documents))
		}
		responses[i] = InterviewResponse{
			ID:              interview.ID.Hex(),
			CandidateName:   interview.CandidateName,
			Email:           interview.Email,
			Role:            interview.Role,
			GitHub:          interview.GitHub,
			LinkedIn:        interview.LinkedIn,
			Portfolio:       interview.Portfolio,
			Documents:       interview.Documents,
			Status:          interview.Status,
			Rejected:        interview.Rejected,
			RejectionReason: interview.RejectionReason,
			CreatedAt:       interview.CreatedAt,
			MessageCount:    len(interview.Messages),
			EvaluationID:    interview.EvaluationID.Hex(),
		}
	}

	log.Printf("[ADMIN] Returning %d interviews to dashboard", len(responses))
	c.JSON(http.StatusOK, responses)
}

// StartAIQuestion generates the initial AI question for an interview, saves it, and returns it.
func (ic *InterviewController) StartAIQuestion(c *gin.Context) {
	interviewID := c.Param("id")
	
	collection := ic.db.Collection("interviews")
	var interview models.Interview
	
	// Try session_id first
	err := collection.FindOne(context.Background(), bson.M{"session_id": interviewID}).Decode(&interview)
	if err != nil {
		// Try by ObjectID
		objID, parseErr := primitive.ObjectIDFromHex(interviewID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
			return
		}
		err = collection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&interview)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Interview not found"})
			return
		}
	}

	qs := utils.GetHRInterviewQuestions(interview.Role)
	if len(qs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No questions available"})
		return
	}

	first := qs[0]
	msg := models.Message{Role: "ai", Content: first, Timestamp: time.Now().Unix()}

	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": interview.ID}, bson.M{
		"$push": bson.M{"messages": msg},
		"$set": bson.M{"updated_at": time.Now()},
		"$inc": bson.M{"hr_questions_asked": 1},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"question": first})
}

// UploadDocument uploads a document for an interview and stores it in MongoDB
func (ic *InterviewController) UploadDocument(c *gin.Context) {
	interviewID := c.Param("id")

	// Get file from form data
	header, err := c.FormFile("document")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Open the file
	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Read file content
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Encode to base64
	encodedData := base64.StdEncoding.EncodeToString(fileData)

	// Create document object
	doc := models.Document{
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Data:        encodedData,
		UploadedAt:  time.Now().Unix(),
	}

	// Add to interview documents array - try session_id first
	collection := ic.db.Collection("interviews")
	result, err := collection.UpdateOne(
		context.Background(),
		bson.M{"session_id": interviewID},
		bson.M{
			"$push": bson.M{"documents": doc},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	
	// If not found by session_id, try by ObjectID
	if err != nil || result.MatchedCount == 0 {
		objID, parseErr := primitive.ObjectIDFromHex(interviewID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
			return
		}
		_, err = collection.UpdateOne(
			context.Background(),
			bson.M{"_id": objID},
			bson.M{
				"$push": bson.M{"documents": doc},
				"$set":  bson.M{"updated_at": time.Now()},
			},
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload document"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Document uploaded successfully",
		"file":    header.Filename,
	})
}
