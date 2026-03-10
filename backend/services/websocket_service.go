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

func NewWebSocketServer(interviewCollection, hrMemoryCollection *mongo.Collection) *WebSocketServer {
	return &WebSocketServer{
		chatService: NewChatService(interviewCollection, hrMemoryCollection),
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
	client, ok := ws.clients[clientID]
	if !ok {
		return nil
	}

	if err := ws.chatService.SaveMessage(ctx, client.InterviewID, "candidate", content); err != nil {
		return err
	}

	isRejected, reason, err := ws.chatService.CheckDealbreaker(ctx, client.InterviewID, content)
	if err != nil {
		log.Printf("Error checking dealbreaker: %v", err)
	}

	if isRejected {
		if err := ws.chatService.MarkAsRejected(ctx, client.InterviewID, reason); err != nil {
			log.Printf("Error marking as rejected: %v", err)
		}
	}

	response, err := ws.chatService.ProcessMessage(ctx, client.InterviewID, content)
	if err != nil {
		return err
	}

	if err := ws.chatService.SaveMessage(ctx, client.InterviewID, "ai", response); err != nil {
		return err
	}

	aiMsg := SocketMessage{
		Type:      "ai_message",
		Content:   response,
		Role:      "ai",
		Sender:    "ai",
		Timestamp: 0,
	}

	client.Send <- aiMsg
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
