package routes

import (
	"ai-recruiter/backend/controllers"
	"ai-recruiter/backend/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func SetupHRMemoryRoutes(router *gin.Engine, mongoClient *mongo.Client) {
	controller := controllers.NewHRMemoryController(mongoClient)

	public := router.Group("/hr-memory")
	{
		public.GET("/category/:category", controller.GetQuestionsByCategory)
	}

	protected := router.Group("/admin/hr-memory")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("", controller.CreateQuestion)
		protected.GET("", controller.GetAllQuestions)
		protected.PUT("/:id", controller.UpdateQuestion)
		protected.DELETE("/:id", controller.DeleteQuestion)
	}

	// Alias routes for older frontend endpoints: /admin/questions
	adminProtected := router.Group("/admin")
	adminProtected.Use(middleware.AuthMiddleware())
	{
		adminProtected.POST("/questions", controller.CreateQuestion)
		adminProtected.GET("/questions", controller.GetAllQuestions)
		adminProtected.PUT("/questions/:id", controller.UpdateQuestion)
		adminProtected.DELETE("/questions/:id", controller.DeleteQuestion)
	}
}
