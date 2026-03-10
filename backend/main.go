package main

import (
	"ai-recruiter/backend/config"
	"ai-recruiter/backend/routes"
	"ai-recruiter/backend/models"
	"ai-recruiter/backend/services"
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoClient, err := config.InitMongoDB()
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Start incremental evaluation worker (non-blocking)
	go startEvaluationWorker(mongoClient)

	// Development: optionally create a default admin from env vars
	if os.Getenv("DEV_CREATE_ADMIN") == "true" {
		adminID := os.Getenv("DEV_ADMIN_ID")
		adminPass := os.Getenv("DEV_ADMIN_PASS")
		adminName := os.Getenv("DEV_ADMIN_NAME")
		adminEmail := os.Getenv("DEV_ADMIN_EMAIL")
		if adminID != "" && adminPass != "" {
			coll := mongoClient.Database("ai_recruiter").Collection("admins")
			ctx := context.Background()
			var existing models.Admin
			err := coll.FindOne(ctx, map[string]interface{}{"admin_id": adminID}).Decode(&existing)
			if err != nil {
				// not found -> create
				_, err := coll.InsertOne(ctx, models.Admin{AdminID: adminID, Password: adminPass, Name: adminName, Email: adminEmail})
				if err != nil {
					log.Printf("Failed to create dev admin: %v", err)
				} else {
					log.Printf("Created dev admin '%s'", adminID)
				}
			} else {
				log.Printf("Dev admin '%s' already exists", adminID)
			}
		}
	}

	router := gin.Default()

	router.Use(corsMiddleware())

	routes.SetupInterviewRoutes(router, mongoClient)
	routes.SetupAdminRoutes(router, mongoClient)

	log.Printf("Server running on port %s", port)
	router.Run(":" + port)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func startEvaluationWorker(mongoClient *mongo.Client) {
	queue := services.NewEvalQueue()
	defer queue.Close()

	if !queue.IsAvailable() {
		log.Println("Redis not available; evaluation worker disabled")
		return
	}

	log.Println("Incremental evaluation worker started")

	evaluator := services.NewIncrementalEvaluator(mongoClient)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		job, err := queue.DequeueJob(ctx)
		cancel()

		if err != nil {
			log.Printf("Error dequeuing job: %v", err)
			continue
		}

		if job == nil {
			// Timeout, try again
			continue
		}

		log.Printf("Processing evaluation job for interview %s (attempt %d)", job.InterviewID, job.Attempt)

		evalCtx, evalCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := evaluator.EvaluateAndUpdate(evalCtx, job.InterviewID); err != nil {
			log.Printf("Evaluation error for %s: %v, requeuing...", job.InterviewID, err)
			if err := queue.RequeueJob(job); err != nil {
				log.Printf("Failed to requeue job: %v", err)
			}
		}
		evalCancel()
	}
}
