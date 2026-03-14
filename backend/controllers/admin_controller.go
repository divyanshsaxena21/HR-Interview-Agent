package controllers

import (
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/services"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type AdminController struct {
	db *mongo.Client
}

type InterviewWithEvaluation struct {
	ID              string            `json:"id"`
	CandidateName   string            `json:"candidate_name"`
	Email           string            `json:"email"`
	Role            string            `json:"role"`
	GitHub          string            `json:"github,omitempty"`
	LinkedIn        string            `json:"linkedin,omitempty"`
	Portfolio       string            `json:"portfolio,omitempty"`
	Documents       []models.Document `json:"documents,omitempty"`
	Status          string            `json:"status"`
	Rejected        bool              `json:"rejected"`
	RejectionReason string            `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Evaluation      *models.Evaluation `json:"evaluation,omitempty"`
	MessageCount    int               `json:"message_count"`
}

func NewAdminController(db *mongo.Client) *AdminController {
	return &AdminController{db: db}
}

func (ac *AdminController) Login(c *gin.Context) {
	var req models.AdminLoginRequest

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	collection := ac.db.Database("ai_recruiter").Collection("admins")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	err := collection.FindOne(ctx, bson.M{"admin_id": req.AdminID}).Decode(&admin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if admin.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"admin_id": admin.AdminID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
		"jti":      uuid.New().String(),
	})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key"
	}

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, models.AdminLoginResponse{
		Token: tokenString,
		Name:  admin.Name,
	})
}

func (ac *AdminController) GetAllInterviews(c *gin.Context) {
	collection := ac.db.Database("ai_recruiter").Collection("interviews")
	evaluationCollection := ac.db.Database("ai_recruiter").Collection("evaluations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	var interviews []models.Interview
	if err := cursor.All(ctx, &interviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fetch evaluations and build response
	var result []InterviewWithEvaluation
	for _, interview := range interviews {
		resp := InterviewWithEvaluation{
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
			UpdatedAt:       interview.UpdatedAt,
			MessageCount:    len(interview.Messages),
		}

		// Fetch evaluation if EvaluationID exists
		if interview.EvaluationID != primitive.NilObjectID {
			var evaluation models.Evaluation
			err := evaluationCollection.FindOne(ctx, bson.M{"_id": interview.EvaluationID}).Decode(&evaluation)
			if err == nil {
				resp.Evaluation = &evaluation
			}
		}

		result = append(result, resp)
	}

	c.JSON(http.StatusOK, result)
}

// ImportCandidates handles CSV or Excel file uploads
func (ac *AdminController) ImportCandidates(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	// Read file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	// Parse file based on extension
	importService := services.NewCandidateImportService(ac.db)
	var candidates []models.Candidate

	if file.Filename[len(file.Filename)-4:] == ".csv" {
		candidates, err = importService.ParseCSV(data)
	} else if file.Filename[len(file.Filename)-5:] == ".xlsx" {
		candidates, err = importService.ParseExcel(data)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be CSV or Excel"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse file: %v", err)})
		return
	}

	// Import candidates
	count, err := importService.ImportCandidates(candidates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to import candidates: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully imported %d candidates", count),
		"count":   count,
	})
}

// GetCandidates retrieves all candidates
func (ac *AdminController) GetCandidates(c *gin.Context) {
	importService := services.NewCandidateImportService(ac.db)
	candidates, err := importService.GetCandidates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, candidates)
}

// ScreenCandidate marks a candidate as screened and triggers scheduling
func (ac *AdminController) ScreenCandidate(c *gin.Context) {
	candidateID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(candidateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update candidate status
	coll := ac.db.Database("ai_recruiter").Collection("candidates")
	_, err = coll.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{"status": "screened"},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update candidate"})
		return
	}

	// Trigger scheduling agent
	llm := services.NewGroqLLMService()
	graph := services.NewAgentGraph()
	graph.AddAgent(services.NewSchedulingAgent(ac.db, llm))

	state := &services.AgentState{
		CandidateID: candidateID,
		Messages:    []map[string]interface{}{},
		Context:     make(map[string]interface{}),
	}

	_, err = graph.Execute(ctx, "SchedulingAgent", state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to schedule interview: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Candidate screened and interview scheduled",
		"session_id": state.SessionID,
	})
}

// DeleteCandidate removes a candidate
func (ac *AdminController) DeleteCandidate(c *gin.Context) {
	candidateID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(candidateID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid candidate ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete candidate
	coll := ac.db.Database("ai_recruiter").Collection("candidates")
	result, err := coll.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete candidate"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Candidate not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Candidate deleted successfully",
	})
}

// DeleteInterview removes an interview
func (ac *AdminController) DeleteInterview(c *gin.Context) {
	interviewID := c.Param("id")

	objID, err := primitive.ObjectIDFromHex(interviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete interview
	coll := ac.db.Database("ai_recruiter").Collection("interviews")
	result, err := coll.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete interview"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interview not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Interview deleted successfully",
	})
}
