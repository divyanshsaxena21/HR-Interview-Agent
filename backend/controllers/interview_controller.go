package controllers

import (
	"ai-recruiter/backend/config"
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/services"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InterviewController struct {
	db *mongo.Database
}

type StartInterviewRequest struct {
	Name string `json:"name"`
	Email string `json:"email"`
	Role string `json:"role"`
}

type ChatRequest struct {
	InterviewID string `json:"interview_id"`
	Transcript  string `json:"transcript"`
}

type ChatResponse struct {
	InterviewID string `json:"interview_id"`
	Question    string `json:"question"`
	AudioURL    string `json:"audio_url"`
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

	agent := services.NewLangChainAgent()
	question, err := agent.GenerateInitialQuestion(req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create interview with initial question in transcript
	interview := models.Interview{
		CandidateName: req.Name,
		Role:          req.Role,
		Transcript: []models.Message{
			{
				Role:      "ai",
				Content:   question,
				Timestamp: time.Now().Unix(),
			},
		},
		Status:    "in_progress",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	collection := ic.db.Collection("interviews")
	result, err := collection.InsertOne(context.Background(), interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	interviewID := result.InsertedID.(primitive.ObjectID).Hex()

	ttsService := services.NewMurfTTSService()
	audioURL := ""
	if audio, err := ttsService.TextToSpeech(question); err != nil {
		log.Println("TTS error (StartInterview):", err)
	} else {
		audioURL = audio
	}

	// persist the audio URL for the initial question so the frontend can load it on /interview/:id
	if audioURL != "" {
		_, _ = collection.UpdateOne(context.Background(), bson.M{"_id": result.InsertedID}, bson.M{
			"$set": bson.M{"audio_url": audioURL, "updated_at": time.Now()},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"interview_id": interviewID,
		"question":     question,
		"audio_url":    audioURL,
	})
}

func (ic *InterviewController) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interviewID, err := primitive.ObjectIDFromHex(req.InterviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	update := bson.M{
		"$push": bson.M{
			"transcript": bson.M{
				"role":      "candidate",
				"content":   req.Transcript,
				"timestamp": time.Now().Unix(),
			},
		},
	}

	interview := models.Interview{}
	err = collection.FindOneAndUpdate(context.Background(), bson.M{"_id": interviewID}, update, opts).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	agent := services.NewLangChainAgent()
	question, err := agent.GenerateQuestion(interview.Transcript, interview.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if interview should END
	if question == "END_INTERVIEW" {
		// Add thank you message to transcript
		thankYouMsg := "Thank you so much for your time today! We really appreciate you taking the time to speak with us. We'll review your responses and get back to you with our decision within the next few days. Best of luck!"
		
		_, err := collection.UpdateOne(context.Background(), bson.M{"_id": interviewID}, bson.M{
			"$push": bson.M{
				"transcript": bson.M{
					"role":      "ai",
					"content":   thankYouMsg,
					"timestamp": time.Now().Unix(),
				},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Generate evaluation and analytics
		evalService := services.NewEvaluationService()
		evaluation, err := evalService.EvaluateInterview(interview)
		if err != nil {
			log.Printf("Evaluation error: %v", err)
			// on error, create a minimal placeholder evaluation so downstream logic can continue
			evaluation = &models.Evaluation{
				InterviewID: interviewID,
				CreatedAt:   time.Now(),
			}
		}

		var evalResult *mongo.InsertOneResult
		if evaluation != nil {
			evalCollection := ic.db.Collection("evaluations")
			er, err := evalCollection.InsertOne(context.Background(), evaluation)
			if err != nil {
				log.Printf("Eval insert error: %v", err)
			} else {
				evalResult = er
			}
		}

		analyticsService := services.NewAnalyticsService()
		analytics, err := analyticsService.ComputeAnalytics(interview)
		if err != nil {
			log.Printf("Analytics error: %v", err)
			analytics = &models.Analytics{
				InterviewID: interviewID,
				CreatedAt:   time.Now(),
			}
		}

		var analyticsResult *mongo.InsertOneResult
		analyticsCollection := ic.db.Collection("analytics")
		ar, err := analyticsCollection.InsertOne(context.Background(), analytics)
		if err != nil {
			log.Printf("Analytics insert error: %v", err)
		} else {
			analyticsResult = ar
		}

		// Build update document conditionally including evaluation/analytics ids
		updateDoc := bson.M{"status": "completed", "updated_at": time.Now()}
		if evalResult != nil {
			updateDoc["evaluation_id"] = evalResult.InsertedID
		}
		if analyticsResult != nil {
			updateDoc["analytics_id"] = analyticsResult.InsertedID
		}

		_, err = collection.UpdateOne(context.Background(), bson.M{"_id": interviewID}, bson.M{"$set": updateDoc})
		if err != nil {
			log.Printf("Interview update error: %v", err)
		}

		ttsService := services.NewMurfTTSService()
		audioURL := ""
		if audio, err := ttsService.TextToSpeech(thankYouMsg); err != nil {
			log.Println("TTS error (end interview):", err)
		} else {
			audioURL = audio
		}

		c.JSON(http.StatusOK, gin.H{
			"finished":    true,
			"reason":      "interview_complete",
			"message":     thankYouMsg,
			"audio_url":   audioURL,
			"evaluation":  evaluation,
			"analytics":   analytics,
		})
		return
	}

	// Add AI question to transcript and get updated interview
	opts2 := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = collection.FindOneAndUpdate(context.Background(), bson.M{"_id": interviewID}, bson.M{
		"$push": bson.M{
			"transcript": bson.M{
				"role":      "ai",
				"content":   question,
				"timestamp": time.Now().Unix(),
			},
		},
	}, opts2).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ttsService := services.NewMurfTTSService()
	audioURL := ""
	if audio, err := ttsService.TextToSpeech(question); err != nil {
		log.Println("TTS error (Chat):", err)
	} else {
		audioURL = audio
	}

	// persist the audio URL so the frontend can reload the page and still have the latest audio
	if audioURL != "" {
		_, _ = collection.UpdateOne(context.Background(), bson.M{"_id": interviewID}, bson.M{
			"$set": bson.M{"audio_url": audioURL, "updated_at": time.Now()},
		})
	}

	// Enqueue incremental evaluation job (non-blocking)
	go func() {
		queue := services.NewEvalQueue()
		defer queue.Close()
		if queue.IsAvailable() {
			if err := queue.EnqueueJob(req.InterviewID); err != nil {
				log.Printf("Failed to enqueue eval job for %s: %v", req.InterviewID, err)
			}
		}
	}()

	c.JSON(http.StatusOK, ChatResponse{
		InterviewID: req.InterviewID,
		Question:    question,
		AudioURL:    audioURL,
	})
}

// UploadAudio accepts a multipart file 'audio', uploads it to AssemblyAI,
// runs transcription, and returns the recognized text and the assembly upload URL.
func (ic *InterviewController) UploadAudio(c *gin.Context) {
	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio file required (field: audio)"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()

	buf := make([]byte, file.Size)
	n, err := f.Read(buf)
	if err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	buf = buf[:n]

	as := services.NewAssemblyAISTTService()
	uploadURL, err := as.UploadBytes(buf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Now request transcription
	text, err := as.SpeechToText(uploadURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transcript": text,
		"audio_url":  uploadURL,
	})
}

func (ic *InterviewController) FinishInterview(c *gin.Context) {
	var req struct {
		InterviewID string `json:"interview_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interviewID, err := primitive.ObjectIDFromHex(req.InterviewID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	interview := models.Interview{}
	err = collection.FindOne(context.Background(), bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	evalService := services.NewEvaluationService()
	evaluation, err := evalService.EvaluateInterview(interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var evalResult *mongo.InsertOneResult
	if evaluation != nil {
		evalCollection := ic.db.Collection("evaluations")
		er, err := evalCollection.InsertOne(context.Background(), evaluation)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		evalResult = er
	}

	analyticsService := services.NewAnalyticsService()
	analytics, err := analyticsService.ComputeAnalytics(interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	analyticsCollection := ic.db.Collection("analytics")
	analyticsResult, err := analyticsCollection.InsertOne(context.Background(), analytics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updateDoc := bson.M{"status": "completed", "updated_at": time.Now()}
	if evalResult != nil {
		updateDoc["evaluation_id"] = evalResult.InsertedID
	}
	if analyticsResult != nil {
		updateDoc["analytics_id"] = analyticsResult.InsertedID
	}
	collection.UpdateOne(context.Background(), bson.M{"_id": interviewID}, bson.M{"$set": updateDoc})

	c.JSON(http.StatusOK, gin.H{
		"interview_id": req.InterviewID,
		"evaluation":   evaluation,
		"analytics":    analytics,
	})
}

func (ic *InterviewController) GetInterview(c *gin.Context) {
	id := c.Param("id")
	interviewID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interview ID"})
		return
	}

	collection := ic.db.Collection("interviews")
	interview := models.Interview{}
	err = collection.FindOne(context.Background(), bson.M{"_id": interviewID}).Decode(&interview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var evaluation models.Evaluation
	var analytics models.Analytics
	var currentQuestion string

	// Determine if requester is an admin by validating JWT in Authorization header
	isAdmin := false
	tokenString := c.GetHeader("Authorization")
	if tokenString != "" {
		// strip Bearer prefix
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "your-secret-key"
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err == nil && token.Valid {
			isAdmin = true
		}
	}

	// Get evaluation and analytics only for admins when interview is completed
	if interview.Status == "completed" && isAdmin {
		evalCollection := ic.db.Collection("evaluations")
		evalCollection.FindOne(context.Background(), bson.M{"_id": interview.EvaluationID}).Decode(&evaluation)

		analyticsCollection := ic.db.Collection("analytics")
		analyticsCollection.FindOne(context.Background(), bson.M{"_id": interview.AnalyticsID}).Decode(&analytics)
	} else {
		// For in-progress interviews (or non-admin view), get the last AI question
		for i := len(interview.Transcript) - 1; i >= 0; i-- {
			if interview.Transcript[i].Role == "ai" {
				currentQuestion = interview.Transcript[i].Content
				break
			}
		}
	}

	// Only include evaluation/analytics when admin requested them
	resp := gin.H{
		"interview":       interview,
		"currentQuestion": currentQuestion,
	}
	if isAdmin {
		resp["evaluation"] = evaluation
		resp["analytics"] = analytics
	}

	c.JSON(http.StatusOK, resp)
}

func (ic *InterviewController) GetAllInterviews(c *gin.Context) {
	// Determine if requester is an admin by validating JWT in Authorization header
	isAdmin := false
	tokenString := c.GetHeader("Authorization")
	if tokenString != "" {
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "your-secret-key"
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err == nil && token.Valid {
			isAdmin = true
		}
	}

	collection := ic.db.Collection("interviews")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var interviews []models.Interview
	if err = cursor.All(context.Background(), &interviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build response entries; include evaluation/analytics when admin
	resp := []gin.H{}
	for _, iv := range interviews {
		// Flatten interview fields at top-level so frontend can consume directly
		item := gin.H{
			"_id":            iv.ID,
			"candidate_name": iv.CandidateName,
			"role":           iv.Role,
			"status":         iv.Status,
			"transcript":     iv.Transcript,
		}

		if isAdmin && iv.Status == "completed" {
			var evaluation models.Evaluation
			var analytics models.Analytics
			if iv.EvaluationID != primitive.NilObjectID {
				evalCollection := ic.db.Collection("evaluations")
				_ = evalCollection.FindOne(context.Background(), bson.M{"_id": iv.EvaluationID}).Decode(&evaluation)
				item["evaluation"] = evaluation
			}
			if iv.AnalyticsID != primitive.NilObjectID {
				analyticsCollection := ic.db.Collection("analytics")
				_ = analyticsCollection.FindOne(context.Background(), bson.M{"_id": iv.AnalyticsID}).Decode(&analytics)
				item["analytics"] = analytics
			}
		}

		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, resp)
}
