package routes

import (
	"ai-recruiter/backend/controllers"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupInterviewRoutes(router *gin.Engine, mongoClient *mongo.Client) {
	controller := controllers.NewInterviewController(mongoClient)

	group := router.Group("/interview")
	{
		group.POST("/start", controller.StartInterview)
		group.POST("/:id/ai-start", controller.StartAIQuestion)
		group.GET("/:id", controller.GetInterview)
		group.PUT("/:id/info", controller.UpdateCandidateInfo)
		group.POST("/:id/end", controller.EndInterview)
	}
}
