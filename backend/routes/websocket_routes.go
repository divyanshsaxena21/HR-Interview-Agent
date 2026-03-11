package routes

import (
	"ai-recruiter/backend/services"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func SetupWebSocketRoutes(router *gin.Engine, mongoClient *mongo.Client) {
	db := mongoClient.Database("ai_recruiter")
	interviewCollection := db.Collection("interviews")
	hrMemoryCollection := db.Collection("hr_memory")
	evaluationsCollection := db.Collection("evaluations")
	chatService := services.NewChatServiceWithEvaluations(interviewCollection, hrMemoryCollection, evaluationsCollection)

	router.GET("/ws/:interviewId/:candidateName", func(c *gin.Context) {
		interviewId := c.Param("interviewId")
		candidateName := c.Param("candidateName")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WS] ✗ Upgrade error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
			return
		}
		defer conn.Close()

		log.Printf("[WS] ✓ Client connected: interviewId=%s, candidateName=%s", interviewId, candidateName)

		// Handle messages from client
		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Printf("[WS] Client disconnected: %v", err)
				break
			}

			msgType, ok := msg["type"].(string)
			if !ok || msgType == "" {
				log.Printf("[WS] ✗ Invalid message: no type")
				continue
			}

			log.Printf("[WS] ✓ Received %s from client", msgType)

			switch msgType {
			case "candidate_message":
				content, ok := msg["content"].(string)
				if !ok || content == "" {
					log.Printf("[WS] ✗ Invalid message content")
					continue
				}

				log.Printf("[WS] Processing message: %s", content)

				// Convert interviewId and process message
				objID, err := services.ConvertToObjectID(interviewId)
				if err != nil {
					log.Printf("[WS] ✗ Invalid interview ID: %v", err)
					conn.WriteJSON(map[string]interface{}{
						"type":    "error",
						"content": "Invalid interview ID",
					})
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Save candidate message
				if err := chatService.SaveMessage(ctx, objID, "candidate", content); err != nil {
					log.Printf("[WS] ✗ Error saving message: %v", err)
					conn.WriteJSON(map[string]interface{}{
						"type":    "error",
						"content": "Error saving message",
					})
					continue
				}

				// Extract and save profile links from candidate message
				if err := chatService.ExtractAndSaveProfileLinks(ctx, objID, content); err != nil {
					log.Printf("[WS] Error extracting profile links: %v", err)
				}

				// Check dealbreaker
				isRejected, reason, err := chatService.CheckDealbreaker(ctx, objID, content)
				if err != nil {
					log.Printf("[WS] Error checking dealbreaker: %v", err)
				}

				if isRejected {
					log.Printf("[WS] Candidate rejected: %s", reason)
					if err := chatService.MarkAsRejected(ctx, objID, reason); err != nil {
						log.Printf("[WS] Error marking as rejected: %v", err)
					}
				}

				// Generate AI response
				response, err := chatService.ProcessMessage(ctx, objID, content)
				if err != nil {
					log.Printf("[WS] ✗ Error generating response: %v", err)
					conn.WriteJSON(map[string]interface{}{
						"type":    "error",
						"content": "Error processing message",
					})
					continue
				}

				// Save AI message
				if err := chatService.SaveMessage(ctx, objID, "ai", response); err != nil {
					log.Printf("[WS] ✗ Error saving AI message: %v", err)
				}

				// Track which HR questions have been asked
				if err := chatService.TrackAskedQuestions(ctx, objID); err != nil {
					log.Printf("[WS] Error tracking asked questions: %v", err)
				}

				log.Printf("[WS] ✓ Sending response of length %d", len(response))

				// Send response to client
				err = conn.WriteJSON(map[string]interface{}{
					"type":    "ai_message",
					"content": response,
				})
				if err != nil {
					log.Printf("[WS] ✗ Error sending response: %v", err)
					break
				}
			}
		}
	})
}
