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
		group.POST("/upload", controller.UploadAudio)
		group.POST("/chat", controller.Chat)
		group.POST("/finish", controller.FinishInterview)
		group.GET("/:id", controller.GetInterview)
	}
}
