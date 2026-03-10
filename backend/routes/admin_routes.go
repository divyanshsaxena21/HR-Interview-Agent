package routes

import (
	"ai-recruiter/backend/controllers"
	"ai-recruiter/backend/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupAdminRoutes(router *gin.Engine, mongoClient *mongo.Client) {
	adminController := controllers.NewAdminController(mongoClient)

	router.POST("/admin/login", adminController.Login)

	protectedGroup := router.Group("/admin")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		protectedGroup.GET("/interviews", adminController.GetAllInterviews)
	}
}
