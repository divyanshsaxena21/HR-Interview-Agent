package routes

import (
	"ai-recruiter/backend/controllers"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupAdminRoutes(router *gin.Engine, mongoClient *mongo.Client) {
	adminController := controllers.NewAdminController(mongoClient)

	// Public admin login route
	router.POST("/admin/login", adminController.Login)

	// Protected admin routes (all interviews endpoints)
	// Mount under /admin so frontend can call /admin/interviews
	protectedGroup := router.Group("/admin")
	protectedGroup.Use(adminController.AuthMiddleware())
	protectedGroup.GET("/interviews", func(c *gin.Context) {
		// This will use the existing interview controller
		interviewController := controllers.NewInterviewController(mongoClient)
		interviewController.GetAllInterviews(c)
	})
}
