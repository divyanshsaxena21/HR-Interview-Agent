package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type CallService struct {
	authID     string
	authToken  string
	fromNumber string
	answerURL  string
}

// NewCallService creates a CallService configured for Plivo.
// Expected environment variables:
//   PLIVO_AUTH_ID      – Plivo Auth ID
//   PLIVO_AUTH_TOKEN   – Plivo Auth Token
//   PLIVO_FROM_NUMBER  – The caller ID number registered with Plivo
//   PLIVO_ANSWER_URL   – TwiML/XML URL for Plivo call flow
func NewCallService() *CallService {
	authID := os.Getenv("PLIVO_AUTH_ID")
	authToken := os.Getenv("PLIVO_AUTH_TOKEN")
	from := os.Getenv("PLIVO_FROM_NUMBER")
	answerURL := os.Getenv("PLIVO_ANSWER_URL")
	return &CallService{authID: authID, authToken: authToken, fromNumber: from, answerURL: answerURL}
}

type PlivoCallRequest struct {
	From      string `json:"from"`
	To        string `json:"to"`
	AnswerURL string `json:"answer_url"`
}

// CallCandidate attempts to place a voice call to the given contact (phone number).
// Retries up to 3 times with a 15‑second interval.
// Returns true if any attempt receives a 201/200 response.
func (cs *CallService) CallCandidate(contact string) (bool, error) {
	if cs.authID == "" || cs.authToken == "" || cs.fromNumber == "" {
		log.Printf("[CALL] Plivo credentials not configured, skipping call to %s", contact)
		return false, nil
	}

	// Plivo call creation endpoint
	endpoint := fmt.Sprintf("https://api.plivo.com/v1/Account/%s/Call/", cs.authID)
	
	// Ensure phone numbers are in E.164 format (start with '+')
	formatNumber := func(num string) string {
		if strings.HasPrefix(num, "+") {
			return num
		}
		return "+" + strings.TrimSpace(num)
	}

	targetURL := cs.answerURL
	if targetURL == "" {
		targetURL = "http://twimlets.com/echo?Twiml=%3CResponse%3E%3CSay%3EHello%2C+this+is+your+HR+interview+call.%3C%2FSay%3E%3C%2FResponse%3E"
	}

	payloadObj := PlivoCallRequest{
		From:      formatNumber(cs.fromNumber),
		To:        formatNumber(contact),
		AnswerURL: targetURL,
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return false, err
	}

	log.Printf("[CALL] Plivo request to %s with payload: %s", endpoint, string(payloadBytes))

	client := &http.Client{Timeout: 20 * time.Second}
	attempts := 3
	interval := 15 * time.Second
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
		if err != nil {
			return false, err
		}
		
		req.SetBasicAuth(cs.authID, cs.authToken)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[CALL] Attempt %d failed: %v", i+1, err)
		} else {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			
			if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
				log.Printf("[CALL] Call placed to %s on attempt %d, status %d, response: %s", contact, i+1, resp.StatusCode, string(bodyBytes))
				return true, nil
			}
			log.Printf("[CALL] Attempt %d got status %d, response: %s", i+1, resp.StatusCode, string(bodyBytes))
		}
		if i < attempts-1 {
			time.Sleep(interval)
		}
	}
	return false, nil
}