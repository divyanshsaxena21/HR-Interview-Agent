package controllers

import (
	"ai-recruiter/backend/config"
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/services"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type HRMemoryController struct {
	service *services.HRMemoryService
}

func NewHRMemoryController(mongoClient *mongo.Client) *HRMemoryController {
	db := config.GetDatabase(mongoClient)
	collection := db.Collection("hr_memory")
	return &HRMemoryController{
		service: services.NewHRMemoryService(collection),
	}
}

func (hc *HRMemoryController) CreateQuestion(c *gin.Context) {
	var req models.HRMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	question, err := hc.service.CreateQuestion(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, question)
}

func (hc *HRMemoryController) GetAllQuestions(c *gin.Context) {
	admin := c.MustGet("admin")
	if admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	questions, err := hc.service.GetAllQuestionsAdmin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, questions)
}

func (hc *HRMemoryController) GetQuestionsByCategory(c *gin.Context) {
	category := c.Param("category")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	questions, err := hc.service.GetQuestionsByCategory(ctx, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, questions)
}

func (hc *HRMemoryController) UpdateQuestion(c *gin.Context) {
	admin := c.MustGet("admin")
	if admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID"})
		return
	}

	var req models.HRMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	question, err := hc.service.UpdateQuestion(ctx, objID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, question)
}

func (hc *HRMemoryController) DeleteQuestion(c *gin.Context) {
	admin := c.MustGet("admin")
	if admin == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = hc.service.DeleteQuestion(ctx, objID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted"})
}
