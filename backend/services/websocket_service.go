package services

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WebSocketServer struct {
	chatService *ChatService
	clients     map[string]*Client
}

type Client struct {
	ID          string
	InterviewID primitive.ObjectID
	Send        chan interface{}
}

type SocketMessage struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Role      string `json:"role"`
	Sender    string `json:"sender"`
	Timestamp int64  `json:"timestamp"`
}

func NewWebSocketServer(interviewCollection, hrMemoryCollection, evaluationsCollection *mongo.Collection) *WebSocketServer {
	return &WebSocketServer{
		chatService: NewChatServiceWithEvaluations(interviewCollection, hrMemoryCollection, evaluationsCollection),
		clients:     make(map[string]*Client),
	}
}

func (ws *WebSocketServer) RegisterClient(clientID string, interviewID primitive.ObjectID) *Client {
	client := &Client{
		ID:          clientID,
		InterviewID: interviewID,
		Send:        make(chan interface{}, 256),
	}
	ws.clients[clientID] = client
	return client
}

func (ws *WebSocketServer) UnregisterClient(clientID string) {
	if client, ok := ws.clients[clientID]; ok {
		close(client.Send)
		delete(ws.clients, clientID)
	}
}

func (ws *WebSocketServer) HandleCandidateMessage(ctx context.Context, clientID string, content string) error {
	log.Printf("[WEBSOCKET] HandleCandidateMessage called for client %s with content: %s", clientID, content)
	client, ok := ws.clients[clientID]
	if !ok {
		log.Printf("[WEBSOCKET] ✗ Client %s not found in registered clients", clientID)
		return nil
	}

	log.Printf("[WEBSOCKET] Saving candidate message for interview %s", client.InterviewID.Hex())
	if err := ws.chatService.SaveMessage(ctx, client.InterviewID, "candidate", content); err != nil {
		log.Printf("Error saving candidate message: %v", err)
		return err
	}

	log.Printf("Checking dealbreaker for interview %s", client.InterviewID.Hex())
	isRejected, reason, err := ws.chatService.CheckDealbreaker(ctx, client.InterviewID, content)
	if err != nil {
		log.Printf("Error checking dealbreaker: %v", err)
	}

	if isRejected {
		log.Printf("Candidate rejected: %s", reason)
		if err := ws.chatService.MarkAsRejected(ctx, client.InterviewID, reason); err != nil {
			log.Printf("Error marking as rejected: %v", err)
		}
	}

	log.Printf("[WEBSOCKET] Processing message for interview %s", client.InterviewID.Hex())
	response, err := ws.chatService.ProcessMessage(ctx, client.InterviewID, content)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		return err
	}

	log.Printf("[WEBSOCKET] ✓ Got response of length %d, saving AI message for interview %s", len(response), client.InterviewID.Hex())
	if err := ws.chatService.SaveMessage(ctx, client.InterviewID, "ai", response); err != nil {
		log.Printf("Error saving AI message: %v", err)
		return err
	}

	aiMsg := SocketMessage{
		Type:      "ai_message",
		Content:   response,
		Role:      "ai",
		Sender:    "ai",
		Timestamp: 0,
	}

	log.Printf("[WEBSOCKET] Sending AI message to client %s via channel", clientID)
	client.Send <- aiMsg
	log.Printf("[WEBSOCKET] ✓ AI message sent to client %s channel", clientID)
	return nil
}

func (ws *WebSocketServer) BroadcastToClient(clientID string, msg interface{}) {
	if client, ok := ws.clients[clientID]; ok {
		select {
		case client.Send <- msg:
		default:
			log.Printf("Client %s message channel full, dropping message", clientID)
		}
	}
}

func (ws *WebSocketServer) GetClient(clientID string) *Client {
	return ws.clients[clientID]
}

func ConvertToObjectID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}
